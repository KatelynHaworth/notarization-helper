package dmg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var (
	UDIFSignature = [4]byte{'k', 'o', 'l', 'y'}

	ErrMagicMismatch = errors.New("magic doesn't match expected for a UDIF file")

	UDIFResourceFileSize = binary.Size(UDIFResourceFile{})
)

type UDIFResourceFile struct {
	Signature             [4]byte // magic 'koly'
	Version               uint32  // 4 (as of 2013)
	HeaderSize            uint32  // sizeof(this) =  512 (as of 2013)
	Flags                 uint32
	RunningDataForkOffset uint64
	DataForkOffset        uint64 // usually 0, beginning of file
	DataForkLength        uint64
	RsrcForkOffset        uint64 // resource fork offset and length
	RsrcForkLength        uint64
	SegmentNumber         uint32 // Usually 1, can be 0
	SegmentCount          uint32 // Usually 1, can be 0
	SegmentID             [16]byte
	DataChecksumType      uint32 // Data fork checksum
	DataChecksumSize      uint32
	DataChecksum          [32]uint32
	XMLOffset             uint64 // Position of XML property list in file
	XMLLength             uint64
	Reserved1             [68]byte
	CodeSignOffset        uint32
	Reserved2             [4]byte
	CodeSignLength        uint32
	Reserved3             [40]byte
	ChecksumType          uint32 // Master checksum
	ChecksumSize          uint32
	Checksum              [32]uint32
	ImageVariant          uint32 // Unknown, commonly 1
	SectorCount           uint64
	Reserved4             [12]byte
}

type readAtSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

func ReadUDIF(src readAtSeeker) (*UDIFResourceFile, int64, error) {
	trailerOffset, err := src.Seek(int64(UDIFResourceFileSize)*-1, io.SeekEnd)
	if err != nil {
		return nil, -1, fmt.Errorf("seek to potential UDIF trailer")
	}

	var magic [4]byte
	if _, err = src.ReadAt(magic[:], trailerOffset); err != nil {
		return nil, -1, fmt.Errorf("read magic from potential UDIF trailer location")
	} else if !bytes.Equal(magic[:], UDIFSignature[:]) {
		return nil, -1, fmt.Errorf("0x%x != 0x%x: %w", magic, UDIFSignature, ErrMagicMismatch)
	}

	var trailer UDIFResourceFile
	if err = binary.Read(src, binary.BigEndian, &trailer); err != nil {
		return nil, -1, fmt.Errorf("read trailer: %w", err)
	}

	return &trailer, trailerOffset, nil
}

func WriteUDIF(trailer *UDIFResourceFile, dst io.Writer) error {
	if !bytes.Equal(trailer.Signature[:], UDIFSignature[:]) {
		trailer.Signature = UDIFSignature
	}

	if err := binary.Write(dst, binary.BigEndian, trailer); err != nil {
		return fmt.Errorf("write trailer: %w", err)
	}

	return nil
}
