package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"july/internal/config"
	"july/internal/testutil"
)

func TestSessionCookieDomain(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	t.Run("production sets cookie domain", func(t *testing.T) {
		prodCfg := config.Session{
			CookieName:   "july_session",
			CookieDomain: ".julython.org",
		}
		sm := testutil.NewTestSessionManager(t, env.Pool, prodCfg, true)
		assert.Equal(t, ".julython.org", sm.Cookie.Domain, "cookie domain should be set in production")
		assert.True(t, sm.Cookie.Secure, "cookie should be Secure in production")
	})

	t.Run("development leaves domain empty", func(t *testing.T) {
		devCfg := config.Session{
			CookieName: "july_session",
		}
		sm := testutil.NewTestSessionManager(t, env.Pool, devCfg, false)
		assert.Equal(t, "", sm.Cookie.Domain, "cookie domain should be empty in development")
		assert.False(t, sm.Cookie.Secure, "cookie should not be Secure in development")
	})

	t.Run("production with empty domain leaves it blank", func(t *testing.T) {
		emptyCfg := config.Session{
			CookieName: "july_session",
		}
		sm := testutil.NewTestSessionManager(t, env.Pool, emptyCfg, true)
		assert.Equal(t, "", sm.Cookie.Domain, "empty domain should not be set even in production")
		assert.True(t, sm.Cookie.Secure, "cookie should be Secure in production")
	})
}
