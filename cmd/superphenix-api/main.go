package main

import (
	"context"
	l "log"
	"os"
	"time"

	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/api"
	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/app"
	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/config"
	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/metrics"
	"github.com/super-phenix/superphenix/internal/superphenix-api/pkg/opentelemetry/tracing"

	spxId "github.com/super-phenix/superphenix/pkg/superphenix-id"
	"github.com/super-phenix/superphenix/pkg/utils/secret"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer cleanup(ctx)

	loadConfig()
	startLogging()
	checkAZAuthSecrets()
	startTracing()
	startMetrics()
	spxId.SetFrameworkPrefix(config.Global.SpxPrefix)
	app.StartGarbageCollection(ctx, &config.Global)
	app.StartAuditLogGC(ctx, &config.Global)

	api.StartAPI()
}

func loadConfig() {
	if err := config.LoadConfig(); err != nil {
		if err == config.FileNotFound {
			l.Print("Config file not found, falling back to environment variables and defaults")
		} else {
			l.Fatalf("An error occured while loading config file: %v", err)
		}
	}
}

// checkAZAuthSecrets refuses to start when an AZ controller secret is empty,
// short or publicly known: the controller trusts any caller presenting it.
func checkAZAuthSecrets() {
	for code, az := range config.Global.AZs {
		if err := secret.Check(az.AuthSecret); err != nil {
			log.Fatal().Err(err).Str("az", code).Msg("Invalid AZ authSecret, set a random value of at least 16 characters (e.g. through the SUPERPHENIX-API_AZS_<AZ>_AUTHSECRET environment variable)")
		}
	}
}

func startLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if config.Global.Logging.Pretty {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Caller().Logger()
	} else {
		log.Logger = log.Logger.With().Timestamp().Caller().Logger()
	}

	log.Debug().Msg("Starting logging")
}

func startTracing() {
	if !config.Global.Tracing.Enabled {
		return
	}

	err := tracing.StartTracing()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start tracer")
	}

	log.Debug().Msg("Starting tracing")
}

func startMetrics() {
	if !config.Global.Metrics.Enabled {
		return
	}

	log.Debug().Msg("Starting collecting and exposing metrics")

	go func() {
		err := metrics.StartMetrics()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to start metric endpoint")
		}
	}()
}

// Post shutdown cleanup
func cleanup(ctx context.Context) {
	tracing.StopTracing(ctx)
}
