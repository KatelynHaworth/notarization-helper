package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"howett.net/plist"
)

func (worker *Worker) canStaple() bool {
	stat, _ := os.Stat(worker.target.File)
	switch filepath.Ext(worker.target.File) {
	case ".dmg":
		fallthrough
	case ".pkg":
		return true

	case ".kext":
		fallthrough
	case ".app":
		return stat.IsDir()

	default:
		return false
	}
}

func (worker *Worker) uploadForNotarization() (*NotarizationUpload, error) {
	if len(worker.zipFile) > 0 {
		defer os.Remove(worker.zipFile)
	}

	cmd := exec.Command("xcrun", "altool", "--notarize-app")
	cmd.Args = append(cmd.Args, []string{
		"--output-format", "xml",
		"--primary-bundle-id", worker.target.BundleID,
		"--username", worker.config.Username,
		"--password", worker.config.Password,
	}...)

	if len(worker.config.TeamID) > 0 {
		cmd.Args = append(cmd.Args, []string{
			"-itc_provider", worker.config.TeamID,
		}...)
	}

	if len(worker.zipFile) > 0 {
		cmd.Args = append(cmd.Args, []string{
			"--file", worker.zipFile,
		}...)
	} else {
		cmd.Args = append(cmd.Args, []string{
			"--file", worker.target.File,
		}...)
	}

	stdOut, err := cmd.Output()
	output := new(CommandOutput)
	if _, err := plist.Unmarshal(stdOut, output); err != nil {
		return nil, errors.Wrap(err, "unmarshal command output")
	}

	switch {
	case len(output.ProductErrors) > 0:
		return nil, errors.Wrap(output.ProductErrors[0], "execute altool")

	case err != nil:
		return nil, errors.Wrap(err, "execute altool")

	default:
		return &output.Upload, nil
	}
}

func (worker *Worker) getNotarizationStatus(upload *NotarizationUpload) (*NotarizationInfo, error) {
	cmd := exec.Command("xcrun", "altool", "--notarization-info", upload.RequestUUID, "--username", worker.config.Username, "--password", worker.config.Password, "--output-format", "xml")

	stdOut, err := cmd.Output()
	output := new(CommandOutput)
	if _, err := plist.Unmarshal(stdOut, output); err != nil {
		return nil, errors.Wrap(err, "unmarshal command output")
	}

	switch {
	case len(output.ProductErrors) > 0:
		return nil, errors.Wrap(output.ProductErrors[0], "execute altool")

	case err != nil:
		return nil, errors.Wrap(err, "execute altool")

	default:
		return &output.Info, nil
	}
}

func (worker *Worker) staplePackage() error {
	switch {
	case !worker.target.Staple:
		return nil

	case !worker.canStaple():
		worker.logger.Warn().Msg("This file type is not supported for stapling, continuing without stapling")
		return nil

	default:
		worker.logger.Info().Msg("Stapling notarization ticket to package")

		cmd := exec.Command("xcrun", "stapler", "staple", worker.target.File)
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "execute stapler")
		}

		return nil
	}
}
