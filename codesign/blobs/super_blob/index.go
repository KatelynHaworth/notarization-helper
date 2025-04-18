package super_blob

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs"
)

type (
	// Index defines the representation
	// of a slot within a SuperBlob describing
	// the Slot type and the Blob it contains
	Index struct {
		// Type specifies the Slot type
		// describing the data stored
		// in the Blob at this index.
		Type Slot

		// Blob specifies the decoded Code
		// Signature Blob stored at this
		// index.
		Blob blobs.Blob

		offset uint32
		pos    int
	}

	// rawIndex describes the raw format
	// of an Index when encoding as part
	// of a raw SuperBlob.
	rawIndex struct {
		Type   Slot
		Offset uint32
	}
)

var (
	// rawIndexSize defines the size, in
	// bytes, of a rawIndex when encoded
	// as part of a raw SuperBlob.
	rawIndexSize = uint32(binary.Size(rawIndex{}))
)

// String returns a single line
// representation of this Index.
func (index Index) String() string {
	return fmt.Sprintf("Index{position: %d, type: %s, offset: %d, magic: %s}", index.pos, index.Type, index.offset, index.Blob.Magic())
}

// decodeIndex attempts to read a
// rawIndex from the supplied reader
// and then decode the blobs.Blob stored
// at the offset described in the rawIndex.
func (super *SuperBlob) decodeIndex(src *io.SectionReader) (*Index, error) {
	var raw rawIndex

	if err := binary.Read(src, binary.BigEndian, &raw); err != nil {
		return nil, fmt.Errorf("read raw: %w", err)
	} else if super.hdr.Length < raw.Offset {
		return nil, fmt.Errorf("offset overflows blob length")
	}

	var blobHdr blobs.BlobHeader
	if _, err := blobHdr.ReadFrom(io.NewSectionReader(src, int64(raw.Offset), 8)); err != nil {
		return nil, fmt.Errorf("read blob header at offset %d: %w", raw.Offset, err)
	} else if super.hdr.Length < blobHdr.Length {
		return nil, fmt.Errorf("blob header describes a length that overflows this blob length")
	}

	src = io.NewSectionReader(src, int64(raw.Offset), int64(blobHdr.Length))
	if _, err := src.Seek(int64(blobs.BlobHeaderSize), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek past blob header: %w", err)
	}

	blob, err := blobHdr.Magic.Decode(blobHdr, src)
	if err != nil {
		return nil, fmt.Errorf("decode blob at offset %d: %w", raw.Offset, err)
	}

	return &Index{
		Type: raw.Type,
		Blob: blob,

		offset: raw.Offset,
	}, nil
}

// calculateIndexes calculates both the
// raw size of this SuperBlob but also
// the appropriate offsets for each Index
// based on the raw size of the blobs.Blob
// contained in each Index.
func (super *SuperBlob) calculateIndexes() (uint32, []rawIndex, error) {
	hdr := blobs.BlobHeaderSize + 4 /* indexCount */
	hdr += rawIndexSize * uint32(len(super.blobs))
	bodyOffset := hdr

	rawIndexes := make([]rawIndex, len(super.blobs))
	for i, index := range super.blobs {
		n, err := index.Blob.Length()
		if err != nil {
			return 0, nil, fmt.Errorf("get length for index %d: %w", i, err)
		}

		rawIndexes[i] = rawIndex{
			Type:   index.Type,
			Offset: bodyOffset,
		}

		bodyOffset += n
	}

	return bodyOffset, rawIndexes, nil
}
