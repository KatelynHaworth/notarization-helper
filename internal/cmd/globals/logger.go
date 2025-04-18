package globals

import (
	"time"

	"github.com/rs/zerolog"
)

var (
	Logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger()
)
