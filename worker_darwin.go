package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func (worker *Worker) uploadAndWait() {
	defer worker.wait.Done()

	worker.logger.Info().Msg("Uploading package to Apple notarization service")
	info, err := worker.uploadForNotarization()
	if err != nil {
		worker.logger.Error().Err(err).Msg("Upload to Apple failed")
		return
	}

	if strings.ToLower(info.Status) == "invalid" {
		worker.logger.Error().Msg("Notarization failed, please see the log file for more information")
	} else {
		worker.logger.Info().Msgf("Notarization completed successfully with status: %s", info.Status)
	}

	if strings.ToLower(info.Status) != "invalid" {
		if err := worker.staplePackage(); err != nil {
			worker.logger.Error().Err(err).Msg("Failed to staple notarization to package")
		}
	}

	worker.logger.Info().Msg("Retrieving notarization log from Apple")
	if err := worker.downloadNotarizationLog(info.RequestUUID); err != nil {
		worker.logger.Error().Err(err).Msg("Failed to download notarization log")
	}

	worker.logger.Info().Msg("Notarization finished")
}

func (worker *Worker) downloadNotarizationLog(requestUUID string) error {
	if len(requestUUID) == 0 {
		return fmt.Errorf("request uuid not found")
	}
	cmd := exec.Command("xcrun", "notarytool", "log")
	cmd.Args = append(cmd.Args, []string{
		"--apple-id", worker.config.Username,
		"--keychain-profile", worker.config.Password,
		requestUUID,
	}...)

	stdOut, err := cmd.Output()
	if err != nil {
		return errors.Wrap(err, "request notarization log")
	}

	worker.log = new(NotarizationLog)
	if err := json.Unmarshal(stdOut, worker.log); err != nil {
		return errors.Wrap(err, "unmarshal notarization log")
	}

	return nil
}
