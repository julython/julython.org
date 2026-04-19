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

	if err := s.queries.SetCommitGame(ctx, db.SetCommitGameParams{
		ID:     commit.ID,
		GameID: db.UUID(game.ID),
	}); err != nil {
		return fmt.Errorf("set commit game: %w", err)
	}

	if err := s.addPointsToBoard(ctx, game, commit); err != nil {
		logger.Debug().Msgf("add points to board: %s", err)
		return fmt.Errorf("add points to board: %w", err)
	}

	if err := s.addLanguageBoards(ctx, game, commit); err != nil {
		logger.Debug().Msgf("add language boards: %s", err)
		return fmt.Errorf("add language boards: %w", err)
	}

	if commit.UserID.Valid {
		userID, _ := uuid.FromBytes(commit.UserID.Bytes[:])
		logger.Debug().Msgf("Upsert user %v", userID)
		if err := s.UpsertPlayer(ctx, game, userID); err != nil {
			return fmt.Errorf("upsert player: %w", err)
		}
	}

	return nil
}

func (s *GameService) addPointsToBoard(ctx context.Context, game db.Game, commit db.Commit) error {
	_, err := s.queries.GetBoardByProjectAndGame(ctx, db.GetBoardByProjectAndGameParams{
		ProjectID: commit.ProjectID,
		GameID:    game.ID,
	})
	isNewProject := errors.Is(err, pgx.ErrNoRows)

	points := game.CommitPoints
	if isNewProject {
		points += game.ProjectPoints
	}

	_, err = s.queries.UpsertBoard(ctx, db.UpsertBoardParams{
		ID:               db.NewID(),
		GameID:           game.ID,
		ProjectID:        commit.ProjectID,
		Points:           points,
		CommitCount:      1,
		ContributorCount: 1,
	})
	return err
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
		GameID:      gameID,
		LimitCount:  limit,
		OffsetCount: offset,
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
