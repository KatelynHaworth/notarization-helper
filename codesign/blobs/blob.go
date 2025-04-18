package blobs

import (
	"encoding/binary"
	"fmt"
	"io"
)

type (
	// Magic represents a 32-bit unsigned integer
	// that is included as the first 4 bytes of
	// a Code Signature blob to declare the type
	// of that blob.
	Magic uint32

	// BlobHeader defines the generic header
	// data structure that is included at the
	// start of all Code Signature blobs.
	BlobHeader struct {
		// Magic specifies the unique identifier
		// of the Code Signature blob following
		// this header.
		Magic Magic

		// Length specifies, in bytes, the size
		// of the Code Signature blob following
		// this header.
		Length uint32
	}

	// Blob defines a generic set of functions
	// a structure must implement to be able
	// to represent a Code Signature blob
	Blob interface {
		// Length returns, in bytes, the size
		// of the Code Signature blob when it
		// is encoded to its raw format.
		Length() (uint32, error)

		// Magic returns the unique identifier
		// of the Code Signature blob.
		Magic() Magic
	}
)

var (
	// BlobHeaderSize defines the raw size, in bytes,
	// a BlobHeader takes when it is encoded to its
	// raw format.
	BlobHeaderSize = uint32(binary.Size(BlobHeader{}))
)

// ReadFrom attempts to read and decode
// this BlobHeader from the supplied reader.
func (hdr *BlobHeader) ReadFrom(r io.Reader) (int64, error) {
	if err := binary.Read(r, binary.BigEndian, hdr); err != nil {
		return -1, err
	}

	if hdr.Length < BlobHeaderSize {
		return -1, fmt.Errorf("header contains invalid length: %d", hdr.Length)
	}

	return int64(BlobHeaderSize), nil
}

// WriteTo attempts to encode and write
// this BlobHeader to the supplied writer.
func (hdr *BlobHeader) WriteTo(w io.Writer) (int64, error) {
	if hdr.Length < BlobHeaderSize {
		return -1, fmt.Errorf("header contains invalid length: %d", hdr.Length)
	}

	if err := binary.Write(w, binary.BigEndian, hdr); err != nil {
		return -1, err
	}

	return int64(BlobHeaderSize), nil
}
