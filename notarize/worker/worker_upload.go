package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	submissionStatusPollInterval = 5 * time.Second
	submissionStatusPollJitter   = 5
)

func (worker *Worker) UploadAndWait(ctx context.Context) error {
	worker.logger.Info().Msg("Creating new notary submission")
	submissionResp, err := api.StartNewSubmission(ctx, worker.auth.AuthenticateApiRequests, &api.SubmissionRequest{
		Name: filepath.Base(worker.getTargetFile()),
		Hash: worker.uploadFileHash,
	})

	if err != nil {
		worker.logger.Error().Err(err).Msg("Failed to create new submission on Notary API")
		return fmt.Errorf("create new notary submission entry: %w", err)
	}

	worker.submissionId = submissionResp.Id
	worker.logger = worker.logger.With().Str("submissionId", submissionResp.Id).Logger()

	worker.logger.Info().Msg("Uploading file to notary S3 bucket")
	if err = worker.uploadFile(ctx, submissionResp); err != nil {
		worker.logger.Error().Err(err).Msg("Failed to upload file to S3 bucket")
		return fmt.Errorf("upload file to S3 notary bucket: %w", err)
	}

	worker.logger.Info().Msg("Successfully uploaded, waiting for submission to complete")
	finalState := worker.waitForCompletion(ctx)
	if err = ctx.Err(); err != nil {
		// NOTE: Context was cancelled, not need to
		//       attempt further actions as they will
		//       fail immediately
		return err
	}

	worker.logger.Info().Msg("Retrieving notarization log for this submission")
	if err = worker.downloadNotarizationLog(ctx); err != nil {
		worker.logger.Error().Err(err).Msg("Failed to retrieve notarization log")
		return fmt.Errorf("retrieve notarization log: %w", err)
	}

	if finalState == api.SubmissionStatusStateAccepted {
		if err = worker.stapleTicket(ctx); err != nil {
			worker.logger.Error().Err(err).Msg("Failed to staple notarization ticket to package")
		}
	}

	if issueCount := len(worker.notarizationLog.Issues); issueCount != 0 {
		worker.logger.Warn().Int("numIssues", issueCount).Msg("This package has one or more issues detected by the Notary")
	}

	if finalState == api.SubmissionStatusStateAccepted {
		worker.logger.Info().Msg("Notarization was completed successfully")
	} else {
		worker.logger.Warn().Msg("Notarization was unsuccessful")
	}

	return nil
}

func (worker *Worker) uploadFile(ctx context.Context, subResp *api.SubmissionResponse) error {
	srcFile, err := os.OpenFile(worker.getTargetFile(), os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file for upload: %w", err)
	}
	defer srcFile.Close()

	client := s3.New(s3.Options{
		Credentials:   subResp,
		Region:        "us-west-2",
		UseAccelerate: true,
	})

	multiPart, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(subResp.Bucket),
		Key:    aws.String(subResp.Object),
	})
	if err != nil {
		return fmt.Errorf("start S3 multipart upload: %w", err)
	}

	buffer := make([]byte, 5*1024*1024 /* 5Mib */)
	var uploadLog types.CompletedMultipartUpload
	for part := int32(1); err == nil; part++ {
		var n int
		n, err = srcFile.Read(buffer)
		if err != nil {
			break
		}

		worker.logger.Debug().Int32("part", part).Int("size", n).Msg("Uploading part of file to S3")
		var resp *s3.UploadPartOutput
		resp, err = client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:        multiPart.Bucket,
			Key:           multiPart.Key,
			PartNumber:    aws.Int32(part),
			UploadId:      multiPart.UploadId,
			Body:          bytes.NewReader(buffer[:n]),
			ContentLength: aws.Int64(int64(n)),
		})

		if err == nil {
			uploadLog.Parts = append(uploadLog.Parts, types.CompletedPart{
				ChecksumCRC64NVME: resp.ChecksumCRC64NVME,
				ETag:              resp.ETag,
				PartNumber:        aws.Int32(part),
			})
		}
	}

	if !errors.Is(err, io.EOF) && err != nil {
		_, _ = client.AbortMultipartUpload(context.Background(), &s3.AbortMultipartUploadInput{
			Bucket:   multiPart.Bucket,
			Key:      multiPart.Key,
			UploadId: multiPart.UploadId,
		})

		return fmt.Errorf("upload file to notary S3 bucket: %w", err)
	}

	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          multiPart.Bucket,
		Key:             multiPart.Key,
		UploadId:        multiPart.UploadId,
		MultipartUpload: &uploadLog,
	})
	if err != nil {
		return fmt.Errorf("finalise S3 upload: %w", err)
	}

	return nil
}

func (worker *Worker) waitForCompletion(ctx context.Context) api.SubmissionStatusState {
	ticker := time.NewTicker(submissionStatusPollInterval)
	defer ticker.Stop()

	for {
		status, err := api.GetSubmissionStatus(ctx, worker.auth.AuthenticateApiRequests, worker.submissionId)
		if err != nil {
			worker.logger.Error().Err(err).Msg("Notary API returned an error, submission failed")
			return api.SubmissionStatusStateInvalid
		}

		state := status.Status
		switch state {
		case api.SubmissionStatusStateInProgress:
			worker.logger.Debug().Msg("Submission still in progress")

		case api.SubmissionStatusStateInvalid, api.SubmissionStatusStateRejected:
			worker.logger.Error().Str("state", state.String()).Msg("Submission failed")
			return state

		case api.SubmissionStatusStateAccepted:
			worker.logger.Info().Msg("Submission was successful")
			return state
		}

		// Apply jitter to requests to spread out
		// load produced by checking the status of
		// multiple submissions at the same time
		jitter := time.Duration(rand.Intn(submissionStatusPollJitter)) * time.Second
		ticker.Reset(submissionStatusPollInterval + jitter)

		select {
		case <-ctx.Done():
			return api.SubmissionStatusStateInvalid

		case <-ticker.C:
		}
	}
}

func (worker *Worker) downloadNotarizationLog(ctx context.Context) error {
	urlResp, err := api.GetSubmissionLogURL(ctx, worker.auth.AuthenticateApiRequests, worker.submissionId)
	if err != nil {
		return fmt.Errorf("get submission log URL: %w", err)
	}

	worker.notarizationLog, err = api.DownloadNotaryLog(ctx, urlResp.DeveloperLogUrl)
	if err != nil {
		return fmt.Errorf("download log file: %w", err)
	}

	return nil
}
