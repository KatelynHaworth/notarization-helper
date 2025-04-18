package code_directory

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

var (
	SupportsVersionExecSeg = RegisterSupportsVersion(SupportsMetadata{
		Version:              0x020400,
		FullName:             "CODEDIRECTORY_SUPPORTS_EXECSEG",
		ShortName:            "exec_seg",
		Decoder:              decodeExecSegment,
		EncodeSizeCalculator: calculateExecSegmentSize,
		HeaderEncoder:        encodeExecSegmentHeader,
	})
)

type ExecSegment struct {
	SegmentBase  uint64
	SegmentLimit uint64
	Flags        ExecSegmentFlag
}

func decodeExecSegment(_ *CodeDirectory, src *io.SectionReader) (any, error) {
	var seg ExecSegment
	if err := binary.Read(src, binary.BigEndian, &seg); err != nil {
		return nil, fmt.Errorf("read raw: %w", err)
	}

	return seg, nil
}

func calculateExecSegmentSize(data any, _ *CodeDirectory) (uint32, uint32, error) {
	if _, ok := data.(ExecSegment); !ok {
		return 0, 0, fmt.Errorf("unexpected data type %T", data)
	}

	return uint32(binary.Size(ExecSegment{})), 0, nil
}

func encodeExecSegmentHeader(data any, _ *CodeDirectory, dst io.Writer, _, _ uint32) (int64, error) {
	// Data type validation already done in calculateExecSegmentSize
	seg := data.(ExecSegment)

	if err := binary.Write(dst, binary.BigEndian, seg); err != nil {
		return 0, fmt.Errorf("write raw: %w", err)
	}

	return int64(binary.Size(seg)), nil
}

type ExecSegmentFlag uint64

const (
	ExecSegmentFlagMainBinary     ExecSegmentFlag = 0x01
	ExecSegmentFlagsAllowUnsigned ExecSegmentFlag = 0x10 << (iota - 1)
	ExecSegmentFlagDebugger
	ExecSegmentFlagJIT
	ExecSegmentFlagSkipLV
	ExecSegmentFlagCanLoadCDHASH
	ExecSegmentFlagCanExecCDHASH
)

var segmentFlagToName = map[ExecSegmentFlag]string{
	ExecSegmentFlagMainBinary:     "MAIN_BINARY",
	ExecSegmentFlagsAllowUnsigned: "ALLOW_UNSIGNED",
	ExecSegmentFlagDebugger:       "DEBUGGER",
	ExecSegmentFlagJIT:            "JIT",
	ExecSegmentFlagSkipLV:         "SKIP_LV",
	ExecSegmentFlagCanLoadCDHASH:  "LOAD_CDHASH",
	ExecSegmentFlagCanExecCDHASH:  "EXEC_CDHASH",
}

func (flags ExecSegmentFlag) String() string {
	var flagNames []string

	for flag, name := range segmentFlagToName {
		if flags&flag == flag {
			flagNames = append(flagNames, name)
		}
	}

	return fmt.Sprintf("0x%d (%s)", uint64(flags), strings.Join(flagNames, ","))
}

func (flags *ExecSegmentFlag) Set(flag ExecSegmentFlag) {
	*flags |= flag
}
