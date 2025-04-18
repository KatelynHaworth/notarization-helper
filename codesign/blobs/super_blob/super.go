package super_blob

import (
	"encoding/binary"
	"fmt"
	"io"
	"slices"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/code_directory"
)

var (
	// magicValue defines the 32-bit unsigned
	// integer used to represent a SuperBlob
	// when encoded.
	magicValue = uint32(0xfade0cc0)

	// Metadata defines information about
	// the SuperBlob Code Signature blob
	// type.
	Metadata = blobs.BlobMetadata{
		MagicValue: magicValue,
		Name:       "CSMAGIC_EMBEDDED_SIGNATURE",
		Decoder:    Decoder,
		Encoder:    Encoder,
	}
)

// SuperBlob defines a Code Signature
// Blob that can store within itself
// multiple Code Signature blobs of
// different types.
type SuperBlob struct {
	hdr    blobs.BlobHeader
	bestCd *code_directory.CodeDirectory

	blobs []*Index
}

// Decoder defines a blobs.BlobDecoder that
// can decode a SuperBlob from the supplied
// reader.
//
// Decoders for all Code Signature blobs contained
// within the SuperBlob will be invoked as
// part of decoding the SuperBlob.
//
// The decoder will also conduct validation to
// ensure there are no duplicate slots or code
// directories within the SuperBlob.
func Decoder(hdr blobs.BlobHeader, src *io.SectionReader) (blobs.Blob, error) {
	if magic := uint32(hdr.Magic); magic != magicValue {
		return nil, fmt.Errorf("magic in blob header (0x%d) doesn't match the expected value (0x%x)", magic, magicValue)
	}

	var indexCount uint32
	if err := binary.Read(src, binary.BigEndian, &indexCount); err != nil {
		return nil, fmt.Errorf("read super blob index count: %w", err)
	}

	if hdr.Length/rawIndexSize < indexCount {
		return nil, fmt.Errorf("super blob is too small fit specified (%d) number of indexes", indexCount)
	}

	super := &SuperBlob{
		hdr: hdr,

		blobs: make([]*Index, indexCount),
	}

	for pos := range super.blobs {
		index, err := super.decodeIndex(src)
		switch {
		case err != nil:
			return nil, fmt.Errorf("decode index %d: %w", pos, err)

		case super.GetSlot(index.Type) != nil:
			return nil, fmt.Errorf("duplicate indexes of slot type '%s' detected", index.Type)

		case index.Type == SlotCodeDirectory || (SlotAlternativeCodeDirectories <= index.Type && index.Type < SlotAlternativeCodeDirectoryLimit):
			cd, ok := index.Blob.(*code_directory.CodeDirectory)
			switch {
			case !ok:
				return nil, fmt.Errorf("index slot type '%s' doesn't contain a code directory: found %s", index.Type, index.Blob.Magic())

			case super.bestCd == nil || super.bestCd.HashType.Priority() < cd.HashType.Priority():
				super.bestCd = cd

			case super.bestCd != nil || super.bestCd.HashType.Priority() == cd.HashType.Priority():
				return nil, fmt.Errorf("multiple %d code directories found, rejecting", cd.HashType)
			}
		}

		index.pos = pos
		super.blobs[pos] = index
	}

	return super, nil
}

// Encoder defines a blobs.BlobEncoder that
// can encode a SuperBlob into its raw format
// and write that output to the supplied writer.
//
// The encoder for each Code Signature blob contained
// within the SuperBlob will also be invoked as part
// of encoding the SuperBlob.
func Encoder(blob blobs.Blob, dst io.Writer) (int64, error) {
	super, ok := blob.(*SuperBlob)
	if !ok {
		return -1, fmt.Errorf("super blob encoder invoked for blob of a different type: %T", blob)
	}

	length, indexes, err := super.calculateIndexes()
	if err != nil {
		return -1, fmt.Errorf("calculate index sizes: %w", err)
	}

	blobHdr := blobs.BlobHeader{Magic: blobs.Magic(magicValue), Length: length}
	writeCount, err := blobHdr.WriteTo(dst)
	if err != nil {
		return writeCount, fmt.Errorf("write blob header: %w", err)
	}

	if err = binary.Write(dst, binary.BigEndian, uint32(len(indexes))); err != nil {
		return writeCount, fmt.Errorf("write index count: %w", err)
	}
	writeCount += 4

	for i, index := range indexes {
		if err = binary.Write(dst, binary.BigEndian, index); err != nil {
			return writeCount, fmt.Errorf("write index %d header: %w", i, err)
		}

		writeCount += int64(rawIndexSize)
	}

	for i, index := range super.blobs {
		n, err := index.Blob.Magic().Encode(index.Blob, dst)
		writeCount += n

		if err != nil {
			return writeCount, fmt.Errorf("write index %d body: %w", i, err)
		}
	}

	return writeCount, nil
}

