package boards

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"july/internal/components/layout"
	"july/internal/db"
	"july/internal/services"
)

func Register(mux *http.ServeMux, q *db.Queries, gs *services.GameService) {
	h := &handler{queries: q, gameService: gs}
	mux.HandleFunc("GET /player/{username}", h.Player)
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

type playerData struct {
	Username string
	Name     string
	AvatarURL string
	Boards   []boardInfo
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
		renderPage(w, r, username, nil)
		return
	}

	// Get the player for the requested user
	u, _ := h.queries.GetUserByUsername(ctx, username)
	player, err := h.queries.GetPlayerByUserAndGame(ctx, db.GetPlayerByUserAndGameParams{
		UserID: u.ID,
		GameID: game.ID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		renderPage(w, r, username, nil)
		return
	}
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

	data := playerData{
		Username:  u.Username,
		Name:      u.Name,
		AvatarURL: u.AvatarUrl.String,
		Boards:    rows,
	}

	renderPage(w, r, username, &data)
}

func renderPage(w http.ResponseWriter, r *http.Request, username string, data *playerData) {
	ctx := r.Context()

	ld := layout.LayoutData{
		Title:       "Player: " + username,
		CurrentPath: "/player/" + username,
	}

	pd := BoardsData{
		Username: username,
	}
	if data != nil {
		pd.Username = data.Username
		pd.Name = data.Name
		pd.AvatarURL = data.AvatarURL
		pd.Boards = data.Boards
	}

	if r.Header.Get("HX-Request") == "true" {
		BoardsList(pd).Render(ctx, w)
	} else {
		BoardsPage(ld, pd).Render(ctx, w)
	}
}
