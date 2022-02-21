package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

const (
	itmspMetadataTemplateSource = `<?xml version="1.0" encoding="UTF-8"?>
<package version="software5.9" xmlns="http://apple.com/itunes/importer">
    <software_assets
        app_platform="osx"
        primary_bundle_identifier="{{ .PrimaryBundleId }}">
        <asset type="developer-id-package">
            <data_file>
                <file_name>{{ .FileName }}</file_name>
                <checksum type="md5">{{ .Checksum }}</checksum>
                <size>{{ .ByteSize }}</size>
            </data_file>
        </asset>
    </software_assets>
</package>`
)

var (
	itmspMetadataTemplate = template.Must(template.New("metadata").Parse(itmspMetadataTemplateSource))
)

type itmspMetadata struct {
	PrimaryBundleId string
	FileName        string
	Checksum        string
	ByteSize        int
}

func (metadata itmspMetadata) WriteTo(dst io.Writer) (int64, error) {
	return 1, itmspMetadataTemplate.Execute(dst, metadata)
}

func (worker *Worker) uploadForNotarization() (*NotarizationUpload, error) {
	if len(worker.zipFile) > 0 {
		defer os.Remove(worker.zipFile)
	}

	if strings.HasPrefix(worker.config.Password, "@keychain") {
		return nil, errors.New("usage of '@keychain' in password is not supported on linux")
	}

	itmspPackage, err := worker.createITMSP()
	if err != nil {
		return nil, errors.Wrap(err, "create ITMSP package")
	}
	defer os.RemoveAll(itmspPackage)

	cmd := exec.Command("iTMSTransporter")
	cmd.Args = append(cmd.Args,
		"-m", "upload",
		"-vp", "json",
		"-f", itmspPackage,
		"-u", worker.config.Username,
		"-p", worker.config.Password,
		"-distribution", "DeveloperId",
		"-primaryBundleId", worker.target.BundleID,
	)

	if len(worker.config.TeamID) > 0 {
		cmd.Args = append(cmd.Args, "-itc_provider", worker.config.TeamID)
	}

	cmdOut, err := cmd.CombinedOutput()
	output, err2 := worker.processITMSOutput(string(cmdOut))
	if err2 != nil {
		return nil, errors.Wrap(err, "unmarshal command output")
	}

	switch {
	case len(output.ProductErrors) > 0:
		return nil, errors.Wrap(output.ProductErrors[0], "execute iTMSTransporter")

	case err != nil:
		return nil, errors.Wrap(err, "execute iTMSTransporter")

	default:
		return &output.Upload, nil
	}
}

func (worker *Worker) createITMSP() (string, error) {
	itmspDir, err := os.MkdirTemp("", "*.itmsp")
	if err != nil {
		return "", errors.Wrap(err, "create temporary itmsp package directory")
	}

	srcFile := worker.target.File
	if len(worker.zipFile) > 0 {
		srcFile = worker.zipFile
	}

	file, err := os.Open(srcFile)
	if err != nil {
		return "", errors.Wrap(err, "open source file")
	}
	defer file.Close()

	dstFile, err := os.OpenFile(filepath.Join(itmspDir, filepath.Base(srcFile)), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", errors.Wrap(err, "create temporary file in itmsp package")
	}
	defer dstFile.Close()

	hash := md5.New()
	dst := io.MultiWriter(hash, dstFile)

	byteSize, err := io.Copy(dst, file)
	if err != nil {
		return "", errors.Wrap(err, "copy file into itmsp package")
	}

	metadata := itmspMetadata{
		PrimaryBundleId: worker.target.BundleID,
		FileName:        filepath.Base(srcFile),
		Checksum:        hex.EncodeToString(hash.Sum(nil)),
		ByteSize:        int(byteSize),
	}

	metadataFile, err := os.OpenFile(filepath.Join(itmspDir, "metadata.xml"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return "", errors.Wrap(err, "create metadata file in itmsp package")
	}
	defer metadataFile.Close()

	if _, err = metadata.WriteTo(metadataFile); err != nil {
		return "", errors.Wrap(err, "write itmsp package metadata")
	}

	return itmspDir, nil
}

func (*Worker) processITMSOutput(output string) (*CommandOutput, error) {
	cmdOut := new(CommandOutput)
	if len(output) == 0 {
		return cmdOut, nil
	}

	if jsonIndex := strings.Index(output, "JSON-START>>"); jsonIndex > -1 {
		output = output[jsonIndex+len("JSON-START>>"):]
		output = output[:strings.Index(output, "<<JSON-END")]

		outputData := struct {
			Results struct {
				UploadId string `json:"upload_id"`
			} `json:"dev-id-results"`
		}{}

		if err := json.Unmarshal([]byte(output), &outputData); err != nil {
			return nil, errors.Wrap(err, "parse json output")
		}

		cmdOut.Upload.RequestUUID = outputData.Results.UploadId
		return cmdOut, nil
	}

	if errorIndex := strings.Index(output, "ERROR: ERROR ITMS-"); errorIndex > 0 {
		output = output[errorIndex:]
		output = output[:strings.IndexByte(output, '\n')]

		errMessage := output[strings.IndexByte(output, '"')+1:]
		errMessage = errMessage[:strings.LastIndexByte(errMessage, '"')]

		cmdOut.ProductErrors = []ProductError{{
			Message: errMessage,
		}}

		return cmdOut, nil
	}

	return nil, errors.New("found neither a success or error in ITMS output")
}
