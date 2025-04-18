package codesign

import (
	"fmt"
	"io"
	"os"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/dmg"
)

func ReadFromDMG[T Blob](srcFile *os.File) (T, error) {
	var zero T

	trailer, _, err := dmg.ReadUDIF(srcFile)
	if err != nil {
		return zero, fmt.Errorf("read UDIF: %w", err)
	}

	return ReadFrom[T](io.NewSectionReader(srcFile, int64(trailer.CodeSignOffset), int64(trailer.CodeSignLength)))
}

func WriteToDMG(blob Blob, dstFile *os.File) error {
	blobTemp, blobSize, err := WriteToTemp(blob)
	if err != nil {
		return err
	}
	defer blobTemp.Close()

	trailer, _, err := dmg.ReadUDIF(dstFile)
	if err != nil {
		return fmt.Errorf("read existing UDIF: %w", err)
	}

	if err = dstFile.Truncate(int64(trailer.CodeSignOffset)); err != nil {
		return fmt.Errorf("strip existing code signature and UDIF: %w", err)
	}

	if _, err = io.Copy(dstFile, blobTemp); err != nil {
		return fmt.Errorf("copy codesign blob to dest file: %w", err)
	}

	trailer.CodeSignLength = uint32(blobSize)
	if err = dmg.WriteUDIF(trailer, dstFile); err != nil {
		return fmt.Errorf("write existing UDIF: %w", err)
	}

	return nil
}