// Length returns the size of the SuperBlob
// in bytes by calculating the number of indexes
// and the size of the Code Signature blobs contained
// within each index.
func (super *SuperBlob) Length() (uint32, error) {
	length, _, err := super.calculateIndexes()
	return length, err
}

// Magic returns the blobs.Magic from the
// SuperBlob's header.
func (super *SuperBlob) Magic() blobs.Magic {
	return super.hdr.Magic
}

// String returns a single line representation
// of the SuperBlob.
func (super *SuperBlob) String() string {
	return fmt.Sprintf("SuperBlob{magic: %s, length: %d, entries: %d}", super.Magic(), super.hdr.Length, len(super.blobs))
}

// BestCodeDirectory will return a *code_directory.CodeDirectory
// from within the SuperBlob that has been deemed
// to be the "best" candidate based on its hash.Type
// priority.
func (super *SuperBlob) BestCodeDirectory() *code_directory.CodeDirectory {
	return super.bestCd
}

// Count returns the number of Code Signature
// blobs.Blob stored within the SuperBlob.
func (super *SuperBlob) Count() int {
	return len(super.blobs)
}

// GetIndex will return *Index stored at
// the supplied position.
//
// If the position exceeds the size of
// the index array a nil *Index will be
// returned.
func (super *SuperBlob) GetIndex(i int) *Index {
	if i > len(super.blobs) {
		return nil
	}

	return super.blobs[i]
}

// GetSlot will return the *Index stored
// within the SuperBlob that has a matching
// Slot.
//
// If no matching Slot could be found a nil
// *Index will be returned.
func (super *SuperBlob) GetSlot(slot Slot) *Index {
	exists, i := super.slotExists(slot)
	if !exists {
		return nil
	}

	return super.blobs[i]
}

// AddBlob inserts a Code Signature blobs.Blob
// into the SuperBlob creating a new *Index
// of the Slot type supplied.
//
// If the SuperBlob already contains an *Index
// with a matching Slot type then an error will
// be returned.
//
// Additionally, if the Slot type is for a Code
// Directory, validation will be completed to ensure
// the blobs.Blob is a *code_directory.CodeDirectory
// and the Code Directory's hash.Type is unique.
func (super *SuperBlob) AddBlob(slot Slot, blob blobs.Blob) error {
	if exists, _ := super.slotExists(slot); exists {
		return fmt.Errorf("slot '%s' already has a blob assigned", slot)
	}

	if slot == SlotCodeDirectory || (SlotAlternativeCodeDirectories <= slot && slot < SlotAlternativeCodeDirectoryLimit) {
		cd, ok := blob.(*code_directory.CodeDirectory)
		switch {
		case !ok:
			return fmt.Errorf("index slot type '%s' may only store a *code_directory.CodeDirectory, received: %T", slot, blob)

		case super.bestCd == nil || super.bestCd.HashType.Priority() < cd.HashType.Priority():
			super.bestCd = cd

		case super.bestCd != nil || super.bestCd.HashType.Priority() == cd.HashType.Priority():
			return fmt.Errorf("code directory has the same hash type (%d) as the current best code directory", cd.HashType)
		}
	}

	super.blobs = append(super.blobs, &Index{
		Type: slot,
		Blob: blob,

		pos: len(super.blobs),
	})

	return nil
}

// slotExists checks if the SuperBlob
// contains an *Index with the same
// Slot type as the type supplied.
//
// If a matching slot is found, the
// position of that *Index within the
// SuperBlob is returned.
func (super *SuperBlob) slotExists(slot Slot) (bool, int) {
	i := slices.IndexFunc(super.blobs, func(index *Index) bool {
		if index != nil {
			// An index in the slice will only ever
			// be nil when first decoding and validating
			// the super blob

			return index.Type == slot
		}

		return false
	})

	return i > -1, i
}
