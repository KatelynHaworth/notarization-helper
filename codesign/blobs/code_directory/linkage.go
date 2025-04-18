package code_directory

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/hash"
)

var (
	SupportsVersionLinkage = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020600,
		FullName:             "CODEDIRECTORY_SUPPORTS_LINKAGE",
		ShortName:            "linkage",
		Decoder:              decodeLinkage,
		EncodeSizeCalculator: calculateLinkageSize,
		HeaderEncoder:        encodeLinkageHeader,
		BodyEncoder:          encodeLinkageBody,
	})
)

type (
	Linkage struct {
		HashType           hash.Type
		ApplicationType    uint8
		ApplicationSubType uint16
		Data               []byte
	}

	rawLinkage struct {
		HashType           hash.Type
		ApplicationType    uint8
		ApplicationSubType uint16
		Offset             uint32
		Size               uint32
	}
)

func decodeLinkage(cd *CodeDirectory, src *io.SectionReader) (any, error) {
	var raw rawLinkage

	err := binary.Read(src, binary.BigEndian, &raw)
	switch {
	case err != nil:
		return nil, fmt.Errorf("read raw: %w", err)

	case raw.HashType == hash.TypeInvalid:
		return nil, nil

	case cd.hdr.Length < raw.Offset:
		return nil, fmt.Errorf("offset overflows blob length")

	case cd.hdr.Length < raw.Offset+raw.Size:
		return nil, fmt.Errorf("data size overflows blob length")
	}

	data := make([]byte, raw.Size)
	if _, err = src.ReadAt(data, int64(raw.Offset)); err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	return Linkage{
		HashType:           raw.HashType,
		ApplicationType:    raw.ApplicationType,
		ApplicationSubType: raw.ApplicationSubType,
		Data:               data,
	}, nil
}

func calculateLinkageSize(data any, _ *CodeDirectory) (hdr, body uint32, _ error) {
	hdr = uint32(binary.Size(rawLinkage{}))

	if data != nil {
		linkage, ok := data.(Linkage)
		if !ok {
			return 0, 0, fmt.Errorf("unexpected data type %T", data)
		}

		body = uint32(len(linkage.Data))
	}

	return
}

func encodeLinkageHeader(data any, _ *CodeDirectory, dst io.Writer, _, dataOffset uint32) (int64, error) {
	var raw rawLinkage

	if data != nil {
		// Data type validation already done in calculateRuntimeSize
		linkage := data.(Linkage)

		if linkage.HashType != hash.TypeInvalid {
			raw.HashType = linkage.HashType
			raw.ApplicationType = linkage.ApplicationType
			raw.ApplicationSubType = linkage.ApplicationSubType
			raw.Offset = dataOffset
			raw.Size = uint32(len(linkage.Data))
		}
	}

	if err := binary.Write(dst, binary.BigEndian, raw); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(raw)), nil
}

func encodeLinkageBody(data any, _ *CodeDirectory, dst io.Writer) (int64, error) {
	if data == nil {
		return 0, nil
	}

	linkage := data.(Linkage)
	if linkage.HashType == hash.TypeInvalid {
		return 0, nil
	}

	n, err := dst.Write(linkage.Data)
	if err != nil {
		return int64(n), err
	}

	return int64(n), nil
}
