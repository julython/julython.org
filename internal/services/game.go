package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/db"
)

var (
	ErrNoActiveGame = errors.New("no active game found")
	ErrInvalidDates = errors.New("game start must be before end")
)

type GameService struct {
	queries *db.Queries
}

func NewGameService(q *db.Queries) *GameService {
	return &GameService{queries: q}
}

// CreateGame creates a new game with the given parameters.
func (s *GameService) CreateGame(
	ctx context.Context,
	name string,
	startsAt, endsAt time.Time,
	commitPoints, projectPoints int32,
	isActive, deactivateOthers bool,
) (db.Game, error) {
	logger := log.Ctx(ctx).
		With().
		Time("start", startsAt).
		Time("end", endsAt).
		Str("name", name).
		Logger()

	if !startsAt.Before(endsAt) {
		logger.Info().Msgf("Invalid dates for Game")
		return db.Game{}, fmt.Errorf("%w: start=%v end=%v", ErrInvalidDates, startsAt, endsAt)
	}

	if isActive && deactivateOthers {
		if err := s.queries.DeactivateAllGames(ctx); err != nil {
			return db.Game{}, fmt.Errorf("deactivate games: %w", err)
		}
		logger.Info().Msg("deactivated all games")
	}

	game, err := s.queries.CreateGame(ctx, db.CreateGameParams{
		ID:            db.NewID(),
		Name:          name,
		StartsAt:      startsAt,
		EndsAt:        endsAt,
		CommitPoints:  commitPoints,
		ProjectPoints: projectPoints,
		IsActive:      isActive,
	})
	if err != nil {
		logger.Warn().Msg("Failed to create game")
		return db.Game{}, fmt.Errorf("create game: %w", err)
	}

	logger.Info().Msg("created game")
	return game, nil
}

// CreateJulythonGame creates a standard Julython game for the given year/month.
func (s *GameService) CreateJulythonGame(
	ctx context.Context,
	year, month int,
	isActive, deactivateOthers bool,
) (db.Game, error) {
	var name string
	var startsAt, endsAt time.Time

	switch month {
	case 7:
		name = fmt.Sprintf("Julython %d", year)
		// Use noon UTC to avoid timezone rollover issues
		startsAt = time.Date(year, time.July, 1, 12, 0, 0, 0, time.UTC)
		endsAt = time.Date(year, time.July, 31, 12, 0, 0, 0, time.UTC)
	case 1:
		name = fmt.Sprintf("J(an)ulython %d", year)
		startsAt = time.Date(year, time.January, 1, 12, 0, 0, 0, time.UTC)
		endsAt = time.Date(year, time.January, 31, 12, 0, 0, 0, time.UTC)
	default:
		startsAt = time.Date(year, time.Month(month), 1, 12, 0, 0, 0, time.UTC)
		// Last day: first day of next month minus one day
		endsAt = time.Date(year, time.Month(month+1), 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		name = fmt.Sprintf("Test Game %s", startsAt.Format("January 2006"))
	}

	return s.CreateGame(ctx, name, startsAt, endsAt, 1, 10, isActive, deactivateOthers)
}

// GetActiveGame returns the currently active game, or ErrNoActiveGame.
func (s *GameService) GetActiveGame(ctx context.Context) (db.Game, error) {
	return s.GetActiveGameAt(ctx, time.Now())
}

// GetActiveGameAt returns the active game at a specific time.
func (s *GameService) GetActiveGameAt(ctx context.Context, t time.Time) (db.Game, error) {
	game, err := s.queries.GetActiveGameAtTime(ctx, t)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Game{}, ErrNoActiveGame
	}
	if err != nil {
		return db.Game{}, fmt.Errorf("get active game: %w", err)
	}
	return game, nil
}

// GetActiveOrLatestGame returns an active game or the most recently ended one.
func (s *GameService) GetActiveOrLatestGame(ctx context.Context) (db.Game, error) {
	now := time.Now()

	game, err := s.GetActiveGameAt(ctx, now)
	if err == nil {
		return game, nil
	}
	if !errors.Is(err, ErrNoActiveGame) {
		return db.Game{}, err
	}

	game, err = s.queries.GetLatestGame(ctx, now)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Game{}, ErrNoActiveGame
	}
	if err != nil {
		return db.Game{}, fmt.Errorf("get latest game: %w", err)
	}
	return game, nil
}

