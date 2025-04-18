package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/KatelynHaworth/notarization-helper/v2/config"
	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
	"github.com/rs/zerolog"
)

var (
	fileNameEscapeRegexp = regexp.MustCompile(`[/: ]+`)

	allowedFileExtensions = []string{
		".pkg",
		".dmg",
		".zip",
	}
)

type Worker struct {
	auth   *config.ConfigurationV2_NotaryAuth
	target config.Package
	logger zerolog.Logger

	zipFile         string
	uploadFileHash  string
	submissionId    string
	notarizationLog *api.NotarizationLog
}

func NewWorker(auth *config.ConfigurationV2_NotaryAuth, p config.Package, logger zerolog.Logger) (*Worker, error) {
	worker := &Worker{
		auth:   auth,
		target: p,
		logger: logger,
	}

	stat, err := os.Stat(worker.target.File)
	switch {
	case err != nil && os.IsNotExist(err):
		return nil, fmt.Errorf("package file doesn't exist: %w", err)

	case err != nil:
		return nil, fmt.Errorf("stat package file: %w", err)

	case stat.IsDir() || !worker.allowedFileExtension():
		if err = worker.zipPackageFile(stat.IsDir()); err != nil {
			return nil, fmt.Errorf("create temporary ZIP for package: %w", err)
		}
	}

	if worker.uploadFileHash, err = worker.getFileHash(sha256.New()); err != nil {
		return nil, fmt.Errorf("hash file: %w", err)
	}

	return worker, nil
}

func (worker *Worker) Logger() zerolog.Logger {
	return worker.logger
}

func (worker *Worker) GetNotarizationLog() *api.NotarizationLog {
	return worker.notarizationLog
}

func (worker *Worker) SaveNotarizationLog() (string, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("%s.notarization-log", worker.target.File), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	if err := json.NewEncoder(logFile).Encode(worker.notarizationLog); err != nil {
		return "", fmt.Errorf("write log to file: %w", err)
	}

	return logFile.Name(), nil
}

func (worker *Worker) allowedFileExtension() bool {
	return slices.Contains(allowedFileExtensions, filepath.Ext(worker.target.File))
}

func (worker *Worker) getTargetFile() string {
	if len(worker.zipFile) > 0 {
		return worker.zipFile
	}

	return worker.target.File
}

func (worker *Worker) getFileHash(hasher hash.Hash) (string, error) {
	src, err := os.OpenFile(worker.getTargetFile(), os.O_RDONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("open file to hash: %w", err)
	}
	defer src.Close()

	if _, err = io.Copy(hasher, src); err != nil {
		return "", fmt.Errorf("stream file into hasher: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
