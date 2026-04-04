package testutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"july/internal/db"
)

func CreateUser(t *testing.T, env *TestEnv, username, name string) db.User {
	t.Helper()

	user, err := env.Queries.CreateUser(context.Background(), db.CreateUserParams{
		ID:        db.NewID(),
		Name:      name,
		Username:  username,
		AvatarUrl: db.Text(""),
		Role:      "user",
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func CreateUserIdentifier(t *testing.T, env *TestEnv, userID uuid.UUID, idType, value string, verified, primary bool) db.UserIdentifier {
	t.Helper()

	key := fmt.Sprintf("%s:%s", idType, value)
	identifier, err := env.Queries.UpsertUserIdentifier(context.Background(), db.UpsertUserIdentifierParams{
		Value:     key,
		Type:      idType,
		UserID:    userID,
		Verified:  verified,
		IsPrimary: primary,
		Data:      []byte("{}"),
	})
	if err != nil {
		t.Fatalf("failed to create user identifier: %v", err)
	}
	return identifier
}

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
