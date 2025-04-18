package code_directory

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/hash"
)

var (
	SupportsVersionBase = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x0,
		FullName:             "CODEDIRECTORY_SUPPORTS_BASE",
		ShortName:            "base",
		Decoder:              decodeBase,
		EncodeSizeCalculator: calculateBaseSize,
		HeaderEncoder:        encodeBaseHeader,
		BodyEncoder:          encodeBaseBody,
	})
)

type rawCodeDirectory struct {
	Version        SupportsVersion
	Flags          CodeDirectoryFlag
	HashesOffset   uint32
	IdentityOffset uint32
	SpecialSlots   uint32
	CodeSlots      uint32
	CodeLimit      uint32
	HashSize       uint8
	HashType       hash.Type
	Platform       uint8
	PageSize       uint8
	_              uint32
}

func decodeBase(cd *CodeDirectory, src *io.SectionReader) (any, error) {
	var base rawCodeDirectory
	if err := binary.Read(src, binary.BigEndian, &base); err != nil {
		return nil, fmt.Errorf("read raw: %w", err)
	}

	cd.SupportsData[base.Version] = nil
	cd.Flags = base.Flags
	cd.CodeLimit = base.CodeLimit
	cd.Platform = base.Platform
	cd.PageSize = uint32(1 << base.PageSize)

	if err := base.decodeIdentityTo(cd, src); err != nil {
		return nil, fmt.Errorf("decode identity: %w", err)
	}

	if err := base.decodeHashesTo(cd, src); err != nil {
		return nil, fmt.Errorf("decode hashes: %w", err)
	}

	return nil, nil
}

func (base *rawCodeDirectory) decodeIdentityTo(cd *CodeDirectory, src *io.SectionReader) error {
	if cd.hdr.Length < base.IdentityOffset {
		return fmt.Errorf("identity offset overflows past blob buffer")
	} else if base.IdentityOffset == 0 {
		return nil
	}

	identityBuf := make([]byte, cd.hdr.Length-base.IdentityOffset)
	if _, err := src.ReadAt(identityBuf, int64(base.IdentityOffset)); err != nil {
		return fmt.Errorf("read identity from offset: %w", err)
	}

	if nullIndex := bytes.IndexByte(identityBuf, 0x0); nullIndex < 0 {
		return fmt.Errorf("identity string not found at specified offset %d", base.IdentityOffset)
	} else {
		cd.Identity = string(identityBuf[:nullIndex])
		return nil
	}
}

func (base *rawCodeDirectory) decodeHashesTo(cd *CodeDirectory, src *io.SectionReader) error {
	switch {
	case !base.HashType.Valid():
		return fmt.Errorf("unsupported hash type: %s", base.HashType)

	case base.HashSize != base.HashType.Size():
		return fmt.Errorf("hash size mis-match: slot hash size (%d) != hash type size(%d)", base.HashSize, base.HashType.Size())

	case cd.hdr.Length < base.HashesOffset:
		return fmt.Errorf("hashes offset (%d) outside of blob bounds (%d)", base.HashesOffset, cd.hdr.Length)

	case base.HashesOffset/uint32(base.HashType.Size()) < base.SpecialSlots:
		return fmt.Errorf("special slots overflows into hash offset")

	case (cd.hdr.Length-base.HashesOffset)/uint32(base.HashType.Size()) < base.CodeSlots:
		return fmt.Errorf("code slots overflows past blob buffer")
	}

	cd.SpecialSlots = make([][]byte, base.SpecialSlots)
	for slot := len(cd.SpecialSlots); slot > 0; slot-- {
		cd.SpecialSlots[slot-1] = make([]byte, base.HashType.Size())
		slotOffset := base.HashesOffset - (uint32(base.HashType.Size()) * uint32(slot))

		if _, err := src.ReadAt(cd.SpecialSlots[slot-1], int64(slotOffset)); err != nil {
			return fmt.Errorf("read special slot %d at 0x%x: %w", slot, slotOffset, err)
		}
	}

	cd.CodeSlots = make([][]byte, base.CodeSlots)
	for slot := range cd.CodeSlots {
		cd.CodeSlots[slot] = make([]byte, base.HashType.Size())
		slotOffset := base.HashesOffset + (uint32(base.HashType.Size()) * uint32(slot))

		if _, err := src.ReadAt(cd.CodeSlots[slot], int64(slotOffset)); err != nil {
			return fmt.Errorf("read code slot %d at 0x%x: %w", slot, slotOffset, err)
		}
	}

	cd.HashType = base.HashType
	return nil
}

