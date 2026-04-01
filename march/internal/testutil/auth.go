package testutil

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"july/internal/db"
)

// LoginAs authenticates the test client as the given user by hitting the
// test-only /test/login endpoint (registered by api.NewTestRouter).
// Subsequent requests on env.Client will carry the session cookie.
func (env *TestEnv) LoginAs(t *testing.T, user db.User) {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	env.Client.Jar = jar

	resp, err := env.Client.Get(env.Server.URL + "/test/login?userID=" + user.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "test login failed — is the route registered?")

	// Fail fast if SCS didn't set a session cookie. Without this the next
	// authenticated request silently returns 401 and the real cause is buried.
	u, _ := url.Parse(env.Server.URL)
	require.NotEmpty(t, jar.Cookies(u), "no session cookie after login — check SessionKeyUserID type in test_router.go")
}

// Logout clears the session cookie so the next request is unauthenticated.
func (env *TestEnv) Logout() {
	env.Client.Jar = nil
}
