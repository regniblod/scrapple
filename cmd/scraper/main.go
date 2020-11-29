package main

import (
	"expvar"
	"fmt"
	"net/http"
	"os"

	"github.com/ardanlabs/conf"
	ihttp "github.com/regniblod/scrapple/internal/http"
	"github.com/regniblod/scrapple/internal/scrap"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// build is the git version of this program. It is set using build flags in the makefile.
var build string = "develop"
var devEnv string = "dev"

type Config struct {
	Name     string `conf:""`
	Env      string `conf:"default:dev"`
	Database struct {
		Host     string `conf:"default:localhost"`
		Port     int    `conf:"default:5432"`
		User     string `conf:"default:app"`
		Password string `conf:"default:app,noprint"`
		Name     string `conf:"default:app"`
	}
	Scrapper struct {
		Locales    []string `conf:""`
		Categories []string `conf:""`
	}
}

func main() {
	if err := run(); err != nil {
		log.Error().Err(err).Msg("error in main")
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	logger := configureLogging()
	logger.Info().Str("build", build).Msg("initializing application")

	cfg, err := parseConfig(logger)
	if err != nil {
		return err
	}

	scraper := scrap.NewScraper(logger, ihttp.NewURLGetter(*http.DefaultClient))
	products := scraper.Scrap(cfg.Scrapper.Locales, cfg.Scrapper.Categories)
	logger.Info().Int("total_products", len(products)).Msg("finished scraping")

	return nil
}

func configureLogging() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var log zerolog.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	if os.Getenv("APP_ENV") == "dev" {
		log = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	log.Info().Msg("logging loaded")

	return log
}

func parseConfig(logger zerolog.Logger) (*Config, error) {
	cfg := &Config{}

	if err := conf.Parse(os.Args[1:], "app", cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage("", &cfg)
			if err != nil {
				return nil, fmt.Errorf("generating config usage: %w", err)
			}

			fmt.Println(usage)
			os.Exit(0)
		case conf.ErrVersionWanted:
			version, err := conf.VersionString("", cfg)
			if err != nil {
				return nil, fmt.Errorf("generating config version: %w", err)
			}

			fmt.Println(version)
			os.Exit(0)
		}

		return nil, fmt.Errorf("parsing config: %w", err)
	}

	expvar.NewString("build").Set(build)

	out, err := conf.String(cfg)
	if err != nil {
		return nil, fmt.Errorf("generating config for output: %w", err)
	}

	logger.Info().Msgf("loaded config:\n%s", out)

	return cfg, nil
}
