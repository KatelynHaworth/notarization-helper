package code_directory

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var (
	SupportsVersionTeamID = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020200,
		FullName:             "CODEDIRECTORY_SUPPORTS_TEAMID",
		ShortName:            "team_id",
		Decoder:              decodeTeamId,
		EncodeSizeCalculator: calculateTeamIDSize,
		HeaderEncoder:        encodeTeamIdHeader,
	})
)

type rawTeamId struct {
	Offset uint32
}

func decodeTeamId(cd *CodeDirectory, src *io.SectionReader) (any, error) {
	var raw rawTeamId

	err := binary.Read(src, binary.BigEndian, &raw)
	switch {
	case err != nil:
		return nil, fmt.Errorf("read raw: %w", err)

	case cd.hdr.Length < raw.Offset:
		return nil, fmt.Errorf("offset overflows past blob length")

	case raw.Offset == 0:
		return nil, nil
	}

	teamIdBuf := make([]byte, cd.hdr.Length-raw.Offset)
	if _, err = src.ReadAt(teamIdBuf, int64(raw.Offset)); err != nil {
		return nil, fmt.Errorf("read data from offset: %w", err)
	}

	if nullIndex := bytes.IndexByte(teamIdBuf, 0x0); nullIndex < 0 {
		return nil, fmt.Errorf("string not found at specified offset %d", raw.Offset)
	} else {
		return string(teamIdBuf[:nullIndex]), nil
	}
}

func calculateTeamIDSize(data any, _ *CodeDirectory) (hdr, body uint32, _ error) {
	hdr = uint32(binary.Size(rawTeamId{}))

	if data != nil {
		teamId, ok := data.(string)
		if !ok {
			return 0, 0, fmt.Errorf("unexpected data type %T", data)
		}

		body = uint32(len(teamId) + 1 /* null terminated */)
	}

	return
}

func encodeTeamIdHeader(data any, cd *CodeDirectory, dst io.Writer, dataOffset, _ uint32) (int64, error) {
	var raw rawTeamId

	if data != nil {
		// Data type validation already done in calculateTeamIDSize

		// See the comment in encodeBaseHeader
		identityEnd := uint32(len(cd.Identity)) + 1
		raw.Offset = dataOffset + identityEnd
	}

	if err := binary.Write(dst, binary.BigEndian, raw); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(raw)), nil
}
