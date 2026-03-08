package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"

	"july/internal/db"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrIdentifierConflict = errors.New("identifier belongs to another user")
	ErrMultipleUsers      = errors.New("emails match multiple existing users")
)

type IdentifierType string

const (
	IdentifierEmail  IdentifierType = "email"
	IdentifierGitHub IdentifierType = "github"
	IdentifierGitLab IdentifierType = "gitlab"
)

type UserService struct {
	queries *db.Queries
}

func NewUserService(queries *db.Queries) *UserService {
	return &UserService{queries: queries}
}

func (s *UserService) FindByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrUserNotFound
	}
	return user, err
}

func (s *UserService) FindByUsername(ctx context.Context, username string) (db.User, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrUserNotFound
	}
	return user, err
}

// FindByKey looks up a user by identifier value (e.g., "github:12345")
func (s *UserService) FindByKey(ctx context.Context, key string) (db.User, error) {
	user, err := s.queries.FindUserByIdentifier(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, ErrUserNotFound
	}
	return user, err
}

// FindByIdentifier looks up a user by type and value
func (s *UserService) FindByIdentifier(ctx context.Context, idType IdentifierType, value string) (db.User, error) {
	key := fmt.Sprintf("%s:%s", idType, value)
	return s.FindByKey(ctx, key)
}

// FindByEmail looks up a user by email address
func (s *UserService) FindByEmail(ctx context.Context, email string) (db.User, error) {
	return s.FindByIdentifier(ctx, IdentifierEmail, email)
}

// UpsertIdentifier creates or updates an identifier for a user.
// Returns the identifier, whether it was created, and any error.
// Returns ErrIdentifierConflict if the identifier exists for a different user.
func (s *UserService) UpsertIdentifier(
	ctx context.Context,
	userID uuid.UUID,
	idType IdentifierType,
	value string,
	data map[string]any,
	verified bool,
	primary bool,
) (db.UserIdentifier, bool, error) {
	key := fmt.Sprintf("%s:%s", idType, value)

	// Check if identifier exists and belongs to different user
	existing, err := s.queries.GetUserIdentifier(ctx, key)
	if err == nil && existing.UserID != userID {
		return db.UserIdentifier{}, false, ErrIdentifierConflict
	}

	// Upsert - the ON CONFLICT clause handles the rest
	identifier, err := s.queries.UpsertUserIdentifier(ctx, db.UpsertUserIdentifierParams{
		Value:     key,
		Type:      string(idType),
		UserID:    userID,
		Verified:  verified,
		IsPrimary: primary,
		Data:      toJSONB(data),
	})
	if err != nil {
		return db.UserIdentifier{}, false, err
	}

	// Determine if created by comparing timestamps
	created := identifier.CreatedAt.Equal(identifier.UpdatedAt)
	return identifier, created, nil
}

// OAuthLoginOrRegister handles the OAuth login flow:
// 1. Find by OAuth provider ID -> login, update data
// 2. Find by verified email -> link account
// 3. Create new user
// Returns the user and whether a new user/identity was created.
func (s *UserService) OAuthLoginOrRegister(ctx context.Context, oauth OAuthUser) (db.User, bool, error) {
	idType := IdentifierType(oauth.Provider)
	verifiedEmails := filterVerifiedEmails(oauth.Emails)

	// 1. Check for existing OAuth identity
	if user, err := s.FindByKey(ctx, oauth.Key()); err == nil {
		// Update user info
		err = s.queries.UpdateUser(ctx, db.UpdateUserParams{
			ID:        user.ID,
			Name:      toNullString(coalesce(oauth.Name, user.Name)),
			AvatarUrl: toNullString(coalesce(oauth.AvatarURL, stringFromNull(user.AvatarUrl))),
		})
		if err != nil {
			return db.User{}, false, err
		}

		// Refresh user after update
		user, _ = s.queries.GetUserByID(ctx, user.ID)

		// Update OAuth identifier data
		if _, _, err := s.UpsertIdentifier(ctx, user.ID, idType, oauth.ID, oauth.Data, true, false); err != nil {
			return db.User{}, false, err
		}

		// Upsert any new emails
		emailsAdded, err := s.upsertEmails(ctx, user.ID, verifiedEmails)
		if err != nil {
			return db.User{}, false, err
		}

		return user, emailsAdded, nil
	}

	// 2. Check if any verified emails match existing users
	existingUser, err := s.findUserByEmails(ctx, verifiedEmails)
	if err != nil {
		return db.User{}, false, err
	}

	if existingUser != nil {
		// Link OAuth to existing user
		if _, _, err := s.UpsertIdentifier(ctx, existingUser.ID, idType, oauth.ID, oauth.Data, true, false); err != nil {
			return db.User{}, false, err
		}

		if _, err := s.upsertEmails(ctx, existingUser.ID, verifiedEmails); err != nil {
			return db.User{}, false, err
		}

		return *existingUser, false, nil
	}

	// 3. Create new user
	newUser, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		ID:        db.NewID(),
		Username:  oauth.Username,
		Name:      oauth.Name,
		AvatarUrl: toNullString(oauth.AvatarURL),
		Role:      "user",
	})
	if err != nil {
		return db.User{}, false, err
	}

	// Add OAuth identifier
	if _, _, err := s.UpsertIdentifier(ctx, newUser.ID, idType, oauth.ID, oauth.Data, true, false); err != nil {
		return db.User{}, false, err
	}

	// Add email identifiers
	if _, err := s.upsertEmails(ctx, newUser.ID, verifiedEmails); err != nil {
		return db.User{}, false, err
	}

	return newUser, true, nil
}

