package main

import (
	"fmt"
	"os"

	"github.com/regniblod/boxr/internal/scrap"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	jsonlog "github.com/apex/log/handlers/json"

	"github.com/dgraph-io/badger/v2"
)

func main() {
	if err := run(); err != nil {
		log.WithError(err).Error("error in main")
		os.Exit(1)
	}
}

func run() error {
	logger := configureLogging()

	db, err := configureDatabase(logger)
	if err != nil {
		return err
	}

	scraper := scrap.NewScraper(logger, db)
	scraper.Scrap([]string{"https://www.apple.com/es/shop/refurbished/ipad"})
	//logger.Info("application initialized")

	return nil
}

func configureLogging() log.Interface {
	if os.Getenv("APP_ENV") == "dev" {
		log.SetHandler(cli.New(os.Stdout))
	} else {
		log.SetHandler(jsonlog.New(os.Stdout))
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
