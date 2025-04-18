package blobs

import "io"

type (
	// BlobDecoder defines the function signature
	// for a function that can decode a specific
	// Code Signature blob from its raw format
	// into the appropriate data structure that
	// represents it.
	BlobDecoder func(hdr BlobHeader, src *io.SectionReader) (Blob, error)

	// BlobEncoder defines the function signature
	// for a function that can encode a specific
	// Code Signature blob from its data structure
	// into its raw format.
	BlobEncoder func(blob Blob, dst io.Writer) (int64, error)

	// BlobMetadata defines a data structure
	// used to store information about a specific
	// Code Signature blob type so that this library
	// to appropriate encode and decode the blob.
	BlobMetadata struct {
		// MagicValue specifies the 32-bit unsigned
		// integer used to represent the specific
		// Code Signature blob.
		MagicValue uint32

		// Name specifies a unique name for the
		// Code Signature blob can be used when
		// producing error messages for the
		// specific Code Signature blob.
		Name string

		// Decoder specifies the BlobDecoder
		// function to use when decoding the
		// specific Code Signature blob from
		// its raw format.
		Decoder BlobDecoder

		// Encoder specifies the BlobEncoder
		// function to use when encoding the
		// specific Code Signature blob to its
		// raw format.
		Encoder BlobEncoder
	}
)
