package help_test

import (
	"net/http"
	"testing"

	"july/internal/testutil"

	"github.com/stretchr/testify/assert"
)

func TestHelpPages(t *testing.T) {
	t.Run("privacy page renders", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateTestScenario(t, env)

		resp := testutil.GetJSON(t, env, "/privacy")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("about page renders", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateTestScenario(t, env)

		resp := testutil.GetJSON(t, env, "/about")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("about page renders", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		testutil.CreateTestScenario(t, env)

		resp := testutil.GetJSON(t, env, "/help")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("help page renders for logged in users", func(t *testing.T) {
		env := testutil.SetupTestEnv(t)
		user := testutil.CreateUser(t, env, "testuser", "Test User")
		testutil.CreateUserIdentifier(t, env, user.ID, "email", "test@example.com", true, true)
		testutil.CreateUserIdentifier(t, env, user.ID, "github", "12345", true, false)
		env.LoginAs(t, "test@example.com")

		resp := testutil.GetJSON(t, env, "/help")

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
