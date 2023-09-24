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

func (worker *Worker) uploadForNotarization() (*NotaryToolInfo, error) {
	if len(worker.zipFile) > 0 {
		defer os.Remove(worker.zipFile)
	}

	cmd := exec.Command("xcrun", "notarytool", "submit")
	cmd.Args = append(cmd.Args, []string{
		"--output-format", "plist",
		"--apple-id", worker.config.Username,
		"--keychain-profile", worker.config.Password, // TODO: add new field "profile" and replace "password" in Config
		"--wait",
	}...)
	if len(worker.config.TeamID) > 0 {
		cmd.Args = append(cmd.Args, []string{
			"--team-id", worker.config.TeamID,
		}...)
	}

	if len(worker.zipFile) > 0 {
		cmd.Args = append(cmd.Args, worker.zipFile)
	} else {
		cmd.Args = append(cmd.Args, worker.target.File)
	}

	stdOut, err := cmd.Output()
	output := new(NotaryToolInfo)
	if _, err := plist.Unmarshal(stdOut, output); err != nil {
		return nil, errors.Wrap(err, "unmarshal command output")
	}

	switch {
	case err != nil:
		return nil, errors.Wrap(err, "execute notarytool submit")

	default:
		return output, nil
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
