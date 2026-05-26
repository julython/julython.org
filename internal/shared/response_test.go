package shared

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondJSON(t *testing.T) {
	type resp struct {
		Message string `json:"message"`
	}

	tests := []struct {
		name     string
		status   int
		value    any
		wantCode int
	}{
		{name: "OK", status: http.StatusOK, value: resp{Message: "hello"}, wantCode: http.StatusOK},
		{name: "Created", status: http.StatusCreated, value: resp{Message: "created"}, wantCode: http.StatusCreated},
		{name: "Error", status: http.StatusBadRequest, value: resp{Message: "bad"}, wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			RespondJSON(rec, req, tc.status, tc.value)
			if rec.Result().StatusCode != tc.wantCode {
				t.Errorf("status = %d, want %d", rec.Result().StatusCode, tc.wantCode)
			}
		})
	}

	// Test encoding error (no panic)
	t.Run("EncodeError", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		RespondJSON(rec, req, http.StatusOK, make(chan int))
	})
}
