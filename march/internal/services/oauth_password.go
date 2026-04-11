package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"july/internal/db"
)

// PasswordOAuthProvider implements OAuthProvider using bcrypt-hashed credentials
// stored in user_identifiers.data. It is never exposed in the UI — credentials
// are inserted directly by test fixtures via SetPassword in testutil.
type PasswordOAuthProvider struct {
	queries *db.Queries
	baseURL string
}

func NewPasswordOAuth(queries *db.Queries, baseURL string) *PasswordOAuthProvider {
	return &PasswordOAuthProvider{queries: queries, baseURL: baseURL}
}

func (p *PasswordOAuthProvider) Provider() string { return "password" }

// AuthURL returns our own server's callback-prep endpoint so the login flow
// follows the same redirect pattern as real OAuth providers.
func (p *PasswordOAuthProvider) AuthorizationURL(state string, verifier string) string {
	return p.baseURL + "/auth/password/authorize?state=" + state
}

// ExchangeToken decodes a base64 "email:password" code, verifies the bcrypt
// hash stored in the identifier row, and returns the user's UUID as the token.
func (p *PasswordOAuthProvider) ExchangeCode(ctx context.Context, code string, verifier string) (OAuthTokens, error) {
	raw, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("invalid code encoding")
	}

	email, password, ok := strings.Cut(string(raw), ":")
	if !ok {
		return OAuthTokens{}, fmt.Errorf("invalid code format")
	}

	identifierValue := "email:" + email

	row, err := p.queries.GetPasswordHash(ctx, identifierValue)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("invalid credentials")
	}

	var data map[string]string
	if err := json.Unmarshal(row, &data); err != nil {
		return OAuthTokens{}, fmt.Errorf("invalid credentials")
	}
	hash, ok := data["password_hash"]
	if !ok {
		return OAuthTokens{}, fmt.Errorf("no password set for this account")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return OAuthTokens{}, fmt.Errorf("invalid credentials")
	}

	user, err := p.queries.GetUserByPasswordIdentifier(ctx, identifierValue)
	if err != nil {
		return OAuthTokens{}, fmt.Errorf("invalid credentials")
	}

	return OAuthTokens{AccessToken: user.ID.String()}, nil
}

// GetUser looks up the user by the UUID token returned from ExchangeToken.
// The returned OAuthUser.ID becomes the value stored in user_identifiers so
// subsequent logins resolve directly without going through email lookup.
func (p *PasswordOAuthProvider) GetUser(ctx context.Context, token OAuthTokens) (OAuthUser, error) {
	logger := log.Ctx(ctx)
	userID, err := uuid.Parse(token.AccessToken)
	logger.Info().Msgf("Looking for userID: %s", userID)

	if err != nil {
		return OAuthUser{}, fmt.Errorf("invalid token: %w", err)
	}
	user, err := p.queries.GetUserByID(ctx, userID)
	if err != nil {
		return OAuthUser{}, fmt.Errorf("user not found: %w", err)
	}
	identifiers, err := p.queries.GetVerifiedEmails(ctx, userID)
	emails := make([]EmailAddress, len(identifiers))
	for i, id := range identifiers {
		// Parse the email from the value
		email := strings.Split(id.Value, ":")[1]
		logger.Info().Str("email", email).Bool("verified", id.Verified).Msg("found emails")
		emails[i] = EmailAddress{Email: email, Primary: id.IsPrimary, Verified: id.Verified}
	}

	return OAuthUser{
		ID:        user.ID.String(),
		Provider:  "password",
		Username:  user.Username,
		Name:      user.Name,
		AvatarURL: user.AvatarUrl.String,
		Emails:    emails,
	}, nil
}

func (p *PasswordOAuthProvider) RefreshToken(_ context.Context, _ string) (OAuthTokens, error) {
	return OAuthTokens{}, ErrRefreshNotSupported // Password tokens are just the userID
}

func (p *PasswordOAuthProvider) RevokeToken(_ context.Context, _ OAuthTokens) error {
	return ErrRefreshNotSupported // Password tokens are just the userID
}
