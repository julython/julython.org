package players

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"july/internal/auth"
	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/services"
)

func Register(mux *http.ServeMux, q *db.Queries, gs *services.GameService) {
	h := &handler{queries: q, gameService: gs}
	mux.HandleFunc("GET /player/{username}", h.Player)
	mux.HandleFunc("POST /player/{username}", h.Update)
}

type handler struct {
	queries     *db.Queries
	gameService *services.GameService
}

type boardInfo struct {
	ID             uuid.UUID
	Points         int32
	VerifiedPoints int32
	CommitCount    int32
	ProjectName    string
	ProjectSlug    string
}

type PlayerData struct {
	Username  string
	Name      string
	AvatarURL string
	Boards    []boardInfo
}

func (h *handler) Player(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the requested username
	username := r.PathValue("username")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderPage(w, r, username, nil)
		return
	}

	// Fetch player + boards + projects in a single query.
	rows, err := h.queries.GetPlayerWithBoards(ctx, db.GetPlayerWithBoardsParams{
		GameID:   game.ID,
		Username: username,
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(rows) == 0 {
		h.renderPage(w, r, username, nil)
		return
	}

	data := PlayerData{
		Username:  rows[0].Username,
		Name:      rows[0].Name,
		AvatarURL: rows[0].AvatarUrl.String,
		Boards:    make([]boardInfo, 0, len(rows)),
	}
	for _, r := range rows {
		data.Boards = append(data.Boards, boardInfo{
			ID:             r.ID,
			Points:         r.Points,
			VerifiedPoints: r.VerifiedPoints,
			CommitCount:    r.CommitCount,
			ProjectName:    r.ProjectName,
			ProjectSlug:    r.Slug,
		})
	}

	h.renderPage(w, r, username, &data)
}

// Update handles POST /player/{username} — the HTMX endpoint
// to Update boards on the player page. Only the player can Update their own boards.
func (h *handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	username := r.PathValue("username")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	// Auth check: must be logged in
	sessionUser := auth.UserFromContext(ctx)
	if sessionUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get the player for the requested user
	u, err := h.queries.GetUserByUsername(ctx, username)
	if err != nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Auth check: only the player can Update their own boards
	if u.ID != sessionUser.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	player, err := h.queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: u.ID,
		GameID: game.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Parse request body
	var body UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Validate ownership and build params in a single loop.
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Re-render boards list fragment
	h.renderPlayerBoardFragment(w, r, username, &u)
}

func (h *handler) renderPlayerBoardFragment(w http.ResponseWriter, r *http.Request, username string, u *db.User) {
	ctx := r.Context()

	game, err := h.gameService.GetActiveOrLatestGame(ctx)
	if err != nil {
		h.renderNoBoards(w, r, username)
		return
	}

	player, err := h.queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: u.ID,
		GameID: game.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		h.renderNoBoards(w, r, username)
		return
	}

	var rows []boardInfo
	var boardIDs []uuid.UUID
	if player.Board1ID.Valid {
		boardIDs = append(boardIDs, player.Board1ID.Bytes)
	}
	if player.Board2ID.Valid {
		boardIDs = append(boardIDs, player.Board2ID.Bytes)
	}
	if player.Board3ID.Valid {
		boardIDs = append(boardIDs, player.Board3ID.Bytes)
	}

	if len(boardIDs) > 0 {
		boardRows, err := h.queries.GetBoardByIDsAndGame(ctx, db.GetBoardByIDsAndGameParams{
			BoardIds: boardIDs,
			GameID:   game.ID,
		})
		if err == nil {
			for _, b := range boardRows {
				project, _ := h.queries.GetProjectByID(ctx, b.ProjectID)
				rows = append(rows, boardInfo{
					ID:             b.ID,
					Points:         b.Points,
					VerifiedPoints: b.VerifiedPoints,
					CommitCount:    b.CommitCount,
					ProjectName:    project.Name,
					ProjectSlug:    project.Slug,
				})
			}
		}
	}

	h.renderPlayerBoard(w, r, username, player, rows)
}

type UpdateRequest struct {
	Board1ID *uuid.UUID `json:"board_1"`
	Board2ID *uuid.UUID `json:"board_2"`
	Board3ID *uuid.UUID `json:"board_3"`
}

func (h *handler) renderPage(w http.ResponseWriter, r *http.Request, username string, data *PlayerData) {
	ctx := r.Context()

	ld := layout.LayoutData{
		Title:       "Player: " + username,
		CurrentPath: "/player/" + username,
	}

	pd := PlayersData{
		Username: username,
	}
	if data != nil {
		pd.Username = data.Username
		pd.Name = data.Name
		pd.AvatarURL = data.AvatarURL
		pd.Boards = data.Boards
	}

	if r.Header.Get("HX-Request") == "true" {
		PlayersList(pd).Render(ctx, w)
	} else {
		PlayersPage(ld, pd).Render(ctx, w)
	}
}

func (h *handler) renderNoBoards(w http.ResponseWriter, r *http.Request, username string) {
	pd := PlayersData{
		Username: username,
	}
	ctx := r.Context()
	PlayersList(pd).Render(ctx, w)
}

func (h *handler) renderPlayerBoard(w http.ResponseWriter, r *http.Request, username string, player db.Player, rows []boardInfo) {
	pd := PlayersData{
		Username: username,
		Boards:   rows,
	}
	ctx := r.Context()
	PlayersList(pd).Render(ctx, w)
}
