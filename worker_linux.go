package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

func (worker *Worker) uploadAndWait() {
	defer worker.wait.Done()

	worker.logger.Info().Msg("Uploading package to Apple notarization service")
	upload, err := worker.uploadForNotarization()
	if err != nil {
		worker.logger.Error().Err(err).Msg("Upload to Apple failed")
		return
	}

	worker.logger.Info().Str("request-id", upload.RequestUUID).Msg("Successfully uploaded package, waiting for notarization to complete")

waitLoop:
	for {
		info, err := worker.getNotarizationStatus(upload)
		switch {
		case err != nil && errors.Cause(err).Error() == "Could not find the RequestUUID.":
			time.Sleep(infoWaitTime)

		case err != nil:
			worker.logger.Error().Err(err).Msg("Encountered error getting notarization status from Apple")
			return

		case info.Status == "in progress":
			time.Sleep(infoWaitTime)

		case info.Status == "invalid":
			worker.info = info
			worker.logger.Error().Msg("Notarization failed, please see the log file for more information")
			break waitLoop

		default:
			worker.logger.Info().Msg("Notarization completed successfully")
			worker.info = info
			break waitLoop
		}
	}

	if worker.info.Status != "invalid" {
		if err := worker.staplePackage(); err != nil {
			worker.logger.Error().Err(err).Msg("Failed to staple notarization to package")
		}
	}

	worker.logger.Info().Msg("Retrieving notarization log from Apple")
	if err := worker.downloadNotarizationLog(); err != nil {
		worker.logger.Error().Err(err).Msg("Failed to download notarization log")
	}

	worker.logger.Info().Msg("Notarization finished")
}

func (worker *Worker) downloadNotarizationLog() error {
	req, _ := http.NewRequest(http.MethodGet, worker.info.LogFileURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request notarization log")
	}
	defer resp.Body.Close()

	worker.log = new(NotarizationLog)
	if err := json.NewDecoder(resp.Body).Decode(worker.log); err != nil {
		return errors.Wrap(err, "unmarshal notarization log")
	}

	return nil
}
