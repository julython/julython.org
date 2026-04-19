// Create a new game and manage existing games

package main

import (
	"context"
	"flag"
	"july/internal/config"
	"july/internal/db"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"july/internal/services"
)

func main() {
	start := flag.String("start", "", "start of the game")
	inActive := flag.Bool("inactive", false, "make the game inactive")
	deactivateOthers := flag.Bool("deactivate-others", false, "mark other games as inactive")
	dryRun := flag.Bool("dry-run", false, "print changes without writing files")
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	cfg, err := config.Load()
	if err != nil {
		log.Error().Err(err).Msg("boo")
	}

	if *start == "" {
		*start = time.Now().Format("2006-01-02")
	}

	startTime, err := time.Parse(time.DateOnly, *start)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse time")
	}

	log.Info().
		Str("database", cfg.Database.Host).
		Str("date", startTime.Format("2006-01")).
		Bool("dry", *dryRun).
		Msg("Creating game for Julython")

	if !*dryRun {
		// Database
		ctx := context.Background()
		poolCfg, err := pgxpool.ParseConfig(cfg.Database.DSN())
		if err != nil {
			log.Fatal().Err(err).Msg("failed to parse database config")
		}
		poolCfg.MaxConns = int32(cfg.Database.MaxConns)
		poolCfg.MinConns = int32(cfg.Database.MinConns)

		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer pool.Close()

		if err := pool.Ping(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to ping database")
		}

		// Services
		queries := db.New(pool)
		gameSvc := services.NewGameService(queries)

		game, err := gameSvc.CreateJulythonGame(ctx, startTime.Year(), int(startTime.Month()), !*inActive, *deactivateOthers)

		log.Info().
			Str("game", game.Name).
			Str("start", game.StartsAt.Format("2006-01-02")).
			Str("end", game.EndsAt.Format("2006-01-02")).
			Msgf("Julython game created %s", game.Name)
	}

}
