package players

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"july/internal/auth"
	"july/internal/components/analysis"
	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/features/projects"
	"july/internal/services"
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

type PlayerData struct {
	Username  string
	Name      string
	AvatarURL string
	Boards    []boardInfo
}

type UpdateRequest struct {
	Board1ID *uuid.UUID `json:"board_1"`
	Board2ID *uuid.UUID `json:"board_2"`
	Board3ID *uuid.UUID `json:"board_3"`
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

	var body UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	params := db.AssignBoardsParams{PlayerID: player.ID}
	for i, idPtr := range []*uuid.UUID{body.Board1ID, body.Board2ID, body.Board3ID} {
		if idPtr == nil {
			continue
		}
		owned, err := h.queries.ValidateBoardOwnership(ctx, db.ValidateBoardOwnershipParams{
			BoardID: *idPtr,
			GameID:  game.ID,
			Owner:   u.Username,
		})
		if err != nil || owned.ID == uuid.Nil {
			http.Error(w, "Board not owned by player", http.StatusForbidden)
			return
		}
		switch i {
		case 0:
			params.Board1ID = db.UUID(*idPtr)
		case 1:
			params.Board2ID = db.UUID(*idPtr)
		case 2:
			params.Board3ID = db.UUID(*idPtr)
		}
	}

	if _, err := h.queries.AssignBoards(ctx, params); err != nil {
		logger.Error().Err(err).Msg("AssignBoards failed")
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
