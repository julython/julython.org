package players

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/components/analysis"
	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/features/projects"
	"july/internal/services"
	"july/internal/shared"
)

func Register(mux *http.ServeMux, q *db.Queries, gs *services.GameService) {
	h := &handler{queries: q, gameService: gs, projectService: projects.NewProjectService(q)}
	mux.HandleFunc("GET /player/{username}", h.Player)
	mux.HandleFunc("POST /player/{username}", h.Update)
}

type handler struct {
	queries        *db.Queries
	gameService    *services.GameService
	projectService *projects.ProjectService
}

type boardInfo struct {
	ID             uuid.UUID
	Points         int32
	VerifiedPoints int32
	CommitCount    int32
	ProjectName    string
	ProjectSlug    string

	// Analysis data (populated by projectService).
	AnalysisTiles     []analysis.AnalysisTile
	AnalysisEarnedPts int
	AnalysisMaxPts    int
	LastAnalyzedAgo   string
	AnalysisRunCount  int

	// Game activity (populated by projectService)
	HasGame          bool
	Board            *analysis.BoardStats
	CommitsThisMonth int
	CommitsThisWeek  int
	FileTouchCount   int
	UniqueDirs       int
}

type PlayersData struct {
	Username  string
	Name      string
	AvatarURL string
	Boards    []boardInfo
	IsOwner   bool

	// Recent commits (only for the player's own page).
	RecentCommits []RecentCommit
}

// RecentCommit is a shared type used by multiple features (home, activity).
type RecentCommit struct {
	Username    string
	Hash        string
	Name        string // Display name for avatar initials
	Author      string
	AvatarURL   string
	Message     string
	Project     string
	ProjectName string
	TimeAgo     string
}

// Player handles GET /player/{username}.
func (h *handler) Player(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	username := r.PathValue("username")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		logger.Info().Msg("Active game not found")
		http.NotFound(w, r)
		return
	}

	h.renderPlayerData(w, r, game.ID, username)
}

