package code_directory

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	SupportsVersionCodeLimit64 = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020300,
		FullName:             "CODEDIRECTORY_SUPPORTS_CODELIMIT64",
		ShortName:            "code_limit_64",
		Decoder:              decodeCodeLimit64,
		EncodeSizeCalculator: calculateCodeLimit64Size,
		HeaderEncoder:        encodeCodeLimit64Header,
	})
)

type rawCodeLimit64 struct {
	_       uint32 // reserved
	Limit64 uint64
}

func decodeCodeLimit64(_ *CodeDirectory, src *io.SectionReader) (any, error) {
	var raw rawCodeLimit64
	if err := binary.Read(src, binary.BigEndian, &raw); err != nil {
		return nil, fmt.Errorf("read raw: %w", err)
	}

	return raw.Limit64, nil
}

func calculateCodeLimit64Size(data any, _ *CodeDirectory) (uint32, uint32, error) {
	if data != nil {
		if _, ok := data.(uint64); !ok {
			return 0, 0, fmt.Errorf("unexpected data type %T", data)
		}
	}

	return uint32(binary.Size(rawCodeLimit64{})), 0, nil
}

func encodeCodeLimit64Header(data any, _ *CodeDirectory, dst io.Writer, _, _ uint32) (int64, error) {
	var raw rawCodeLimit64

	if data != nil {
		// Data type validation already done in calculateCodeLimit64Size
		raw.Limit64 = data.(uint64)
	}

	if err := binary.Write(dst, binary.BigEndian, raw); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(raw)), nil
}
