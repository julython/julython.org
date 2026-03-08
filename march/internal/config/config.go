package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Config struct {
	Env      string   `mapstructure:"env"`
	Server   Server   `mapstructure:"server"`
	Database Database `mapstructure:"database"`
	Session  Session  `mapstructure:"session"`
	OAuth    OAuth    `mapstructure:"oauth"`
	Webhooks Webhooks `mapstructure:"webhooks"`
}

type Server struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int    `mapstructure:"max_conns"`
	MinConns int    `mapstructure:"min_conns"`
}

func (d Database) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type Session struct {
	CookieName      string        `mapstructure:"cookie_name"`
	Lifetime        time.Duration `mapstructure:"lifetime"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

type OAuth struct {
	BaseURL string        `mapstructure:"base_url"`
	GitHub  OAuthProvider `mapstructure:"github"`
	GitLab  OAuthProvider `mapstructure:"gitlab"`
}

func (o OAuth) CallbackURL() string {
	return o.BaseURL + "/auth/callback"
}

type OAuthProvider struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	Enabled      bool   `mapstructure:"enabled"`
}

type Webhooks struct {
	GitHub    string `mapstructure:"github"`
	GitLab    string `mapstructure:"gitlab"`
	BitBucket string `mapstructure:"bitbucket"`
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func (c Config) IsDevelopment() bool {
	return c.Env == "development" || c.Env == ""
}

// Load reads configuration with the following precedence (highest to lowest):
// 1. Environment variables (JULY_*)
// 2. config.{env}.yaml (e.g., config.production.yaml)
// 3. config.yaml (base config)
// 4. Default values
func Load() (*Config, error) {
	env := os.Getenv("JULY_ENV")
	if env == "" {
		env = "development"
	}

	v := viper.New()
	v.SetConfigType("yaml")

	// Set all defaults first (required for env var binding)
	setDefaults(v)

	// Load base config (config.yaml)
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading base config: %w", err)
		}
	} else {
		log.Debug().Str("file", v.ConfigFileUsed()).Msg("loaded base config")
	}

	// Merge environment-specific config (config.{env}.yaml)
	v.SetConfigName("config." + env)
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading %s config: %w", env, err)
		}
	} else {
		log.Debug().Str("file", v.ConfigFileUsed()).Msg("merged environment config")
	}

	// Environment variables override everything
	v.SetEnvPrefix("JULY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.Set("env", env)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Environment
	v.SetDefault("env", "development")

	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8000)

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "julython")
	v.SetDefault("database.password", "")
	v.SetDefault("database.name", "julython")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_conns", 25)
	v.SetDefault("database.min_conns", 5)

	// Session
	v.SetDefault("session.cookie_name", "july_session")
	v.SetDefault("session.lifetime", "168h")
	v.SetDefault("session.cleanup_interval", "15m")

	// OAuth
	v.SetDefault("oauth.base_url", "http://localhost:8000")
	v.SetDefault("oauth.github.client_id", "")
	v.SetDefault("oauth.github.client_secret", "")
	v.SetDefault("oauth.github.enabled", true)
	v.SetDefault("oauth.gitlab.client_id", "")
	v.SetDefault("oauth.gitlab.client_secret", "")
	v.SetDefault("oauth.gitlab.enabled", false)

	// Webhooks
	v.SetDefault("webhooks.github", "https://julython.org/api/v1/github")
	v.SetDefault("webhooks.gitlab", "https://julython.org/api/v1/gitlab")
	v.SetDefault("webhooks.bitbucket", "https://julython.org/api/v1/bitbucket")
}
