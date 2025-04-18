package code_directory

import (
	"encoding/hex"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/hash"
)

var (
	magicValue = uint32(0xfade0c02)
	Metadata   = blobs.BlobMetadata{
		Name:       "CSMAGIC_CODEDIRECTORY",
		MagicValue: magicValue,
		Decoder:    Decoder,
		Encoder:    Encoder,
	}
)

type CodeDirectory struct {
	hdr blobs.BlobHeader

	Flags        CodeDirectoryFlag
	Identity     string
	HashType     hash.Type
	CodeSlots    [][]byte
	SpecialSlots [][]byte
	CodeLimit    uint32
	Platform     uint8
	PageSize     uint32

	SupportsData map[SupportsVersion]any
}

func Decoder(hdr blobs.BlobHeader, src *io.SectionReader) (blobs.Blob, error) {
	if magic := uint32(hdr.Magic); magic != magicValue {
		return nil, fmt.Errorf("magic in blob header (0x%d) doesn't match the expected value (0x%x)", magic, magicValue)
	}

	cd := &CodeDirectory{
		hdr: hdr,

		SupportsData: make(map[SupportsVersion]any),
	}

	version := cd.Version()
	for _, verMeta := range supportsRegistry {
		if version < verMeta.ver {
			break
		}

		data, err := verMeta.Decoder(cd, src)
		if err != nil {
			return nil, fmt.Errorf("decode version '%s' (0x%x): %w", verMeta.ShortName, verMeta.Version, err)
		} else if data == nil {
			continue
		}

		cd.SupportsData[verMeta.ver] = data
	}

	return cd, nil
}

func Encoder(blob blobs.Blob, dst io.Writer) (int64, error) {
	cd, ok := blob.(*CodeDirectory)
	if !ok {
		return -1, fmt.Errorf("code directory encoder invoked for blob of a different type: %T", blob)
	}

	sizes, hdrSize, bodySize, err := cd.calculateSizes()
	if err != nil {
		return -1, fmt.Errorf("calculate sizes: %w", err)
	}

	hdrSize += blobs.BlobHeaderSize
	blobHdr := blobs.BlobHeader{Magic: blobs.Magic(magicValue), Length: hdrSize + bodySize}

	writeCount, err := blobHdr.WriteTo(dst)
	if err != nil {
		return writeCount, fmt.Errorf("write blob header: %w", err)
	}

	for _, ver := range sizes {
		n, err := ver.HeaderEncoder(cd.SupportsData[ver.ver], cd, dst, hdrSize, hdrSize+ver.bodyOffset)
		writeCount += n

		if err != nil {
			return writeCount, fmt.Errorf("write '%s' header: %w", ver.ShortName, err)
		}
	}

	for _, ver := range sizes {
		if ver.BodyEncoder == nil {
			continue
		}

		n, err := ver.BodyEncoder(cd.SupportsData[ver.ver], cd, dst)
		writeCount += n

		if err != nil {
			return writeCount, fmt.Errorf("write '%s' body: %w", ver.ShortName, err)
		}
	}

	return writeCount, nil
}

func (cd *CodeDirectory) Version() SupportsVersion {
	var vers []SupportsVersion

	for ver := range cd.SupportsData {
		vers = append(vers, ver)
	}

	if len(vers) == 0 {
		return SupportsVersion(0x0)
	}

	slices.Sort(vers)
	return vers[len(vers)-1]
}

func (cd *CodeDirectory) Hash() ([]byte, error) {
	h := cd.HashType.New()

	if _, err := Encoder(cd, h); err != nil {
		return nil, fmt.Errorf("encode directory to generate hash: %w", err)
	}

	raw := h.Sum(nil)
	return raw[:cd.HashType.Size()], nil
}

func (cd *CodeDirectory) calculateSizes() ([]supportsEncodeSize, uint32, uint32, error) {
	var (
		encodeSizes []supportsEncodeSize
		hdr, body   uint32
	)

	version := cd.Version()
	for _, meta := range supportsRegistry {
		if version < meta.ver {
			break
		}

		verHdr, verBody, err := meta.EncodeSizeCalculator(cd.SupportsData[meta.ver], cd)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("calculate sizes for '%s': %w", meta.ShortName, err)
		}

		encodeSizes = append(encodeSizes, supportsEncodeSize{
			SupportsMetadata: meta,
			bodyOffset:       body,
		})

		hdr += verHdr
		body += verBody
	}

	return encodeSizes, hdr, body, nil
}

func (cd *CodeDirectory) Magic() blobs.Magic {
	return cd.hdr.Magic
}

func (cd *CodeDirectory) Length() (uint32, error) {
	_, hdrs, bodies, err := cd.calculateSizes()
	if err != nil {
		return 0, fmt.Errorf("calculate sizes: %w", err)
	}

	return blobs.BlobHeaderSize + hdrs + bodies, nil
}

func (cd *CodeDirectory) String() string {
	var builder strings.Builder

	builder.WriteString("CodeDirectory{")
	_, _ = fmt.Fprintf(&builder, "length: %d, ", cd.hdr.Length)
	_, _ = fmt.Fprintf(&builder, "version: %s, ", cd.Version)
	_, _ = fmt.Fprintf(&builder, "identity: %s, ", cd.Identity)
	_, _ = fmt.Fprintf(&builder, "flags: %s, ", cd.Flags)
	_, _ = fmt.Fprintf(&builder, "hash_type: %s, ", cd.HashType)
	_, _ = fmt.Fprintf(&builder, "hashes: %d+%d, ", len(cd.CodeSlots), len(cd.SpecialSlots))

	if hash, err := cd.Hash(); err == nil {
		_, _ = fmt.Fprintf(&builder, "cd_hash: %s", hex.EncodeToString(hash))
	} else {
		_, _ = fmt.Fprintf(&builder, "cd_hash: ERR(%s), ", err)
	}

	builder.WriteString("}")

	return builder.String()
}
