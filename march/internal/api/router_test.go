package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderboard(t *testing.T) {
	// Integration tests - full stack

	t.Run("display leaderboard", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateTestScenario(t, env)

		resp := getJSON(t, env, "/leaders")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// Helpers
func postJSON(t *testing.T, env *testutil.TestEnv, path string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := env.Client.Post(env.Server.URL+path, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func getJSON(t *testing.T, env *testutil.TestEnv, path string) *http.Response {
	t.Helper()
	resp, err := env.Client.Get(env.Server.URL + path)
	require.NoError(t, err)
	return resp
}
