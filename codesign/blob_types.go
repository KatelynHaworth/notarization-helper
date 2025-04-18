package codesign

import (
	"fmt"
	"io"
	"os"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/code_directory"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/super_blob"
)

var (
	MagicCodeDirectory     = blobs.RegisterBlobType(code_directory.Metadata)
	MagicEmbeddedSignature = blobs.RegisterBlobType(super_blob.Metadata)
)

// Generic blob types that don't have
// a defined data structure to decode
// into
var (
	MagicRequirement              = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade0c00, Name: "CSMAGIC_REQUIREMENT"})
	MagicRequirements             = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade0c01, Name: "CSMAGIC_REQUIREMENTS"})
	MagicEmbeddedSignatureOld     = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade0b02, Name: "CSMAGIC_EMBEDDED_SIGNATURE_OLD"})
	MagicEmbeddedEntitlements     = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade7171, Name: "CSMAGIC_EMBEDDED_ENTITLEMENTS"})
	MagicEmbeddedDEREntitlements  = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade7172, Name: "CSMAGIC_EMBEDDED_DER_ENTITLEMENTS"})
	MagicDetachedSignature        = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade0cc1, Name: "CSMAGIC_DETACHED_SIGNATURE"})
	MagicBlobWrapper              = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade0b01, Name: "CSMAGIC_BLOBWRAPPER"})
	MagicEmbeddedLaunchConstraint = blobs.RegisterBlobType(blobs.BlobMetadata{MagicValue: 0xfade8181, Name: "CSMAGIC_EMBEDDED_LAUNCH_CONSTRAINT"})
)

type Blob = blobs.Blob

func ReadFrom[T Blob](src *io.SectionReader) (T, error) {
	var (
		blobHdr blobs.BlobHeader
		zero    T
	)

	if _, err := blobHdr.ReadFrom(src); err != nil {
		return zero, fmt.Errorf("read blob header: %w", err)
	}

	src = io.NewSectionReader(src, 0, int64(blobHdr.Length))
	if _, err := src.Seek(int64(blobs.BlobHeaderSize), io.SeekStart); err != nil {
		return zero, fmt.Errorf("seek past blob header: %w", err)
	}

	blob, err := blobHdr.Magic.Decode(blobHdr, src)
	if err != nil {
		return zero, fmt.Errorf("decode blob: %w", err)
	}

	typedBlob, ok := blob.(T)
	if !ok {
		return zero, fmt.Errorf("decoded blob does not match expected type (%T): %T", zero, blob)
	}

	return typedBlob, nil
}

func WriteTo(blob Blob, dst io.Writer) (int64, error) {
	return blob.Magic().Encode(blob, dst)
}

func WriteToTemp(blob Blob) (*os.File, int64, error) {
	blobTemp, err := os.CreateTemp("", "codesign-blob-*")
	if err != nil {
		return nil, -1, fmt.Errorf("create temp file for codesign blob: %w", err)
	}

	size, err := WriteTo(blob, blobTemp)
	if err != nil {
		_ = blobTemp.Close()
		return nil, -1, fmt.Errorf("write codesign blob to temp file: %w", err)
	}

	if _, err = blobTemp.Seek(0, io.SeekStart); err != nil {
		_ = blobTemp.Close()
		return nil, -1, fmt.Errorf("seek back to start of raw codesign blob file: %w", err)
	}

	return blobTemp, size, nil
}
