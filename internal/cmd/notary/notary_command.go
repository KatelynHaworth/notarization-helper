package notary

import (
	. "github.com/KatelynHaworth/notarization-helper/v2/internal/cmd/globals"
	"github.com/KatelynHaworth/notarization-helper/v2/notarize/worker"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	NotarizeCmd = &cobra.Command{
		Use:   "notarize",
		Short: "Upload files to Apple for notarization",
		RunE:  run,
	}
)

func run(cmd *cobra.Command, _ []string) error {
	group, gCtx := errgroup.WithContext(cmd.Context())
	wkrs := make([]*worker.Worker, len(Config.GetPackages()))

	for i, p := range Config.GetPackages() {
		wLogger := Logger.With().Str("file", p.File).Logger()
		wLogger.Info().Str("file", p.File).Msg("Spawning notarization worker")

		wkr, err := worker.NewWorker(Config.NotaryAuth, p, wLogger)
		if err != nil {
			wLogger.Error().Err(err).Msg("Failed to spawn notarization worker")
			continue
		}

		wkrs[i] = wkr
		group.Go(func() error {
			return wkr.UploadAndWait(gCtx)
		})
	}

	if err := group.Wait(); err != nil {
		Logger.Error().Err(err).Msg("One or more notarization workers failed")
	} else {
		Logger.Info().Msg("Notarization completed, saving log files")
	}

	for _, wkr := range wkrs {
		wLogger := wkr.Logger()
		if wkr.GetNotarizationLog() == nil {
			wLogger.Error().Msg("No notarization log file available to save")
			continue
		}

		if path, err := wkr.SaveNotarizationLog(); err != nil {
			wLogger.Error().Err(err).Msg("Encountered error while saving notarization log")
		} else {
			wLogger.Info().Str("log-file", path).Msg("Saved notarization log")
		}
	}

	return nil
}
