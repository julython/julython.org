package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"july/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		value      any
		wantStatus int
		wantType   string
	}{
		{"ok map", http.StatusOK, map[string]bool{"ok": true}, http.StatusOK, "application/json"},
		{"created struct", http.StatusCreated, map[string]string{"id": "abc"}, http.StatusCreated, "application/json"},
		{"no content nil", http.StatusNoContent, nil, http.StatusNoContent, "application/json"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.Background()
			r = r.WithContext(ctx)

			respondJSON(w, r, tc.status, tc.value)

			require.Equal(t, tc.wantStatus, w.Code)
			require.Contains(t, w.Header().Get("Content-Type"), tc.wantType)
		})
	}
}

func TestRespondJSON_JSONBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.Background())

	type resp struct {
		Message string `json:"message"`
	}
	respondJSON(w, r, http.StatusOK, resp{Message: "hello"})

	require.Equal(t, `{"message":"hello"}`+"\n", w.Body.String())
}

func TestTimeAgo(t *testing.T) {
	fixed := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	orig := timeAgoNow
	timeAgoNow = func() time.Time { return fixed }
	defer func() { timeAgoNow = orig }()

	tests := []struct {
		name     string
		offset   time.Duration
		expected string
	}{
		{"just now", 0, "just now"},
		{"30 seconds", 30 * time.Second, "just now"},
		{"1 minute", time.Minute, "1 minute ago"},
		{"5 minutes", 5 * time.Minute, "5 minutes ago"},
		{"59 minutes", 59 * time.Minute, "59 minutes ago"},
		{"1 hour", time.Hour, "1 hour ago"},
		{"3 hours", 3 * time.Hour, "3 hours ago"},
		{"23 hours", 23 * time.Hour, "23 hours ago"},
		{"1 day", 24 * time.Hour, "1 day ago"},
		{"5 days", 5 * 24 * time.Hour, "5 days ago"},
		{"6 days", 6 * 24 * time.Hour, "6 days ago"},
		{"7 days", 7 * 24 * time.Hour, "Apr 25"}, // default format kicks in
		{"30 days", 30 * 24 * time.Hour, "Apr 2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := timeAgo(fixed.Add(-tc.offset))
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestStringFromNull(t *testing.T) {
	tests := []struct {
		name     string
		input    pgtype.Text
		expected string
	}{
		{"valid string", pgtype.Text{String: "hello", Valid: true}, "hello"},
		{"empty string valid", pgtype.Text{String: "", Valid: true}, ""},
		{"invalid null", pgtype.Text{String: "hello", Valid: false}, ""},
		{"empty invalid", pgtype.Text{String: "", Valid: false}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := db.StringFromNull(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}
