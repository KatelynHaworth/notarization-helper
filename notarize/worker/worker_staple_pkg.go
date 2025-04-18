package worker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
)

var (
	packageTrailerMagic = [4]byte{0x74, 0x38, 0x6c, 0x72}
)

type packageTrailerType uint16

const (
	packageTrailerTypeInvalid = iota
	packageTrailerTypeTerminator
	packageTrailerTypeTicket
)

type packageTrailer struct {
	Magic   [4]uint8
	Version uint16
	Type    packageTrailerType
	Length  uint32
	_       uint32
}

func (worker *Worker) stapleToPkg(ticket api.SignedTicket) error {
	var buf bytes.Buffer

	trailer := packageTrailer{Magic: packageTrailerMagic, Version: 1, Type: packageTrailerTypeTerminator, Length: 0}
	if err := binary.Write(&buf, binary.LittleEndian, trailer); err != nil {
		return fmt.Errorf("encoded xar terminator trailer: %w", err)
	}

	buf.Write(ticket.Value)

	trailer = packageTrailer{Magic: packageTrailerMagic, Version: 1, Type: packageTrailerTypeTicket, Length: uint32(len(ticket.Value))}
	if err := binary.Write(&buf, binary.LittleEndian, trailer); err != nil {
		return fmt.Errorf("encoded xar ticket trailer: %w", err)
	}

	packageFile, err := os.OpenFile(worker.target.File, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open package file: %w", err)
	}
	defer packageFile.Close()

	if _, err = buf.WriteTo(packageFile); err != nil {
		return fmt.Errorf("append ticket to package: %w", err)
	}

	return nil
}
