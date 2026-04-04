package testutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

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

	data := []byte("{}")
	if idType == "email" {
		hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
		require.NoError(t, err)

		data, _ = json.Marshal(map[string]string{"password_hash": string(hash)})
	}

	key := fmt.Sprintf("%s:%s", idType, value)
	identifier, err := env.Queries.UpsertUserIdentifier(context.Background(), db.UpsertUserIdentifierParams{
		Value:     key,
		Type:      idType,
		UserID:    userID,
		Verified:  verified,
		IsPrimary: primary,
		Data:      data,
	})
	if err != nil {
		t.Fatalf("failed to create user identifier: %v", err)
	}
	return identifier
}

const testPassword = "test-only-not-for-production"

// LoginAs authenticates the test client by walking the real OAuth flow with
// the password provider:
//
//  1. GET /auth/login/password  — sets PKCE state in session, redirects to
//     /auth/password/authorize?state=xyz
//  2. Extracts state from the redirect Location header.
//  3. GET /auth/callback?code=<base64(email:password)>&state=xyz  — runs the
//     full callback: state check, ExchangeToken, GetUser, session write.
//
// Requires cfg.Auth.PasswordLoginEnabled = true and a prior call to
// CreateUserIdentifier so the email row exists to receive the hash.
func (env *TestEnv) LoginAs(t *testing.T, email string) {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	// Don't follow redirects — we need to read the Location header from step 1.
	noFollow := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: initiate the OAuth flow to get the state written into the session.
	resp, err := noFollow.Get(env.Server.URL + "/auth/login/password")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode,
		"expected redirect from /auth/login/password — is PasswordLoginEnabled set?")

	location := resp.Header.Get("Location")
	parsed, err := url.Parse(location)
	require.NoError(t, err, "could not parse redirect Location: %s", location)
	state := parsed.Query().Get("state")
	require.NotEmpty(t, state, "no state in redirect Location: %s", location)

	// Step 2: call the real callback with credentials encoded as the OAuth code.
	code := base64.StdEncoding.EncodeToString([]byte(email + ":" + testPassword))

	callbackURL := fmt.Sprintf("%s/auth/callback?code=%s&state=%s",
		env.Server.URL,
		url.QueryEscape(code),
		url.QueryEscape(state),
	)
	resp2, err := noFollow.Get(callbackURL)
	require.NoError(t, err)
	defer resp2.Body.Close()
	// Successful login redirects home; anything ≥ 400 is a failure.
	require.Less(t, resp2.StatusCode, 400,
		"callback failed with status %d", resp2.StatusCode)

	// Hand the populated jar to the main test client.
	env.Client.Jar = jar

	u, _ := url.Parse(env.Server.URL)
	require.NotEmpty(t, jar.Cookies(u), "no session cookie after OAuth callback")
}

// Logout clears the session cookie so the next request is unauthenticated.
func (env *TestEnv) Logout() {
	env.Client.Jar = nil
}