func calculateBaseSize(_ any, cd *CodeDirectory) (hdr, data uint32, _ error) {
	hdr = uint32(binary.Size(rawCodeDirectory{}))

	if identitySize := len(cd.Identity); identitySize > 0 {
		data += uint32(identitySize) + 1 /* null terminated */
	}

	hashSize := cd.HashType.Size()

	for slot := len(cd.SpecialSlots); slot > 0; slot-- {
		if len(cd.SpecialSlots[slot-1]) < int(hashSize) {
			return 0, 0, fmt.Errorf("special slot %d contains a hash that is too small for the specified hash type", slot)
		}

		data += uint32(hashSize)
	}

	for slot := range cd.CodeSlots {
		if len(cd.CodeSlots[slot]) < int(hashSize) {
			return 0, 0, fmt.Errorf("code slot %d contains a hash that is too small for the specified hash type", slot)
		}

		data += uint32(hashSize)
	}

	return
}

func encodeBaseHeader(_ any, cd *CodeDirectory, dst io.Writer, dataOffset, _ uint32) (int64, error) {
	var base rawCodeDirectory

	pageSizeLog := math.Log2(float64(cd.PageSize))
	if pageSizeLog > math.MaxUint8 {
		return 0, fmt.Errorf("log2 of page size (%d) overflows an unsigned 8-bit integer", pageSizeLog)
	}

	base.Version = cd.Version()
	base.Flags = cd.Flags
	base.CodeLimit = cd.CodeLimit
	base.Platform = cd.Platform
	base.PageSize = uint8(pageSizeLog)

	if identityLen := uint32(len(cd.Identity)); identityLen > 0 {
		base.IdentityOffset = dataOffset
		dataOffset += identityLen + 1 /* null terminated */
	}

	if teamIdData, exists := cd.SupportsData[SupportsVersionTeamID].(string); exists {
		// Apple is really annoying and decides to shove
		// the _*optional*_ team ID in between the identity
		// and hashes rather than placing data in consecutive
		// order based on versions 🙄
		//
		// As such we need to account for the length of the team
		// ID in the offset before calculating the hashes offset.
		dataOffset += uint32(len(teamIdData) + 1 /* null terminated */)
	}

	// Special Slot hashes are encoded after
	// the identifier but before the start
	// of Code Slot hashes of which HashesOffset
	// points to so set the HashesOffset at
	// the correct offset to allow for the
	// number of Special Shot hashes defined
	base.HashesOffset = dataOffset + uint32(len(cd.SpecialSlots)*int(cd.HashType.Size()))
	base.HashType = cd.HashType
	base.HashSize = cd.HashType.Size()
	base.SpecialSlots = uint32(len(cd.SpecialSlots))
	base.CodeSlots = uint32(len(cd.CodeSlots))

	if err := binary.Write(dst, binary.BigEndian, base); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(base)), nil
}

func encodeBaseBody(_ any, cd *CodeDirectory, dst io.Writer) (writeCount int64, _ error) {
	if len(cd.Identity) > 0 {
		n, err := dst.Write(append([]byte(cd.Identity), 0x0))
		writeCount += int64(n)

		if err != nil {
			return writeCount, fmt.Errorf("encode identity: %w", err)
		}
	}

	if teamIdData, exists := cd.SupportsData[SupportsVersionTeamID].(string); exists {
		// As noted in encodeBaseHeader the team ID is
		// encoded after the identifier even though it
		// is from an optional version.
		//
		// As such, to ensure encoding produces a byte
		// layout that mimics a code directory made by
		// `codesign` we have to write the team ID here
		// rather than in the team ID encoder func.

		n, err := dst.Write(append([]byte(teamIdData), 0x0))
		writeCount += int64(n)

		if err != nil {
			return writeCount, fmt.Errorf("encode team ID: %w", err)
		}
	}

	n, err := encodeBaseBodyHashes(cd, dst)
	writeCount += n

	if err != nil {
		return writeCount, fmt.Errorf("encode hashes: %w", err)
	}

	return
}

func encodeBaseBodyHashes(cd *CodeDirectory, dst io.Writer) (writeCount int64, _ error) {
	hashSize := cd.HashType.Size()

	for slot := len(cd.SpecialSlots); slot > 0; slot-- {
		n, err := dst.Write(cd.SpecialSlots[slot-1][:hashSize])
		writeCount += int64(n)

		if err != nil {
			return writeCount, fmt.Errorf("special slot %d: %w", slot, err)
		}
	}

	for slot := range cd.CodeSlots {
		n, err := dst.Write(cd.CodeSlots[slot][:hashSize])
		writeCount += int64(n)

		if err != nil {
			return writeCount, fmt.Errorf("code slot %d: %w", slot, err)
		}
	}

	return
}
