package code_directory

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	SupportsVersionScatter = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020100,
		FullName:             "CODEDIRECTORY_SUPPORTS_SCATTER",
		ShortName:            "scatter",
		Decoder:              decodeScatter,
		EncodeSizeCalculator: calculateScatterSize,
		HeaderEncoder:        encodeScatterHeader,
		BodyEncoder:          encodeScatterBody,
	})

	scatterSize = uint32(binary.Size(Scatter{}))
)

type (
	Scatter struct {
		Count        uint32
		Base         uint32
		TargetOffset uint64
		_            uint64 // reserved
	}

	rawScatter struct {
		Offset uint32
	}
)

func decodeScatter(cd *CodeDirectory, src *io.SectionReader) (any, error) {
	var raw rawScatter

	err := binary.Read(src, binary.BigEndian, &raw)
	switch {
	case err != nil:
		return nil, fmt.Errorf("read raw: %w", err)

	case cd.hdr.Length < raw.Offset:
		return nil, fmt.Errorf("offset overflows past blob length")

	case raw.Offset == 0:
		return nil, nil
	}

	var (
		readPos    = raw.Offset
		readBuf    = make([]byte, scatterSize)
		scatterSet = make(ScatterSet, 0)
	)

	// Check each scatter buffer, since we don't know the
	// length of the scatter buffer array, we have to
	// check each entry.
	for cd.hdr.Length-readPos > scatterSize {
		if _, err = src.ReadAt(readBuf, int64(readPos)); err != nil {
			return nil, fmt.Errorf("read scatter at offset %d: %w", readPos, err)
		}

		var scatter Scatter
		if _, err = binary.Decode(readBuf, binary.BigEndian, &scatter); err != nil {
			return nil, fmt.Errorf("decode scatter at offset %d: %w", readPos, err)
		}

		if scatter.Count == 0 {
			break
		}

		readPos += scatterSize
		scatterSet = append(scatterSet, scatter)
	}

	if pages, err := scatterSet.totalCount(); err != nil {
		return nil, err
	} else if pages != uint32(len(cd.CodeSlots)) {
		return nil, fmt.Errorf("scatter pages (%d) doesn't match number of code slots (%d)", pages, len(cd.CodeSlots))
	}

	return scatterSet, nil
}

func calculateScatterSize(data any, cd *CodeDirectory) (hdr, body uint32, _ error) {
	hdr = uint32(binary.Size(rawScatter{}))

	if data != nil {
		set, ok := data.(ScatterSet)
		if ok {
			return 0, 0, fmt.Errorf("unexpected data type %T", data)
		}

		if pages, err := set.totalCount(); err != nil {
			return 0, 0, err
		} else if pages != uint32(len(cd.CodeSlots)) {
			return 0, 0, fmt.Errorf("scatter pages (%d) doesn't match number of code slots (%d)", pages, len(cd.CodeSlots))
		}

		body = uint32(len(set)) * scatterSize

		if !set.hasSentinel() {
			body += scatterSize
		}
	}

	return
}

func encodeScatterHeader(data any, _ *CodeDirectory, dst io.Writer, _, dataOffset uint32) (int64, error) {
	var raw rawScatter

	if data != nil {
		// Data type validation already done in calculateScatterSize
		raw.Offset = dataOffset
	}

	if err := binary.Write(dst, binary.BigEndian, raw); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(raw)), nil
}

func encodeScatterBody(data any, _ *CodeDirectory, dst io.Writer) (writeCounter int64, _ error) {
	if data == nil {
		return
	}

	// Data type validation already done in calculateScatterSize
	scatterSet := data.(ScatterSet)
	if !scatterSet.hasSentinel() {
		scatterSet = append(scatterSet, Scatter{})
	}

	for i, scatter := range scatterSet {
		if err := binary.Write(dst, binary.BigEndian, scatter); err != nil {
			return writeCounter, fmt.Errorf("encode scattter %d: %w", i, err)
		}

		writeCounter += int64(scatterSize)
	}

	return
}

type ScatterSet []Scatter

func (set ScatterSet) totalCount() (uint32, error) {
	var pages uint32

	for i, scatter := range set {
		if pages+scatter.Count < pages {
			return 0, fmt.Errorf("scatter %d has a count that causes an overflow", i)
		}

		pages += scatter.Count
	}

	return pages, nil
}

func (set ScatterSet) hasSentinel() bool {
	return set[len(set)-1].Count == 0
}
