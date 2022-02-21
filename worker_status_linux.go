package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"howett.net/plist"
)

const (
	iTunesSoftwareServiceURL = "https://contentdelivery.itunes.apple.com/WebObjects/MZLabelService.woa/json/MZITunesSoftwareService"
	iTunesProducerServiceURL = "https://contentdelivery.itunes.apple.com/WebObjects/MZLabelService.woa/json/MZITunesProducerService"
)

type devInfoRequestStatus int

const (
	devInfoRequestStatusInvalid    devInfoRequestStatus = -2
	devInfoRequestStatusSuccess    devInfoRequestStatus = 0
	devInfoRequestStatusInProgress devInfoRequestStatus = 1
)

var devInfoRequestStatusToString = map[devInfoRequestStatus]string{
	devInfoRequestStatusSuccess:    "success",
	devInfoRequestStatusInProgress: "in progress",
	devInfoRequestStatusInvalid:    "invalid",
}

func (code devInfoRequestStatus) String() string {
	if value, exists := devInfoRequestStatusToString[code]; exists {
		return value
	}

	return fmt.Sprintf("unknown(%d)", code)
}

type tokenRequest struct {
	Application         string `json:"Application"`
	ApplicationBundleId string `json:"ApplicationBundleId"`
	Username            string `json:"Username"`
	Password            string `json:"Password"`
}

type tokenResponse struct {
	DSTokenCookieName string `json:"DSTokenCookieName"`
	DSToken           string `json:"DSToken"`
}

type devInfoRequest struct {
	Application         string `json:"Application"`
	ApplicationBundleId string `json:"ApplicationBundleId"`
	DsPlist             string `json:"DS_PLIST"`
	RequestUUID         string `json:"RequestUUID"`
}

type devInfoResponse struct {
	DevIDPlus struct {
		LogFileURL string `json:"LogFileURL"`
		MoreInfo   struct {
			Hash string `json:"Hash"`
		} `json:"MoreInfo"`
		DateStr       string               `json:"DateStr"`
		StatusCode    int                  `json:"StatusCode"`
		RequestStatus devInfoRequestStatus `json:"RequestStatus"`
		StatusMessage string               `json:"StatusMessage"`
	} `json:"DevIDPlus"`
}

func (worker *Worker) getNotarizationStatus(upload *NotarizationUpload) (*NotarizationInfo, error) {
	var tokenResp tokenResponse
	tokenReq := &tokenRequest{
		Application:         "notarization-helper",
		ApplicationBundleId: "au.id.haworth.NotarizationHelper",
		Username:            worker.config.Username,
		Password:            worker.config.Password,
	}

	if err := worker.jsonRpcSendRequest(iTunesSoftwareServiceURL, "generateAppleConnectToken", tokenReq, &tokenResp); err != nil {
		return nil, errors.Wrap(err, "authenticate with Apple")
	}

	dsPlist, err := plist.Marshal(map[string]string{tokenResp.DSTokenCookieName: tokenResp.DSToken}, plist.XMLFormat)
	if err != nil {
		return nil, errors.Wrap(err, "generate plist from authentication token")
	}

	var devInfoResp devInfoResponse
	devInfoReq := &devInfoRequest{
		Application:         "notarization-helper",
		ApplicationBundleId: "au.id.haworth.NotarizationHelper",
		DsPlist:             string(dsPlist),
		RequestUUID:         upload.RequestUUID,
	}

	if err := worker.jsonRpcSendRequest(iTunesProducerServiceURL, "developerIDPlusInfoForPackageWithArguments", devInfoReq, &devInfoResp); err != nil {
		return nil, errors.Wrap(err, "authenticate with Apple")
	}

	info := new(NotarizationInfo)
	info.Date, _ = time.Parse(time.RFC3339, devInfoResp.DevIDPlus.DateStr)
	info.LogFileURL = devInfoResp.DevIDPlus.LogFileURL
	info.Hash = devInfoResp.DevIDPlus.MoreInfo.Hash
	info.RequestUUID = upload.RequestUUID
	info.Status = devInfoResp.DevIDPlus.RequestStatus.String()
	info.StatusCode = devInfoResp.DevIDPlus.StatusCode
	info.StatusMessage = devInfoResp.DevIDPlus.StatusMessage

	return info, nil
}
