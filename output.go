package main

import "time"

type CommandOutput struct {
	OSVersion string `plist:"os-version"`
	SuccessMessage string `plist:"success-message"`
	ToolPath string `plist:"tool-path"`
	ToolVersion string `plist:"tool-version"`
	ProductErrors []ProductError `plist:"product-errors"`
	Upload NotarizationUpload `plist:"notarization-upload"`
	Info NotarizationInfo `plist:"notarization-info"`
}

type NotarizationUpload struct {
	RequestUUID string `plist:"RequestUUID"`
}

type NotarizationInfo struct {
	Date time.Time `plist:"Date"`
	Hash string `plist:"Hash"`
	LogFileURL string `plist:"LogFileURL"`
	RequestUUID string `plist:"RequestUUID"`
	Status string `plist:"Status"`
	StatusCode int `plist:"Status Code"`
	StatusMessage string `plist:"Status Message"`
}

type ProductError struct {
	Code int `plist:"code"`
	Message string `plist:"message"`
	UserInfo map[string]string `plist:"userInfo"`
}

func (error ProductError) Error() string {
	return error.Message
}

type NotarizationLog struct {
	JobID string `json:"jobId"`
	Status string `json:"status"`
	StatusSummary string `json:"statusSummary"`
	StatusCode int `json:"statusCode"`
	ArchiveFilename string `json:"archiveFilename"`
	UploadDate string `json:"uploadDate"`
	SHA256 string `json:"sha256"`
	TicketContents []NotarizationTicket `json:"ticketContents"`
	Issues []NotarizationIssue `json:"issues"`
}

type NotarizationTicket struct {
	Path string `json:"path"`
	DigestAlgorithm string `json:"digestAlgorithm"`
	CDHash string `json:"cdhash"`
	Arch string `json:"arch"`
}

type NotarizationIssue struct {
	Severity string `json:"severity"`
	Code string `json:"code"`
	Path string `json:"path"`
	Message string `json:"message"`
	DocUrl string `json:"docUrl"`
	Architecture string `json:"architecture"`
}