package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Env       string   `env:"JULY_ENV" envDefault:"development"`
	ProjectID string   `env:"GOOGLE_CLOUD_PROJECT" envDefault:"julython-go"`
	Server    Server   `envPrefix:"JULY_SERVER_"`
	Database  Database `envPrefix:"JULY_DATABASE_"`
	Session   Session  `envPrefix:"JULY_SESSION_"`
	OAuth     OAuth    `envPrefix:"JULY_OAUTH_"`
	Webhooks  Webhooks `envPrefix:"JULY_WEBHOOK_"`
}

func (c Config) IsProduction() bool  { return c.Env == "production" }
func (c Config) IsDevelopment() bool { return c.Env == "development" || c.Env == "" }

type Server struct {
	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port int    `env:"PORT" envDefault:"8000"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Database struct {
	Host     string `env:"HOST"     envDefault:"localhost"`
	Port     int    `env:"PORT"     envDefault:"5432"`
	User     string `env:"USER"     envDefault:"julython"`
	Password string `env:"PASSWORD" envDefault:""`
	Name     string `env:"NAME"     envDefault:"julython"`
	SSLMode  string `env:"SSLMODE"  envDefault:"disable"`
	MaxConns int    `env:"MAX_CONNS" envDefault:"25"`
	MinConns int    `env:"MIN_CONNS" envDefault:"5"`
	EncKey   string `env:"ENC_KEY" envDefault:""`
}

func (d Database) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type Session struct {
	CookieName      string        `env:"COOKIE_NAME"      envDefault:"july_session"`
	Lifetime        time.Duration `env:"LIFETIME"         envDefault:"168h"`
	CleanupInterval time.Duration `env:"CLEANUP_INTERVAL" envDefault:"15m"`
}

type OAuth struct {
	BaseURL  string        `env:"BASE_URL" envDefault:"http://localhost:8000"`
	GitHub   OAuthProvider `envPrefix:"GITHUB_"`
	GitLab   OAuthProvider `envPrefix:"GITLAB_"`
	Password OAuthProvider `envPrefix:"PASSWORD_"`
}

func (o OAuth) CallbackURL() string {
	return o.BaseURL + "/auth/callback"
}

type OAuthProvider struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	Enabled      bool   `env:"ENABLED" envDefault:"true"`
}

type Webhooks struct {
	GitHub    string `env:"GITHUB"    envDefault:"https://julython.org/api/v1/github"`
	GitLab    string `env:"GITLAB"    envDefault:"https://julython.org/api/v1/gitlab"`
	BitBucket string `env:"BITBUCKET" envDefault:"https://julython.org/api/v1/bitbucket"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.Log()
	return cfg, nil
}

func mask(s string) string {
	if s == "" {
		return "(unset)"
	}
	return "***"
}

func (c Config) Log() {
	log.Info().
		// Top-level
		Str("env", c.Env).
		Str("projectId", c.ProjectID).
		// Server
		Str("server.host", c.Server.Host).
		Int("server.port", c.Server.Port).
		// Database
		Str("database.host", c.Database.Host).
		Int("database.port", c.Database.Port).
		Str("database.user", c.Database.User).
		Str("database.password", mask(c.Database.Password)).
		Str("database.name", c.Database.Name).
		Str("database.sslmode", c.Database.SSLMode).
		Int("database.max_conns", c.Database.MaxConns).
		Int("database.min_conns", c.Database.MinConns).
		// Session
		Str("session.cookie_name", c.Session.CookieName).
		Dur("session.lifetime", c.Session.Lifetime).
		Dur("session.cleanup_interval", c.Session.CleanupInterval).
		// OAuth
		Str("oauth.base_url", c.OAuth.BaseURL).
		Bool("oauth.github.enabled", c.OAuth.GitHub.Enabled).
		Str("oauth.github.client_id", mask(c.OAuth.GitHub.ClientID)).
		Str("oauth.github.client_secret", mask(c.OAuth.GitHub.ClientSecret)).
		Bool("oauth.gitlab.enabled", c.OAuth.GitLab.Enabled).
		Str("oauth.gitlab.client_id", mask(c.OAuth.GitLab.ClientID)).
		Str("oauth.gitlab.client_secret", mask(c.OAuth.GitLab.ClientSecret)).
		Bool("oauth.password.enabled", c.OAuth.Password.Enabled).
		// Webhooks
		Str("webhooks.github", c.Webhooks.GitHub).
		Str("webhooks.gitlab", c.Webhooks.GitLab).
		Str("webhooks.bitbucket", c.Webhooks.BitBucket).
		Msg("config loaded")
}