// AddCommit processes a commit and updates game scores.
// Only public projects contribute to game scoring; private repos
// are still analysed but excluded from the game.
func (s *GameService) AddCommit(ctx context.Context, commit db.Commit) error {
	logger := log.Ctx(ctx).
		With().
		Str("hash", commit.Hash.String).
		Time("timestamp", commit.Timestamp).
		Logger()

	game, err := s.GetActiveGameAt(ctx, commit.Timestamp)
	if errors.Is(err, ErrNoActiveGame) {
		logger.Debug().Msg("no active game for commit")
		return nil
	}

	if err != nil {
		return err
	}
	logger.Debug().Msgf("Adding commit to game: %s", game.Name)

	// Only public projects contribute to scoring.
	project, err := s.queries.GetProjectByID(ctx, commit.ProjectID)
	if err != nil {
		logger.Debug().Str("project_id", commit.ProjectID.String()).Msg("project not found, skipping scoring")
		return nil
	}
	if project.IsPrivate {
		logger.Debug().Str("project", project.Slug).Msg("skipping scoring for private project")
		return nil
	}

	if err := s.queries.SetCommitGame(ctx, db.SetCommitGameParams{
		ID:     commit.ID,
		GameID: db.UUID(game.ID),
	}); err != nil {
		return fmt.Errorf("set commit game: %w", err)
	}

	board, err := s.addPointsToBoard(ctx, game, commit)
	if err != nil {
		logger.Debug().Msgf("add points to board: %s", err)
		return fmt.Errorf("add points to board: %w", err)
	}

	if err := s.addLanguageBoards(ctx, game, commit); err != nil {
		logger.Debug().Msgf("add language boards: %s", err)
		return fmt.Errorf("add language boards: %w", err)
	}

	if commit.UserID.Valid {
		userID, _ := uuid.FromBytes(commit.UserID.Bytes[:])
		logger.Debug().Msgf("Assigning player boards: %v", userID)
		if err := s.assignPlayerBoards(ctx, game, userID, board.ID); err != nil {
			return fmt.Errorf("assign player boards: %w", err)
		}
	}

	return nil
}

func (s *GameService) addPointsToBoard(ctx context.Context, game db.Game, commit db.Commit) (db.Board, error) {
	var board db.Board
	_, err := s.queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
		ProjectID: commit.ProjectID,
		GameID:    game.ID,
	})
	isNewProject := errors.Is(err, pgx.ErrNoRows)

	points := game.CommitPoints
	if isNewProject {
		points += game.ProjectPoints
	}

	board, err = s.queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:               db.NewID(),
		GameID:           game.ID,
		ProjectID:        commit.ProjectID,
		Points:           points,
		CommitCount:      1,
		ContributorCount: 1,
	})
	return board, err
}

func (s *GameService) addLanguageBoards(ctx context.Context, game db.Game, commit db.Commit) error {
	for _, name := range commit.Languages {
		if name == "" {
			continue
		}

		lang, err := s.queries.GetOrCreateLanguage(ctx, db.GetOrCreateLanguageParams{
			ID:   db.NewID(),
			Name: name,
		})
		if err != nil {
			return fmt.Errorf("upsert language %q: %w", name, err)
		}

		_, err = s.queries.UpsertLanguageBoard(ctx, db.UpsertLanguageBoardParams{
			ID:          db.NewID(),
			GameID:      game.ID,
			LanguageID:  lang.ID,
			Points:      game.CommitPoints,
			CommitCount: 1,
		})
		if err != nil {
			return fmt.Errorf("upsert language board %q: %w", name, err)
		}
	}
	return nil
}

