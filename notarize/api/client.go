package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/go-resty/resty/v2"
)

const (
	notaryApiV2Base = "https://appstoreconnect.apple.com/notary/v2"
)

type (
	RequestAuthentication func(req *resty.Request) error

	apiDoFunc func(req *resty.Request) (*resty.Response, error)
)

var (
	httpClient     *resty.Client
	httpClientOnce sync.Once
)

func getHttpClient() *resty.Client {
	httpClientOnce.Do(func() {
		httpClient = resty.New()
		httpClient.SetBaseURL(notaryApiV2Base)

		if trace, _ := strconv.ParseBool(os.Getenv("API_TRACE")); trace {
			httpClient.SetDebug(true)
		}
	})

	return httpClient
}

func newApiRequest[T any](ctx context.Context, auth RequestAuthentication, body any, do apiDoFunc) (*ApiResponse[T], error) {
	req := getHttpClient().NewRequest()
	req.SetContext(ctx)

	if err := auth(req); err != nil {
		return nil, fmt.Errorf("apply request authentication: %w", err)
	}

	req.SetError(new(ErrorResponse))
	req.SetHeader("Accept", "application/json")

	resp := new(ApiResponse[T])
	// Notary API returns responses with the content type
	// 'application/octet-stream', even though the response
	// is valid JSON, which breaks automatic JSON unmarshalling
	// in the Resty client so we instead force the correct
	// content type
	req.ForceContentType("application/json")
	req.SetResult(resp)

	if body != nil {
		req.SetHeader("Content-Type", "application/json")
		req.SetBody(body)
	}

	httpResp, err := do(req)
	switch {
	case err == nil && httpResp.IsError():
		err = fmt.Errorf("api error %d: %w", httpResp.StatusCode(), httpResp.Error().(*ErrorResponse).Errs())
		fallthrough

	case err != nil:
		return nil, fmt.Errorf("send api request: %w", err)

	default:
		return resp, nil
	}
}

func GetAppSpecificPasswordToken(ctx context.Context, auth RequestAuthentication) (*AppSpecificPasswordResponse, error) {
	apiResp, err := newApiRequest[AppSpecificPasswordResponse](ctx, auth, nil,
		func(req *resty.Request) (*resty.Response, error) {
			return req.Get("/asp")
		},
	)

	if err != nil {
		return nil, err
	}

	resp := apiResp.getAttributes()
	return &resp, nil
}

func StartNewSubmission(ctx context.Context, auth RequestAuthentication, subRequest *SubmissionRequest) (*SubmissionResponse, error) {
	apiResp, err := newApiRequest[SubmissionResponse](ctx, auth, subRequest,
		func(req *resty.Request) (*resty.Response, error) {
			return req.Post("/submissions")
		},
	)

	if err != nil {
		return nil, err
	}

	resp := apiResp.getAttributes()
	resp.Id = apiResp.Data.Id
	return &resp, nil
}

func GetSubmissionStatus(ctx context.Context, auth RequestAuthentication, submissionId string) (*SubmissionStatusResponse, error) {
	apiResp, err := newApiRequest[SubmissionStatusResponse](ctx, auth, nil,
		func(req *resty.Request) (*resty.Response, error) {
			req.SetPathParam("submissionId", submissionId)
			return req.Get("/submissions/{submissionId}")
		},
	)

	if err != nil {
		return nil, err
	}

	resp := apiResp.getAttributes()
	return &resp, nil
}

func GetSubmissionLogURL(ctx context.Context, auth RequestAuthentication, submissionId string) (*SubmissionLogURLResponse, error) {
	apiResp, err := newApiRequest[SubmissionLogURLResponse](ctx, auth, nil,
		func(req *resty.Request) (*resty.Response, error) {
			req.SetPathParam("submissionId", submissionId)
			return req.Get("/submissions/{submissionId}/logs")
		},
	)

	if err != nil {
		return nil, err
	}

	resp := apiResp.getAttributes()
	return &resp, nil
}

func DownloadNotaryLog(ctx context.Context, logUrl string) (*NotarizationLog, error) {
	req := getHttpClient().NewRequest()
	req.SetContext(ctx)

	req.SetHeader("Accept", "application/json")
	// Apple doesn't seem to set the content type
	// on notary logs when they upload them to S3
	// so we have to do the work for them
	req.ForceContentType("application/json")
	req.SetResult(new(NotarizationLog))

	httpResp, err := req.Get(logUrl)
	switch {
	case err == nil && httpResp.IsError():
		err = fmt.Errorf("api error %d", httpResp.StatusCode())
		fallthrough

	case err != nil:
		return nil, fmt.Errorf("send api request: %w", err)

	default:
		return httpResp.Result().(*NotarizationLog), nil
	}
}

func GetTickets(ctx context.Context, records []TicketRecord) ([]TicketRecord, error) {
	recs := &struct {
		Records []TicketRecord `json:"records"`
	}{records}

	req := getHttpClient().NewRequest()
	req.SetContext(ctx)
	req.SetHeaders(map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	})
	req.SetBody(recs)
	req.SetResult(recs)

	resp, err := req.Post("https://api.apple-cloudkit.com/database/1/com.apple.gk.ticket-delivery/production/public/records/lookup")
	switch {
	case err == nil && resp.IsError():
		err = fmt.Errorf("api error: %s (%d)", resp.Status(), resp.StatusCode())
		fallthrough

	case err != nil:
		return nil, fmt.Errorf("request records from CloudKit API: %w", err)

	case len(recs.Records) == 0:
		return nil, errors.New("empty response received")

	default:
		return recs.Records, nil
	}
}
