package blobs

import (
	"fmt"
	"io"
)

var (
	// blobRegistry defines the lookup
	// table that links a registered
	// Magic to the appropriate BlobMetadata
	// describing it.
	blobRegistry = map[Magic]*BlobMetadata{}
)

// RegisterBlobType will register a new
// Code Signature Blob type with the library
// allowing it to decode and encode the Blob
// to and from its raw format.
//
// If the Magic value specified in the BlobMetadata
// has already been registered this function will
// panic.
func RegisterBlobType(metadata BlobMetadata) Magic {
	if meta := blobRegistry[Magic(metadata.MagicValue)]; meta != nil {
		panic(fmt.Sprintf("magic 0x%x already registered to blob type '%s'", meta.MagicValue, meta.Name))
	}

	blobRegistry[Magic(metadata.MagicValue)] = &metadata
	return Magic(metadata.MagicValue)
}

// Decode looks up the BlobMetadata
// registered to this Magic and, if
// found, uses its BlobDecoder to
// decode the appropriate Blob from
// the supplied reader.
//
// If the Magic doesn't have a BlobMetadata
// registered then the BlobDecoder
// for a Generic blob will be used.
func (magic Magic) Decode(hdr BlobHeader, src *io.SectionReader) (Blob, error) {
	if meta := blobRegistry[magic]; meta != nil && meta.Decoder != nil {
		return meta.Decoder(hdr, src)
	}

	return GenericDecoder(hdr, src)
}

// Encode looks up the BlobMetadata
// registered to this Magic and, if
// found, uses its BlobEncoder to
// encode the Blob in its raw format
// to the supplied writer.
//
// If the Magic doesn't have a BlobMetadata
// registered but the Blob's type
// is *Generic then the BlobEncoder
// for a Generic blob will be used.
func (magic Magic) Encode(blob Blob, dst io.Writer) (int64, error) {
	if meta := blobRegistry[magic]; meta != nil && meta.Encoder != nil {
		return meta.Encoder(blob, dst)
	}

	if generic, ok := blob.(*Generic); ok {
		return GenericEncoder(generic, dst)
	}

	return -1, fmt.Errorf("no encoder has been registered for this custom blob type: %T", blob)
}

// String returns the name of this Magic
// as described in its registered BlobMetadata.
//
// If no BlobMetadata has been registered to
// this Magic then the hex encoding will be
// returned instead.
func (magic Magic) String() string {
	if meta := blobRegistry[magic]; meta != nil {
		return meta.Name
	}

	return fmt.Sprintf("0x%x", uint32(magic))
}
