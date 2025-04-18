package api

import (
	"encoding/json"
	"fmt"
)

type DigestAlgorithm uint8

const (
	DigestAlgorithmInvalid DigestAlgorithm = iota
	DigestAlgorithmSHA1
	DigestAlgorithmSHA256
)

var (
	digestAlgorithmToName = map[DigestAlgorithm]string{
		DigestAlgorithmSHA1:   "SHA-1",
		DigestAlgorithmSHA256: "SHA-256",
	}
)

func (algo DigestAlgorithm) String() string {
	if name, known := digestAlgorithmToName[algo]; known {
		return name
	}

	return fmt.Sprintf("DIGEST_ALGORITHIM(%d)", uint8(algo))
}

func (algo *DigestAlgorithm) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err != nil {
		return err
	}

	for value, knownName := range digestAlgorithmToName {
		if name == knownName {
			*algo = value
			return nil
		}
	}

	*algo = DigestAlgorithmInvalid
	return nil
}

type NotarizationLog struct {
	JobID           string               `json:"jobId"`
	Status          string               `json:"status"`
	StatusSummary   string               `json:"statusSummary"`
	StatusCode      int                  `json:"statusCode"`
	ArchiveFilename string               `json:"archiveFilename"`
	UploadDate      string               `json:"uploadDate"`
	SHA256          string               `json:"sha256"`
	TicketContents  []NotarizationTicket `json:"ticketContents"`
	Issues          []NotarizationIssue  `json:"issues"`
}

type NotarizationTicket struct {
	Path            string          `json:"path"`
	DigestAlgorithm DigestAlgorithm `json:"digestAlgorithm"`
	CDHash          string          `json:"cdhash"`
	Arch            string          `json:"arch"`
}

func (ticket *NotarizationTicket) RecordName() string {
	return fmt.Sprintf("2/%d/%s", uint8(ticket.DigestAlgorithm), ticket.CDHash)
}

type NotarizationIssue struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Path         string `json:"path"`
	Message      string `json:"message"`
	DocUrl       string `json:"docUrl"`
	Architecture string `json:"architecture"`
}
