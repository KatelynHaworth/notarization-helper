package worker

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (worker *Worker) zipPackageFile(isDir bool) error {
	escapedFileName := fileNameEscapeRegexp.ReplaceAllString(filepath.Base(worker.target.File), "")

	zipFile, err := os.CreateTemp("", fmt.Sprintf("*-%s.zip", escapedFileName))
	if err != nil {
		return fmt.Errorf("open temporary file for zip: %w", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	if isDir {
		err = filepath.Walk(worker.target.File, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}

			if err = worker.addFileToZIP(zipWriter, info, path, true); err != nil {
				return fmt.Errorf("add file to ZIP: %w", err)
			}

			return nil
		})
	} else {
		err = worker.addFileToZIP(zipWriter, nil, worker.target.File, false)
	}

	switch {
	case err != nil:
		return fmt.Errorf("walk target directory: %w", err)

	default:
		worker.zipFile = zipFile.Name()
		return nil
	}
}

func (worker *Worker) addFileToZIP(writer *zip.Writer, info os.FileInfo, path string, useRel bool) error {
	if info == nil {
		var err error
		if info, err = os.Stat(path); err != nil {
			return fmt.Errorf("stat source file: %w", err)
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
		return fmt.Errorf("create ZIP entry for file: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("read symlink target: %w", err)
		}

		_, err = fileWriter.Write([]byte(filepath.ToSlash(linkTarget)))
		if err != nil {
			return fmt.Errorf("write ZIP file entry for symlink target: %w", err)
		}
	} else {
		sourceFile, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf("open source file for reading: %w", err)
		}
		defer sourceFile.Close()

		if _, err := io.Copy(fileWriter, sourceFile); err != nil {
			return fmt.Errorf("write ZIP entry data from file: %w", err)
		}
	}

	return nil
}
