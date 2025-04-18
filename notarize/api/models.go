package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Apple is lying about the format of
// an error response in their API docs
type ErrorResponse struct {
	Errors []ErrorResponseErr
}

func (errResp ErrorResponse) Errs() error {
	errs := make([]error, len(errResp.Errors))
	for i, err := range errResp.Errors {
		errs[i] = err
	}

	return errors.Join(errs...)
}

type ErrorResponseErr struct {
	ID     string
	Status string
	Code   string
	Title  string
	Detail string
}

func (err ErrorResponseErr) Error() string {
	return fmt.Sprintf("(%s) %s: %s", err.Code, err.Title, err.Detail)
}

type ApiResponse[T any] struct {
	Data struct {
		Id         string `json:"id"`
		Type       string `json:"type"`
		Attributes T      `json:"attributes"`
	} `json:"data"`
}

func (resp *ApiResponse[T]) getAttributes() T {
	return resp.Data.Attributes
}

type AppSpecificPasswordResponse struct {
	Token string `json:"token"`
}

type SubmissionRequest struct {
	Name string `json:"submissionName"`
	Hash string `json:"sha256"`
}

type SubmissionResponse struct {
	Id                 string `json:"-"`
	AwsAccessKeyId     string `json:"awsAccessKeyId"`
	AwsSecretAccessKey string `json:"awsSecretAccessKey"`
	AwsSessionToken    string `json:"awsSessionToken"`
	Bucket             string `json:"bucket"`
	Object             string `json:"object"`
}

func (resp *SubmissionResponse) Retrieve(_ context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     resp.AwsAccessKeyId,
		SecretAccessKey: resp.AwsSecretAccessKey,
		SessionToken:    resp.AwsSessionToken,
	}, nil
}

type SubmissionStatusResponse struct {
	CreatedDate time.Time             `json:"createdDate"`
	Name        string                `json:"name"`
	Status      SubmissionStatusState `json:"status"`
}

type SubmissionStatusState uint8

const (
	SubmissionStatusStateAccepted SubmissionStatusState = iota
	SubmissionStatusStateInProgress
	SubmissionStatusStateInvalid
	SubmissionStatusStateRejected
)

var (
	submissionStatusStringToVal = map[string]SubmissionStatusState{
		"accepted":    SubmissionStatusStateAccepted,
		"in progress": SubmissionStatusStateInProgress,
		"invalid":     SubmissionStatusStateInvalid,
		"rejected":    SubmissionStatusStateRejected,
	}

	submissionStatusValToString = map[SubmissionStatusState]string{
		SubmissionStatusStateAccepted:   "Accepted",
		SubmissionStatusStateInProgress: "InProgress",
		SubmissionStatusStateInvalid:    "Invalid",
		SubmissionStatusStateRejected:   "Rejected",
	}
)

func (state *SubmissionStatusState) String() string {
	if name, known := submissionStatusValToString[*state]; known {
		return name
	}

	return fmt.Sprintf("UNKNOWN(%d)", state)
}

func (state *SubmissionStatusState) UnmarshalJSON(b []byte) error {
	var name string
	if err := json.Unmarshal(b, &name); err != nil {
		return err
	}

	if val, known := submissionStatusStringToVal[strings.ToLower(name)]; known {
		*state = val
	} else {
		return fmt.Errorf("invalid submission status state: %s", name)
	}

	return nil
}

type SubmissionLogURLResponse struct {
	DeveloperLogUrl string `json:"developerLogUrl"`
}