// GetOAuthToken returns the stored access token for a given provider.
func (s *UserService) GetOAuthToken(ctx context.Context, userID uuid.UUID, provider IdentifierType) (string, error) {
	identifier, err := s.queries.GetUserIdentifierByUserAndType(ctx, db.GetUserIdentifierByUserAndTypeParams{
		UserID: userID,
		Type:   string(provider),
	})
	if err != nil {
		return "", err
	}
	data := fromJSONB(identifier.Data)
	token, _ := data["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("no access token found for %s", provider)
	}
	return token, nil
}

func (s *UserService) findUserByEmails(ctx context.Context, emails []EmailAddress) (*db.User, error) {
	var found *db.User

	for _, email := range emails {
		user, err := s.FindByEmail(ctx, email.Email)
		if errors.Is(err, ErrUserNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		if found == nil {
			found = &user
		} else if found.ID != user.ID {
			return nil, ErrMultipleUsers
		}
	}

	return found, nil
}

func (s *UserService) upsertEmails(ctx context.Context, userID uuid.UUID, emails []EmailAddress) (bool, error) {
	anyCreated := false
	for _, email := range emails {
		_, created, err := s.UpsertIdentifier(
			ctx, userID, IdentifierEmail, email.Email,
			map[string]any{}, email.Verified, email.Primary,
		)
		if err != nil {
			return false, err
		}
		anyCreated = anyCreated || created
	}
	return anyCreated, nil
}

// GetVerifiedEmails returns all verified email addresses for a user
func (s *UserService) GetVerifiedEmails(ctx context.Context, userID uuid.UUID) ([]string, error) {
	identifiers, err := s.queries.GetVerifiedEmails(ctx, userID)
	if err != nil {
		return nil, err
	}

	emails := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		// Extract email from "email:foo@bar.com" format
		if len(id.Value) > 6 {
			emails = append(emails, id.Value[6:])
		}
	}
	return emails, nil
}

// Helpers

func filterVerifiedEmails(emails []EmailAddress) []EmailAddress {
	var verified []EmailAddress
	for _, e := range emails {
		if e.Verified {
			verified = append(verified, e)
		}
	}
	return verified
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func toNullString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func stringFromNull(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func toJSONB(data map[string]any) []byte {
	if data == nil {
		return []byte("{}")
	}
	b, _ := json.Marshal(data)
	return b
}

func fromJSONB(data []byte) map[string]any {
	if len(data) == 0 {
		return nil
	}
	var m map[string]any
	json.Unmarshal(data, &m)
	return m
}

// UpdateProfile updates the user's name and avatar URL
// Returns the updated user and any error
func (s *UserService) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	name string,
) (db.User, error) {
	logger := log.Ctx(ctx)
	// Update user record
	err := s.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:   userID,
		Name: db.Text(name),
	})
	if err != nil {
		logger.Info().Msgf("Error updating profile %s", err)
		return db.User{}, err
	}

	// Return updated user
	return s.FindByID(ctx, userID)
}
