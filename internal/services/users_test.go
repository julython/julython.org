package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"july/internal/db"
	"july/internal/services"
	"july/internal/testutil"
)

func TestOAuthLoginWebhookUser(t *testing.T) {
	env := testutil.SetupTestEnv(t)
	ctx := context.Background()

	t.Run("webhook-created user linked by OAuth", func(t *testing.T) {
		// 1. Create a webhook-created user (gh-prefixed username) WITHOUT email identifier.
		// This ensures the OAuth flow finds them by step 2b (username lookup),
		// NOT by step 2 (email lookup).
		webhookUser, err := env.Queries.CreateUser(ctx, db.CreateUserParams{
			ID:       db.NewID(),
			Username: "gh-testwebhook",
			Name:     "Webhook User",
			Role:     "user",
		})
		require.NoError(t, err)

		// 2. Verify the user was created with the correct username
		lookupUser, err := env.Queries.GetUserByUsername(ctx, "gh-testwebhook")
		require.NoError(t, err)
		assert.Equal(t, "gh-testwebhook", lookupUser.Username)
		assert.Equal(t, "Webhook User", lookupUser.Name)

		// 3. Simulate OAuth login from a GitHub OAuth provider.
		// This user has a username matching the webhook user, but NO email match.
		oauthUser := services.OAuthUser{
			Provider:  "github",
			Username:  "gh-testwebhook",
			ID:        "12345",
			Name:      "Test User OAuth",
			AvatarURL: "https://github.com/testwebhook.png",
			Emails: []services.EmailAddress{
				{Email: "different@example.com", Primary: true, Verified: true},
			},
			Data: map[string]any{
				"access_token": "oauth-token-123",
				"scope":        "user:email",
			},
		}

		// 4. Call OAuthLoginOrRegister — should find the webhook user by username (step 2b),
		// not create a new user, not find by email
		resultUser, created, err := env.UserService.OAuthLoginOrRegister(ctx, oauthUser)
		require.NoError(t, err)

		// 5. Verify: the existing webhook user was returned, not a new user
		assert.False(t, created, "should not create a new user when webhook user exists")
		assert.Equal(t, webhookUser.ID, resultUser.ID, "should link to existing webhook user")

		// 6. Verify: GitHub identifier was added
		ghID, err := env.Queries.GetUserIdentifierByUserAndType(ctx, db.GetUserIdentifierByUserAndTypeParams{
			UserID: webhookUser.ID,
			Type:   "github",
		})
		require.NoError(t, err)
		assert.Equal(t, "github:12345", ghID.Value, "should have github identifier")

		// 7. Verify: name was updated from OAuth data
		actualUser, err := env.Queries.GetUserByID(ctx, resultUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test User OAuth", actualUser.Name, "name should be updated from OAuth data")
	})

	t.Run("new OAuth user creates fresh account", func(t *testing.T) {
		// A user who has never committed — fresh OAuth signup
		oauthUser := services.OAuthUser{
			Provider:  "github",
			Username:  "gh-freshuser",
			ID:        "99999",
			Name:      "Fresh User",
			AvatarURL: "https://github.com/freshuser.png",
			Emails: []services.EmailAddress{
				{Email: "fresh@example.com", Primary: true, Verified: true},
			},
			Data: map[string]any{
				"access_token": "oauth-token-456",
				"scope":        "user:email",
			},
		}

		resultUser, created, err := env.UserService.OAuthLoginOrRegister(ctx, oauthUser)
		require.NoError(t, err)

		assert.True(t, created, "should create a new user for fresh OAuth")

		// Verify user has the right username
		assert.Equal(t, "gh-freshuser", resultUser.Username)

		// Verify GitHub identifier was added
		ghID, err := env.Queries.GetUserIdentifierByUserAndType(ctx, db.GetUserIdentifierByUserAndTypeParams{
			UserID: resultUser.ID,
			Type:   "github",
		})
		require.NoError(t, err)
		assert.Equal(t, "github:99999", ghID.Value)
	})
}
