package blobs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// Generic defines a Code Signature Blob
// type that can contain a raw Blob when
// no BlobMetadata was found that matched
// the Magic in the BlobHeader.
//
// The Generic blob is also useful in cases
// when the blob doesn't have a data structure
// to represent it as the body is opaque data,
// for example a notarization ticket.
type Generic struct {
	hdr BlobHeader
	raw []byte
}

// NewGeneric constructs a new Generic
// Code Signature Blob that will store
// the supplied raw data and represent
// it using the supplied Magic.
func NewGeneric(magic Magic, raw []byte) (*Generic, error) {
	generic := &Generic{
		hdr: BlobHeader{
			Magic:  magic,
			Length: BlobHeaderSize + uint32(len(raw)),
		},
	}

	var buf bytes.Buffer
	if _, err := generic.hdr.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("write blob header into raw buffer: %w", err)
	}

	buf.Write(raw)
	generic.raw = buf.Bytes()
	return generic, nil
}

// GenericDecoder defines a BlobDecoder
// function that can decode a raw Code
// Signature blob of any type into a
// Generic Blob.
func GenericDecoder(hdr BlobHeader, src *io.SectionReader) (Blob, error) {
	generic := &Generic{
		hdr: hdr,
		raw: make([]byte, hdr.Length),
	}

	if _, err := src.ReadAt(generic.raw, 0); err != nil {
		return nil, fmt.Errorf("read blob data: %w", err)
	}

	return generic, nil
}

// GenericEncoder defines a BlobEncoder
// function that can encode the supplied
// Generic into a raw Code Signature Blob.
func GenericEncoder(generic *Generic, dst io.Writer) (int64, error) {
	n, err := dst.Write(generic.raw)
	if err != nil {
		return int64(n), fmt.Errorf("write blob raw: %w", err)
	}

	return int64(n), nil
}

// Length returns the raw size of this
// Blob when encoded in its raw format.
//
// This function returns no error as the
// length is known when the Generic is
// constructed.
func (generic *Generic) Length() (uint32, error) {
	return generic.hdr.Length, nil
}

// Magic returns the Magic defined
// in the BlobHeader of this Generic.
func (generic *Generic) Magic() Magic {
	return generic.hdr.Magic
}

// String returns a single line representation
// of this Generic Blob.
func (generic *Generic) String() string {
	hash := sha256.New()
	hash.Write(generic.raw)

	return fmt.Sprintf("Generic{length: %d, hash: %s}", generic.hdr.Length, hex.EncodeToString(hash.Sum(nil)))
}