// Update handles POST /player/{username} — the HTMX endpoint
// to swap boards on the player page. Only the player can update their own boards.
func (h *handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	username := r.PathValue("username")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	// Auth: must be logged in
	sessionUser := auth.UserFromContext(ctx)
	if sessionUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("GetActiveOrLatestGame failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Auth: only the player can swap their own boards
	u, err := h.queries.GetUserByUsername(ctx, username)
	if err != nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	if u.ID != sessionUser.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Assign the selected boards
	player, err := h.queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: u.ID,
		GameID: game.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.Error().Err(err).Msg("GetPlayerByUserAndGame failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Unlink a board (the only operation the player page handles).
	deleteStr := r.FormValue("delete_board")
	if deleteStr == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deleteID, err := uuid.Parse(deleteStr)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if _, err := h.queries.UnlinkBoard(ctx, db.UnlinkBoardParams{
		PlayerID: player.ID,
		DeleteBoard: pgtype.UUID{
			Bytes: deleteID,
			Valid: true,
		},
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Board not found in player's slots", http.StatusBadRequest)
			return
		}
		logger.Error().Err(err).Msg("UnlinkBoard failed")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.renderPlayerData(w, r, game.ID, username)
}

// renderPlayerData fetches a player's boards and renders the response.
// It handles both full-page and HTMX fragment responses.
func (h *handler) renderPlayerData(w http.ResponseWriter, r *http.Request, gameID uuid.UUID, username string) {
	ctx := r.Context()
	logger := log.Ctx(r.Context())

	isOwner := false
	// Auth: check if the user is the owner
	sessionUser := auth.UserFromContext(ctx)
	if sessionUser != nil {
		isOwner = sessionUser.Username == username
	}

	rows, err := h.queries.GetPlayerWithBoards(ctx, db.GetPlayerWithBoardsParams{
		GameID:   gameID,
		Username: username,
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to find player")
		return
	}

	if len(rows) == 0 {
		h.renderNoBoards(w, r, username)
		return
	}

	ld := layout.LayoutData{
		Title:       "Player: " + username,
		CurrentPath: "/player/" + username,
		User:        layout.UserInfoFromContext(r),
	}

	pd := PlayersData{
		Username:  rows[0].Username,
		Name:      rows[0].Name,
		AvatarURL: rows[0].AvatarUrl.String,
		Boards:    make([]boardInfo, 0, len(rows)),
		IsOwner:   isOwner,
	}
	for _, r := range rows {
		pd.Boards = append(pd.Boards, boardInfo{
			ID:             r.ID,
			Points:         r.Points,
			VerifiedPoints: r.VerifiedPoints,
			CommitCount:    r.CommitCount,
			ProjectName:    r.ProjectName,
			ProjectSlug:    r.Slug,
		})
		// Fetch analysis and game activity for this project.
		if projData, projErr := h.projectService.BuildProjectBoardInfo(ctx, r.ProjectID, gameID); projErr == nil {
			bd := &pd.Boards[len(pd.Boards)-1]
			bd.AnalysisTiles = projData.AnalysisBoard.Tiles
			bd.AnalysisEarnedPts = projData.AnalysisBoard.EarnedPts
			bd.AnalysisMaxPts = projData.AnalysisBoard.MaxPts
			bd.LastAnalyzedAgo = projData.AnalysisBoard.LastAnalyzedAgo
			bd.AnalysisRunCount = projData.AnalysisBoard.AnalysisRunCount
			bd.HasGame = projData.GameActivity.HasGame
			bd.Board = projData.GameActivity.Board
			bd.CommitsThisMonth = projData.GameActivity.CommitsThisMonth
			bd.CommitsThisWeek = projData.GameActivity.CommitsThisWeek
			bd.FileTouchCount = projData.GameActivity.FileTouchCount
			bd.UniqueDirs = projData.GameActivity.UniqueDirs
		}
	}

	// Fetch recent commits for this player (shown to all viewers).
	user, err := h.queries.GetUserByUsername(ctx, username)
	if err == nil {
		commits, err := h.queries.GetCommitsByUserAndGame(ctx, db.GetCommitsByUserAndGameParams{
			UserID:      db.UUID(user.ID),
			GameID:      db.UUID(gameID),
			LimitCount:  20,
			OffsetCount: 0,
		})
		if err == nil {
			pd.RecentCommits = make([]RecentCommit, 0, len(commits))
			for _, c := range commits {
				pd.RecentCommits = append(pd.RecentCommits, RecentCommit{
					Hash:        c.Hash.String,
					Message:     c.Message,
					Name:        c.Author.String,
					Author:      c.Author.String,
					AvatarURL:   user.AvatarUrl.String,
					Project:     c.ProjectSlug,
					ProjectName: c.ProjectName,
					TimeAgo:     shared.TimeAgo(c.Timestamp),
				})
			}
		}
	}

	if r.Header.Get("HX-Request") == "true" || r.Method == "POST" {
		if err := PlayersList(pd).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("PlayersList render failed")
		}
	} else {
		if err := PlayersPage(ld, pd).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("PlayersPage render failed")
		}
	}
}

func (h *handler) renderNoBoards(w http.ResponseWriter, r *http.Request, username string) {
	ctx := r.Context()
	logger := log.Ctx(ctx)

	ld := layout.LayoutData{
		Title:       "Player: " + username,
		CurrentPath: "/player/" + username,
		User:        layout.UserInfoFromContext(r),
	}

	pd := PlayersData{
		Username: username,
	}

	if r.Header.Get("HX-Request") == "true" || r.Method == "POST" {
		if err := PlayersList(pd).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("PlayersList render failed")
		}
	} else {
		if err := PlayersPage(ld, pd).Render(ctx, w); err != nil {
			logger.Error().Err(err).Msg("PlayersPage render failed")
		}
	}
}
