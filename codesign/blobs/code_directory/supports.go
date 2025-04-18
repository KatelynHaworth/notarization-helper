package code_directory

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

type (
	SupportsVersion uint32

	SupportsDecoder func(cd *CodeDirectory, src *io.SectionReader) (any, error)

	SupportsSizeCalculator func(data any, cd *CodeDirectory) (hdr, body uint32, err error)

	SupportsHeaderEncoder func(data any, cd *CodeDirectory, dst io.Writer, dataOffsetStart, dataOffsetCurrent uint32) (int64, error)

	SupportsBodyEncoder func(data any, cd *CodeDirectory, dst io.Writer) (int64, error)

	SupportsMetadata struct {
		// Version specifies the support version
		// associated with this data structure
		Version uint32

		// FullName specifies the name of the
		// support version as defined in the
		// darwin kernel
		FullName string

		// ShortName specifies a smaller name
		// for the support version that can be
		// used in error logs
		ShortName string

		// RawHeaderSize specifies the size
		// of the raw header structure, as it
		// is represented in binary form, when
		// it is included in a raw CodeDirectory
		// header.
		//
		// This value is used to compute the base
		// offset for data locations when encoding
		// a CodeDirectory to its raw value.
		//
		// NOTE: Use binary.Size to obtain this size
		// rather than unsafe.Sizeof as the latter will
		// include struct alignments that don't actually
		// impact the encoded size
		//RawHeaderSize uint32

		// Decoder specifies a utility function
		// to decode the version specific data
		// for this support version into the final
		// decoded CodeDirectory
		Decoder SupportsDecoder

		EncodeSizeCalculator SupportsSizeCalculator

		HeaderEncoder SupportsHeaderEncoder

		BodyEncoder SupportsBodyEncoder

		// ver defines a type cast of Version
		// to a SupportsVersion used for internally
		// when decoding and encoding a CodeDirectory
		ver SupportsVersion
	}

	supportsEncodeSize struct {
		SupportsMetadata

		bodyOffset uint32
	}
)

var (
	supportsRegistry []SupportsMetadata
)

func RegisterSupportsVersion(meta SupportsMetadata) SupportsVersion {
	alreadyExists := slices.ContainsFunc(supportsRegistry, func(existing SupportsMetadata) bool {
		return existing.Version == meta.Version
	})

	if alreadyExists {
		panic(fmt.Sprintf("Code Directory supports version 0x%d is already registered to a metadata"))
	}

	meta.ver = SupportsVersion(meta.Version)
	supportsRegistry = append(supportsRegistry, meta)

	slices.SortFunc(supportsRegistry, func(a, b SupportsMetadata) int {
		return int(a.Version) - int(b.Version)
	})

	return meta.ver
}

func (version SupportsVersion) String() string {
	var verNames []string

	for _, meta := range supportsRegistry {
		if version >= meta.ver {
			verNames = append(verNames, meta.FullName)
		}
	}

	return fmt.Sprintf("0x%x (%s)", uint32(version), strings.Join(verNames, ","))
}
