package codesign

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/code_directory"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/super_blob"
)

const LoadCmdCodeSignature = macho.LoadCmd(0x1d)

var (
	ErrNoCodeSignature = errors.New("code signature not found in file")

	codeSignatureCmdSize = int(unsafe.Sizeof(CodeSignatureCmd{}))
)

type CodeSignatureCmd struct {
	macho.LoadCmd
	_      uint32
	Offset uint32
	Size   uint32
}

type readAtSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

func FindCodeSignatureInFile(r readAtSeeker) (Blob, []byte, *CodeSignatureCmd, error) {
	var file *macho.File

	if _, err := macho.NewFatFile(r); errors.Is(err, macho.ErrNotFat) {
		file, err = macho.NewFile(r)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("read macho file: %w", err)
		}
	} else if err != nil {
		return nil, nil, nil, fmt.Errorf("read macho fat file: %w", err)
	} else {
		// TODO: Determine how to pick a file from within a fat mach-o
		return nil, nil, nil, fmt.Errorf("fat macho files not currently supported")
	}

	cmd := findCodeSignatureCmd(file)
	if cmd == nil {
		return nil, nil, nil, ErrNoCodeSignature
	}

	blob, raw, err := cmd.LoadBlob(r)
	if err != nil {
		return nil, nil, cmd, err
	}

	switch t := blob.(type) {
	case *code_directory.CodeDirectory:
		return t, raw, cmd, nil

	case *super_blob.SuperBlob:
		return t, raw, cmd, nil
	}

	return nil, raw, cmd, io.ErrUnexpectedEOF
}

func findCodeSignatureCmd(file *macho.File) *CodeSignatureCmd {
	// TODO: Get this command added in 'debug/macho' directly
	//       so we don't have to go digging for it manually

	for _, load := range file.Loads {
		bytes, ok := load.(macho.LoadBytes)
		if !ok || len(bytes.Raw()) != codeSignatureCmdSize {
			continue
		}

		var cmd CodeSignatureCmd
		if _, err := binary.Decode(bytes.Raw(), file.ByteOrder, &cmd); err != nil {
			continue
		} else if cmd.LoadCmd == LoadCmdCodeSignature {
			return &cmd
		}
	}

	return nil
}

func (cmd *CodeSignatureCmd) String() string {
	return fmt.Sprintf("LC_CODE_SIGNATURE(%s) - Data Offset: %d, Data Size: %d", cmd.LoadCmd, cmd.Offset, cmd.Size)
}

func (cmd *CodeSignatureCmd) LoadBlob(r io.ReaderAt) (Blob, []byte, error) {
	sec := io.NewSectionReader(r, int64(cmd.Offset), int64(cmd.Size))
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, sec)
	_, _ = sec.Seek(0, io.SeekStart)

	//blob, err := ReadFrom(sec)
	//if err != nil {
	//	return nil, nil, fmt.Errorf("read code signature blob: %w", err)
	//}

	//return blob, buf.Bytes(), nil
	return nil, nil, nil
}
