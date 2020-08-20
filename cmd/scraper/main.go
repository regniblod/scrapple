package main

import (
	"fmt"
	"net/http"
	"os"

	ihttp "github.com/regniblod/boxr/internal/http"
	"github.com/regniblod/boxr/internal/scrap"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/json"

	"github.com/dgraph-io/badger/v2"
)

func main() {
	if err := run(); err != nil {
		log.WithError(err).Error("error in main")
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	logger := configureLogging()

	db, err := configureDatabase(logger)
	if err != nil {
		return err
	}

	// locales := []string{"es", "fr", "uk", "de", "it"}
	// categories := []string{"ipad", "iphone", "mac", "ipod", "appletv", "accessories", "clearance"}
	locales := []string{"es"}
	categories := []string{"ipad"}
	scraper := scrap.NewScraper(logger, db, ihttp.NewURLGetter(*http.DefaultClient))
	products := scraper.Scrap(locales, categories)
	logger.WithField("total_products", len(products)).Info("finished scraping")

	return nil
}

func configureLogging() log.Interface {
	if os.Getenv("APP_ENV") == "dev" {
		log.SetHandler(cli.New(os.Stdout))
	} else {
		log.SetHandler(json.New(os.Stdout))
	}

	log.Log.Info("logging loaded")

	return log.Log
}

func configureDatabase(logger log.Interface) (*badger.DB, error) {
	opts := badger.DefaultOptions("/tmp/badger")
	opts.Logger = &BadgeLogger{logger.(*log.Logger)}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("opening badger db. %w", err)
	}
	defer db.Close()

	return db, nil
}