// assignPlayerBoards creates or updates a player and assigns the project
// board to the first available slot, then recalculates verified_points.
func (s *GameService) assignPlayerBoards(ctx context.Context, game db.Game, userID uuid.UUID, boardID uuid.UUID) error {
	player, err := s.queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: userID,
		GameID: game.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		counts, err := s.queries.CountUserCommitsForGame(ctx, db.CountUserCommitsForGameParams{
			UserID: pgtype.UUID{Bytes: userID, Valid: true},
			GameID: pgtype.UUID{Bytes: game.ID, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("count commits: %w", err)
		}

		points := int32(counts.CommitCount)*game.CommitPoints + int32(counts.ProjectCount)*game.ProjectPoints

		player, err = s.queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
			ID:             db.NewID(),
			GameID:         game.ID,
			UserID:         userID,
			CommitCount:    counts.CommitCount,
			ProjectCount:   counts.ProjectCount,
			Points:         points,
			AnalysisStatus: "pending",
		})
		if err != nil {
			return fmt.Errorf("create player: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("get player: %w", err)
	}

	// Get the player's current board IDs.
	ids, err := s.queries.GetPlayerBoardIds(ctx, player.ID)
	if err != nil {
		return fmt.Errorf("get board ids: %w", err)
	}

	// Count existing assigned boards.
	boardCount := 0
	if ids.Board1ID.Valid {
		boardCount++
	}
	if ids.Board2ID.Valid {
		boardCount++
	}
	if ids.Board3ID.Valid {
		boardCount++
	}

	// Only assign if the player has fewer than 3 boards and the board
	// isn't already assigned to this player.
	if boardCount < 3 {
		alreadyAssigned := (ids.Board1ID.Valid && ids.Board1ID == pgid(boardID)) ||
			(ids.Board2ID.Valid && ids.Board2ID == pgid(boardID)) ||
			(ids.Board3ID.Valid && ids.Board3ID == pgid(boardID))
		if !alreadyAssigned {
			var board1ID, board2ID, board3ID pgtype.UUID
			if ids.Board1ID.Valid {
				board1ID = ids.Board1ID
			}
			if ids.Board2ID.Valid {
				board2ID = ids.Board2ID
			}
			if ids.Board3ID.Valid {
				board3ID = ids.Board3ID
			}
			// Assign to the first available slot.
			switch boardCount {
			case 0:
				board1ID = pgid(boardID)
			case 1:
				if ids.Board1ID.Valid {
					board2ID = pgid(boardID)
				} else {
					board1ID = pgid(boardID)
				}
			case 2:
				if ids.Board1ID.Valid && ids.Board2ID.Valid {
					board3ID = pgid(boardID)
				} else if ids.Board1ID.Valid {
					board2ID = pgid(boardID)
				} else {
					board1ID = pgid(boardID)
				}
			}

			if _, err := s.queries.AssignBoards(ctx, db.AssignBoardsParams{
				PlayerID: player.ID,
				Board1ID: board1ID,
				Board2ID: board2ID,
				Board3ID: board3ID,
			}); err != nil {
				return fmt.Errorf("assign boards: %w", err)
			}
		}
	}

	counts, err := s.queries.CountUserCommitsForGame(ctx, db.CountUserCommitsForGameParams{
		UserID: db.UUID(player.UserID),
		GameID: db.UUID(game.ID),
	})
	if err != nil {
		return fmt.Errorf("count commits: %w", err)
	}

	// Recalculate verified_points from all boards.
	total, err := s.queries.GetPlayerBoardTotal(ctx, db.GetPlayerBoardTotalParams{
		Board1ID: db.UUIDFromPg(ids.Board1ID),
		Board2ID: db.UUIDFromPg(ids.Board2ID),
		Board3ID: db.UUIDFromPg(ids.Board3ID),
	})
	if err != nil {
		return fmt.Errorf("get board total: %w", err)
	}

	verifiedPoints := int32(counts.CommitCount) + total
	if err := s.queries.UpdatePlayerAnalysis(ctx, db.UpdatePlayerAnalysisParams{
		ID:             player.ID,
		VerifiedPoints: verifiedPoints,
		CommitCount:    counts.CommitCount,
		ProjectCount:   counts.ProjectCount,
		AnalysisStatus: "completed",
	}); err != nil {
		return fmt.Errorf("update analysis: %w", err)
	}

	return nil
}

// pgid creates a valid pgtype.UUID from a uuid.UUID.
func pgid(u uuid.UUID) pgtype.UUID {
	var b [16]byte
	copy(b[:], u[:])
	return pgtype.UUID{Bytes: b, Valid: true}
}

// UpsertPlayer recalculates and updates a player's score.
func (s *GameService) UpsertPlayer(ctx context.Context, game db.Game, userID uuid.UUID) error {
	counts, err := s.queries.CountUserCommitsForGame(ctx, db.CountUserCommitsForGameParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		GameID: pgtype.UUID{Bytes: game.ID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("count commits: %w", err)
	}

	points := int32(counts.CommitCount)*game.CommitPoints + int32(counts.ProjectCount)*game.ProjectPoints

	_, err = s.queries.UpsertPlayer(ctx, db.UpsertPlayerParams{
		ID:             db.NewID(),
		GameID:         game.ID,
		UserID:         userID,
		CommitCount:    counts.CommitCount,
		ProjectCount:   counts.ProjectCount,
		Points:         points,
		AnalysisStatus: "pending",
	})
	return err
}

// ClaimOrphanCommits finds commits matching emails and assigns them to a user.
func (s *GameService) ClaimOrphanCommits(ctx context.Context, userID uuid.UUID, emails []string) (int64, error) {
	logger := log.Ctx(ctx).
		With().
		Str("userID", userID.String()).
		Logger()

	if len(emails) == 0 {
		return 0, nil
	}

	game, err := s.GetActiveGame(ctx)
	if errors.Is(err, ErrNoActiveGame) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	claimed, err := s.queries.ClaimOrphanCommits(ctx, db.ClaimOrphanCommitsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Emails: emails,
		GameID: pgtype.UUID{Bytes: game.ID, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("claim commits: %w", err)
	}

	if claimed > 0 {
		if err := s.UpsertPlayer(ctx, game, userID); err != nil {
			return claimed, fmt.Errorf("upsert player after claim: %w", err)
		}
		logger.Info().
			Int64("count", claimed).
			Msg("claimed orphan commits")
	}

	return claimed, nil
}

// ApplyAnalysisAdjustment applies AI analysis results to a player's score.
func (s *GameService) ApplyAnalysisAdjustment(ctx context.Context, playerID uuid.UUID, adjustment int32) error {
	player, err := s.queries.GetPlayerByID(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get player: %w", err)
	}

	verifiedPoints := player.PotentialPoints + adjustment

	return s.queries.UpdatePlayerAnalysis(ctx, db.UpdatePlayerAnalysisParams{
		ID:             playerID,
		VerifiedPoints: verifiedPoints,
		AnalysisStatus: "completed",
	})
}

// GetLeaderboard returns top players for a game.
func (s *GameService) GetLeaderboard(ctx context.Context, gameID uuid.UUID, limit, offset int32) ([]db.GetLeaderboardRow, error) {
	return s.queries.GetLeaderboard(ctx, db.GetLeaderboardParams{
		GameID: gameID,
		Limit:  limit,
		Offset: offset,
	})
}

// GetProjectLeaderboard returns top projects for a game.
func (s *GameService) GetProjectLeaderboard(ctx context.Context, gameID uuid.UUID, limit int32) ([]db.GetProjectLeaderboardRow, error) {
	return s.queries.GetProjectLeaderboard(ctx, db.GetProjectLeaderboardParams{
		GameID:     gameID,
		LimitCount: limit,
	})
}

// GetLanguageLeaderboard returns top languages for a game.
func (s *GameService) GetLanguageLeaderboard(ctx context.Context, gameID uuid.UUID, limit int32) ([]db.GetLanguageLeaderboardRow, error) {
	return s.queries.GetLanguageLeaderboard(ctx, db.GetLanguageLeaderboardParams{
		GameID:     gameID,
		LimitCount: limit,
	})
}

// DeactivateUser marks a user as banned and inactive.
func (s *GameService) DeactivateUser(ctx context.Context, userID uuid.UUID, reason string) error {
	logger := log.Ctx(ctx).
		With().
		Str("userID", userID.String()).
		Str("reason", reason).
		Logger()
	err := s.queries.DeactivateUser(ctx, db.DeactivateUserParams{
		ID:     userID,
		Reason: pgtype.Text{String: reason},
	})
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}
	logger.Info().Msg("deactivated user")
	return nil
}

// DeactivateProject marks a project as inactive.
func (s *GameService) DeactivateProject(ctx context.Context, projectID uuid.UUID) error {
	logger := log.Ctx(ctx).
		With().
		Str("projectID", projectID.String()).
		Logger()

	if err := s.queries.DeactivateProject(ctx, projectID); err != nil {
		return fmt.Errorf("deactivate project: %w", err)
	}
	logger.Info().Msg("deactivated project")
	return nil
}

