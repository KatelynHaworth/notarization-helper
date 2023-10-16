package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const (
	infoWaitTime = 10 * time.Second
)

var (
	fileNameEscapeRegexp = regexp.MustCompile(`[/: ]+`)
)

type Worker struct {
	config  *Configuration
	target  Package
	logger  zerolog.Logger
	zipFile string
	wait    *sync.WaitGroup
	info    *NotarizationInfo
	log     *NotarizationLog
}

func NewWorker(config *Configuration, p Package, wait *sync.WaitGroup, logger zerolog.Logger) (*Worker, error) {
	worker := &Worker{
		config: config,
		target: p,
		logger: logger,
		wait:   wait,
	}

	stat, err := os.Stat(worker.target.File)
	switch {
	case err != nil && os.IsNotExist(err):
		return nil, errors.Wrap(err, "package file doesn't exist")

	case err != nil:
		return nil, errors.Wrap(err, "stat package file")

	case stat.IsDir() || !worker.allowedFileExtension():
		if err := worker.zipPackageFile(stat.IsDir()); err != nil {
			return nil, errors.Wrap(err, "create temporary ZIP for package")
		}
	}

	wait.Add(1)
	go worker.uploadAndWait()
	return worker, nil
}

func (worker *Worker) allowedFileExtension() bool {
	switch filepath.Ext(worker.target.File) {
	case ".zip":
		fallthrough
	case ".dmg":
		fallthrough
	case ".pkg":
		return true

	default:
		return false
	}
}

func (worker *Worker) zipPackageFile(isDir bool) error {
	escapedFileName := fileNameEscapeRegexp.ReplaceAllString(filepath.Base(worker.target.File), "")

	zipFile, err := ioutil.TempFile("", fmt.Sprintf("*-%s.zip", escapedFileName))
	if err != nil {
		return errors.Wrap(err, "open temporary file for zip")
	}

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if isDir {
		err = filepath.Walk(worker.target.File, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}

			if err := worker.addFileToZIP(zipWriter, info, path, true); err != nil {
				return errors.Wrap(err, "add file to ZIP")
			}

			return nil
		})
	} else {
		err = worker.addFileToZIP(zipWriter, nil, worker.target.File, false)
	}

	switch {
	case err != nil:
		return errors.Wrap(err, "walk target directory")

	default:
		worker.zipFile = zipFile.Name()
		return nil
	}
}

func (worker *Worker) addFileToZIP(writer *zip.Writer, info os.FileInfo, path string, useRel bool) (err error) {
	if info == nil {
		info, err = os.Stat(path)
		if err != nil {
			return errors.Wrap(err, "stat source file")
		}
	}

	fileHeader, _ := zip.FileInfoHeader(info)
	if useRel {
		fileHeader.Name, _ = filepath.Rel(filepath.Dir(worker.target.File), path)
	} else {
		fileHeader.Name = filepath.Base(path)
	}

	fileWriter, err := writer.CreateHeader(fileHeader)
	if err != nil {
		return errors.Wrap(err, "create ZIP entry for file")
	}

	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return errors.Wrap(err, "read symlink target")
		}

		_, err = fileWriter.Write([]byte(filepath.ToSlash(linkTarget)))
		if err != nil {
			return errors.Wrap(err, "write ZIP file entry for symlink target")
		}
	} else {
		sourceFile, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return errors.Wrap(err, "open source file for reading")
		}
		defer sourceFile.Close()

		if _, err := io.Copy(fileWriter, sourceFile); err != nil {
			return errors.Wrap(err, "write ZIP entry data from file")
		}
	}

	return nil
}

func (worker *Worker) saveLog() (string, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("%s.notarization-log", worker.target.File), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", errors.Wrap(err, "open log file")
	}
	defer logFile.Close()

	if err := json.NewEncoder(logFile).Encode(worker.log); err != nil {
		return "", errors.Wrap(err, "write log to file")
	}

	return logFile.Name(), nil
}
