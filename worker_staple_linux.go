package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const (
	packageXarHeaderSize = 28
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

func (worker *Worker) canStaple() bool {
	stat, _ := os.Stat(worker.target.File)
	switch filepath.Ext(worker.target.File) {
	case ".dmg":
		// TODO(lh): Need to determine how to do this
		return false

	case ".pkg":
		return true

	case ".kext":
		fallthrough
	case ".app":
		return stat.IsDir()

	default:
		return false
	}
}

func (worker *Worker) staplePackage() error {
	switch {
	case !worker.target.Staple:
		return nil

	case !worker.canStaple():
		worker.logger.Warn().Msg("This file type is not supported for stapling, continuing without stapling")
		return nil

	default:
		worker.logger.Info().Msg("Stapling notarization ticket to package")
	}

	hashType, cdHash, err := worker.getCodeDirectoryHash()
	if err != nil {
		return errors.Wrap(err, "get code directory hash")
	}

	ticket, err := worker.getNotarizationTicket(hashType, cdHash)
	if err != nil {
		return errors.Wrap(err, "get notarization ticket")
	}

	switch filepath.Ext(worker.target.File) {
	case ".pkg":
		return worker.staplePkgTicket(ticket)

	case ".kext":
		fallthrough
	case ".app":
		return worker.stapleFileTicket(ticket)

	default:
		return errors.New("unsupported file type")
	}
}

func (worker *Worker) getCodeDirectoryHash() (int, string, error) {
	switch filepath.Ext(worker.target.File) {
	case ".pkg":
		return worker.getPackageHash()

	case ".kext":
		fallthrough
	case ".app":
		return worker.getCodeDirectoryFileHash()

	default:
		return 0, "", errors.New("unsupported file type")
	}
}

func (worker *Worker) getCodeDirectoryFileHash() (int, string, error) {
	cdFile, err := os.Open(filepath.Join(worker.target.File, "Contents", "_CodeSignature", "CodeDirectory"))
	if err != nil {
		return 0, "", errors.Wrap(err, "open code directory file")
	}
	defer cdFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, cdFile); err != nil {
		return 0, "", errors.Wrap(err, "read code directory file")
	}

	hash := hasher.Sum(nil)
	return 2, hex.EncodeToString(hash[:20]), nil
}

func (worker *Worker) getPackageHash() (int, string, error) {
	packageFile, err := os.Open(worker.target.File)
	if err != nil {
		return 0, "", errors.Wrap(err, "open package file")
	}
	defer packageFile.Close()

	hdr := make([]byte, packageXarHeaderSize)
	if _, err = packageFile.Read(hdr); err != nil {
		return 0, "", errors.Wrap(err, "read package header")
	}

	compressedTocSize := binary.BigEndian.Uint64(hdr[8:16])
	compressedToc := make([]byte, compressedTocSize)
	if _, err = packageFile.Read(compressedToc); err != nil {
		return 0, "", errors.Wrap(err, "read package table of contents")
	}

	compressedReader, err := zlib.NewReader(bytes.NewBuffer(compressedToc))
	if err != nil {
		return 0, "", errors.Wrap(err, "decompress package table of contents")
	}

	toc := struct {
		XMLName xml.Name `xml:"xar"`
		Toc     struct {
			XMLName  xml.Name `xml:"toc"`
			Checksum struct {
				XMLName xml.Name `xml:"checksum"`
				Style   string   `xml:"style,attr"`
				Offset  int64    `xml:"offset"`
				Size    int64    `xml:"size"`
			}
		}
	}{}

	decoder := xml.NewDecoder(compressedReader)
	decoder.Strict = false
	if err = decoder.Decode(&toc); err != nil {
		return 0, "", errors.Wrap(err, "decode package table of contents")
	}

	heapOffset := packageXarHeaderSize + int64(compressedTocSize)
	checksum := make([]byte, toc.Toc.Checksum.Size)
	if _, err := io.ReadFull(io.NewSectionReader(packageFile, heapOffset+toc.Toc.Checksum.Offset, toc.Toc.Checksum.Size), checksum); err != nil {
		return 0, "", errors.Wrap(err, "read package checksum")
	}

	return 1, hex.EncodeToString(checksum), nil
}

func (worker *Worker) stapleFileTicket(ticket []byte) error {
	return os.WriteFile(filepath.Join(worker.target.File, "Contents", "CodeResources"), ticket, 0644)
}

func (worker *Worker) staplePkgTicket(ticket []byte) error {
	var buf bytes.Buffer

	trailer := packageTrailer{Magic: packageTrailerMagic, Version: 1, Type: packageTrailerTypeTerminator, Length: 0}
	if err := binary.Write(&buf, binary.LittleEndian, trailer); err != nil {
		return errors.Wrap(err, "encoded xar terminator trailer")
	}

	buf.Write(ticket)

	trailer = packageTrailer{Magic: packageTrailerMagic, Version: 1, Type: packageTrailerTypeTicket, Length: uint32(len(ticket))}
	if err := binary.Write(&buf, binary.LittleEndian, trailer); err != nil {
		return errors.Wrap(err, "encoded xar ticket trailer")
	}

	packageFile, err := os.OpenFile(worker.target.File, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "open package file")
	}
	defer packageFile.Close()

	if _, err := buf.WriteTo(packageFile); err != nil {
		return errors.Wrap(err, "append ticket to package")
	}

	return nil
}

func (*Worker) getNotarizationTicket(hashType int, cdHash string) ([]byte, error) {
	result := &struct {
		Records []struct {
			ErrorCode string `json:"serverErrorCode"`
			Fields    struct {
				SignedTicket struct {
					Value []byte `json:"value"`
				} `json:"signedTicket"`
			} `json:"fields"`
		} `json:"records"`
	}{}

	request := resty.New().NewRequest()
	request.SetHeaders(map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	})
	request.SetResult(result)
	request.SetBody(map[string]interface{}{
		"records": []map[string]interface{}{{
			"recordName": fmt.Sprintf("2/%d/%s", hashType, cdHash),
		}},
	})

	resp, err := request.Post("https://api.apple-cloudkit.com/database/1/com.apple.gk.ticket-delivery/production/public/records/lookup")
	switch {
	case err == nil && resp.StatusCode() != http.StatusOK:
		err = fmt.Errorf("unexpected response code: %s", resp.Status())
		fallthrough

	case err != nil:
		return nil, errors.Wrap(err, "query Apple public records")

	case len(result.Records) == 0:
		return nil, errors.New("empty result received")

	case result.Records[0].ErrorCode != "":
		return nil, errors.Errorf("unable to retrieve notarization ticket: %s", result.Records[0].ErrorCode)

	default:
		return result.Records[0].Fields.SignedTicket.Value, nil
	}
}
