package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
)

type staplerFunc func(ticket api.SignedTicket) error

func (worker *Worker) stapleTicket(ctx context.Context) error {
	if !worker.target.Staple {
		return nil
	}

	stapler := worker.getAppropriateStapler()
	if stapler == nil {
		worker.logger.Warn().Msg("This file type is not supported for stapling, continuing without stapling")
		return nil
	}

	worker.logger.Info().Msg("Stapling notarization ticket to package")
	ticketContent := worker.findTicketOfBestFit()
	if ticketContent == nil {
		return fmt.Errorf("ticket content of best fit not found in notarization log")
	}

	worker.logger.Debug().Str("recordName", ticketContent.RecordName()).Msg("Downloading ticket")
	tickets, err := api.GetTickets(ctx, []api.TicketRecord{{RecordName: ticketContent.RecordName()}})
	if err != nil {
		return fmt.Errorf("get notarization ticket: %w", err)
	}

	// The return type of notary_api.GetTickets
	// is an array to allow, in the future, for the
	// retrieval of all tickets needed across all
	// workers in a single API call.

	worker.logger.Debug().Str("recordName", ticketContent.RecordName()).Msg("Stapling ticket")
	if err = stapler(tickets[0].Fields.Ticket); err != nil {
		return fmt.Errorf("staple ticket: %w", err)
	}

	worker.logger.Debug().Str("record", ticketContent.RecordName()).Msg("Successfully stapled ticket")
	return nil
}

func (worker *Worker) getAppropriateStapler() staplerFunc {
	stat, _ := os.Stat(worker.target.File)
	ext := filepath.Ext(worker.target.File)

	switch {
	case ext == ".pkg":
		return worker.stapleToPkg

	case ext == ".dmg":
		return worker.stapleToDmg

	case slices.Contains(bundleExtensions, ext) && stat.IsDir():
		return worker.stapleToBundle

	default:
		return nil
	}
}

func (worker *Worker) findTicketOfBestFit() *api.NotarizationTicket {
	if worker.notarizationLog == nil {
		return nil
	}

	// This is admittedly a cheat way to find the
	// correct code directory hash needed to download
	// the notarization ticket but, it removes the need
	// to find the actual code directory and hash it.
	//
	// Maybe in a future version this can be replaced
	// with just calculating the CD hash directly, the
	// only blocker is proper handling of mach-o files.

	expectedPath := filepath.Base(worker.target.File)
	if len(worker.zipFile) > 0 {
		expectedPath = fmt.Sprintf("%s/%s", filepath.Base(worker.zipFile), expectedPath)
	}

	i := slices.IndexFunc(worker.notarizationLog.TicketContents, func(ticket api.NotarizationTicket) bool {
		return ticket.Path == expectedPath
	})

	if i < 0 {
		return nil
	}

	return &worker.notarizationLog.TicketContents[i]
}
