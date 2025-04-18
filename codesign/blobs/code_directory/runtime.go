package code_directory

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	SupportsVersionRuntime = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020500,
		FullName:             "CODEDIRECTORY_SUPPORTS_RUNTIME",
		ShortName:            "runtime",
		Decoder:              decodeRuntime,
		EncodeSizeCalculator: calculateRuntimeSize,
		HeaderEncoder:        encodeRuntimeHeader,
		BodyEncoder:          encodeRuntimeBody,
	})
)

type (
	Runtime struct {
		Version         RuntimeVersion
		PreEncryptSlots [][]byte
	}

	rawRuntime struct {
		Version RuntimeVersion
		Offset  uint32
	}
)

func decodeRuntime(cd *CodeDirectory, src *io.SectionReader) (any, error) {
	var raw rawRuntime
	if err := binary.Read(src, binary.BigEndian, &raw); err != nil {
		return nil, fmt.Errorf("read raw: %w", err)
	}

	runtime := Runtime{Version: raw.Version}
	if raw.Offset == 0 {
		return runtime, nil
	}

	hashSize := uint32(cd.HashType.Size())
	if cd.hdr.Length < raw.Offset+(hashSize*uint32(len(cd.CodeSlots))) {
		return nil, fmt.Errorf("pre-encrypt slots overflow blob length")
	}

	slots := make([][]byte, len(cd.CodeSlots))
	for slot := range slots {
		slots[slot] = make([]byte, hashSize)
		slotOffset := raw.Offset + (hashSize * uint32(slot))

		if _, err := src.ReadAt(slots[slot], int64(slotOffset)); err != nil {
			return nil, fmt.Errorf("read pre-encrypt slot %d at 0x%x: %w", slot, slotOffset, err)
		}
	}

	runtime.PreEncryptSlots = slots
	return runtime, nil
}

func calculateRuntimeSize(data any, cd *CodeDirectory) (hdr, body uint32, _ error) {
	runtime, ok := data.(Runtime)
	if !ok {
		return 0, 0, fmt.Errorf("unexpected data type %T", data)
	}

	hdr = uint32(binary.Size(rawRuntime{}))
	if len(runtime.PreEncryptSlots) > 0 {
		if len(runtime.PreEncryptSlots) != len(cd.CodeSlots) {
			return 0, 0, fmt.Errorf("pre-encrypt slots count doesn't match code slots count")
		}

		hashSize := uint32(cd.HashType.Size())

		for i, slot := range runtime.PreEncryptSlots {
			if len(slot) < int(hashSize) {
				return 0, 0, fmt.Errorf("pre-encrypt slot %d contains a hash that is too small for the specified hash type", i)
			}

			body += hashSize
		}
	}

	return
}

func encodeRuntimeHeader(data any, _ *CodeDirectory, dst io.Writer, _, dataOffset uint32) (int64, error) {
	// Data type validation already done in calculateRuntimeSize
	runtime := data.(Runtime)

	var raw rawRuntime
	raw.Version = runtime.Version

	if len(runtime.PreEncryptSlots) > 0 {
		raw.Offset = dataOffset
	}

	if err := binary.Write(dst, binary.BigEndian, raw); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(raw)), nil
}

func encodeRuntimeBody(data any, cd *CodeDirectory, dst io.Writer) (writeCount int64, _ error) {
	// Data type validation already done in calculateRuntimeSize
	runtime := data.(Runtime)
	if len(runtime.PreEncryptSlots) == 0 {
		return 0, nil
	}

	hashSize := uint32(cd.HashType.Size())
	for i, slot := range runtime.PreEncryptSlots {
		n, err := dst.Write(slot[:hashSize])
		writeCount += int64(n)

		if err != nil {
			return writeCount, fmt.Errorf("pre-encrypt slot %d: %w", i, err)
		}
	}

	return
}

type RuntimeVersion struct {
	Major uint16
	Minor uint8
	Patch uint8
}

func (v RuntimeVersion) String() string {
	var builder strings.Builder
	builder.WriteString(strconv.Itoa(int(v.Major)))

	if v.Minor > 0 {
		builder.WriteByte('.')
		builder.WriteString(strconv.Itoa(int(v.Minor)))
	}

	if v.Patch > 0 {
		builder.WriteByte('.')
		builder.WriteString(strconv.Itoa(int(v.Patch)))
	}

	return builder.String()
}
