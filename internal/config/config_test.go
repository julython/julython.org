package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- IsDevelopment ---

func TestIsDevelopment(t *testing.T) {
	t.Run("true for development", func(t *testing.T) {
		cfg := Config{Env: "development"}
		assert.True(t, cfg.IsDevelopment())
	})

	t.Run("true when Env is empty", func(t *testing.T) {
		cfg := Config{Env: ""}
		assert.True(t, cfg.IsDevelopment())
	})

	t.Run("false for production", func(t *testing.T) {
		cfg := Config{Env: "production"}
		assert.False(t, cfg.IsDevelopment())
	})

	t.Run("false for unknown env", func(t *testing.T) {
		cfg := Config{Env: "staging"}
		assert.False(t, cfg.IsDevelopment())
	})
}

// --- IsProduction ---

func TestIsProduction(t *testing.T) {
	t.Run("true for production", func(t *testing.T) {
		cfg := Config{Env: "production"}
		assert.True(t, cfg.IsProduction())
	})

	t.Run("false for development", func(t *testing.T) {
		cfg := Config{Env: "development"}
		assert.False(t, cfg.IsProduction())
	})

	t.Run("false when Env is empty", func(t *testing.T) {
		cfg := Config{Env: ""}
		assert.False(t, cfg.IsProduction())
	})
}

// --- Server.Addr ---

func TestServerAddr(t *testing.T) {
	t.Run("default host and port", func(t *testing.T) {
		s := Server{Host: "0.0.0.0", Port: 8000}
		assert.Equal(t, "0.0.0.0:8000", s.Addr())
	})

	t.Run("custom host and port", func(t *testing.T) {
		s := Server{Host: "127.0.0.1", Port: 3000}
		assert.Equal(t, "127.0.0.1:3000", s.Addr())
	})

	t.Run("custom port only", func(t *testing.T) {
		s := Server{Host: "0.0.0.0", Port: 9090}
		assert.Equal(t, "0.0.0.0:9090", s.Addr())
	})
}

// --- Database.DSN ---

func TestDatabaseDSN(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		d := Database{
			Host:     "localhost",
			Port:     5432,
			User:     "julython",
			Password: "",
			Name:     "julython",
			SSLMode:  "disable",
		}
		assert.Equal(t, "postgres://julython:@localhost:5432/julython?sslmode=disable", d.DSN())
	})

	t.Run("custom values", func(t *testing.T) {
		d := Database{
			Host:     "db.example.com",
			Port:     5433,
			User:     "admin",
			Password: "secret",
			Name:     "mydb",
			SSLMode:  "require",
		}
		assert.Equal(t, "postgres://admin:secret@db.example.com:5433/mydb?sslmode=require", d.DSN())
	})

	t.Run("empty password is valid", func(t *testing.T) {
		d := Database{Host: "localhost", Port: 5432, User: "julython", Name: "julython", SSLMode: "disable"}
		assert.Equal(t, "postgres://julython:@localhost:5432/julython?sslmode=disable", d.DSN())
	})
}

// --- OAuth.CallbackURL ---

func TestOAuthCallbackURL(t *testing.T) {
	t.Run("default callback URL", func(t *testing.T) {
		o := OAuth{BaseURL: "http://localhost:8000"}
		assert.Equal(t, "http://localhost:8000/auth/callback", o.CallbackURL())
	})

	t.Run("custom base URL", func(t *testing.T) {
		o := OAuth{BaseURL: "https://example.com"}
		assert.Equal(t, "https://example.com/auth/callback", o.CallbackURL())
	})
}

// --- mask ---

func TestMask(t *testing.T) {
	t.Run("empty string becomes unset", func(t *testing.T) {
		assert.Equal(t, "(unset)", mask(""))
	})

	t.Run("non-empty string becomes asterisks", func(t *testing.T) {
		assert.Equal(t, "***", mask("secret"))
	})

	t.Run("whitespace string becomes asterisks", func(t *testing.T) {
		assert.Equal(t, "***", mask(" "))
	})
}

// --- Webhooks defaults ---

func TestWebhooksDefaults(t *testing.T) {
	w := Webhooks{
		GitHub:    "https://julython.org/api/v1/github",
		GitLab:    "https://julython.org/api/v1/gitlab",
		BitBucket: "https://julython.org/api/v1/bitbucket",
	}
	assert.Equal(t, "https://julython.org/api/v1/github", w.GitHub)
	assert.Equal(t, "https://julython.org/api/v1/gitlab", w.GitLab)
	assert.Equal(t, "https://julython.org/api/v1/bitbucket", w.BitBucket)
}

// --- OAuthProvider defaults ---

func TestOAuthProviderDefaults(t *testing.T) {
	p := OAuthProvider{Enabled: true}
	assert.True(t, p.Enabled)
}
