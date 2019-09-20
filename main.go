package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger()
)

func main() {
	logger.Info().Msg("Loading configuration for notarization")
	config, err := loadConfiguration()
	switch {
	case err == errBadFlags:
		logger.Error().Msg("Some required flags are missing, please see usage for more help:")
		fmt.Println()

		flag.Usage()
		return

	case err != nil:
		logger.Fatal().Err(err).Msg("Encountered error while parsing configuration")
	}

	var wait sync.WaitGroup
	workers := make([]*Worker, len(config.Packages))
	for i, p := range config.Packages {
		wLogger := logger.With().Str("file", p.File).Logger()
		wLogger.Info().Str("file", p.File).Msg("Spawning notarization worker")

		worker, err := NewWorker(config, p, &wait, wLogger)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to spawn notarization worker")
			continue
		}

		workers[i] = worker
	}

	wait.Wait()
	logger.Info().Msg("Notarization completed, saving log files")

	for _, worker := range workers {
		if worker.log == nil {
			worker.logger.Error().Msg("No notarization log file available to save")
			continue
		}

		if path, err := worker.saveLog(); err != nil {
			worker.logger.Error().Err(err).Msg("Encountered error while saving notarization log")
		} else {
			worker.logger.Info().Str("log-file", path).Msg("Saved notarization log")
		}
	}
}