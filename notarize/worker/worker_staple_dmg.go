package worker

import (
	"fmt"
	"os"

	"github.com/KatelynHaworth/notarization-helper/v2/codesign"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs"
	"github.com/KatelynHaworth/notarization-helper/v2/codesign/blobs/super_blob"
	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
)

func (worker *Worker) stapleToDmg(ticket api.SignedTicket) error {
	dmgStat, err := os.Stat(worker.target.File)
	if err != nil {
		return fmt.Errorf("get dmg file stats: %w", err)
	}

	dmgFile, err := os.OpenFile(worker.target.File, os.O_APPEND|os.O_RDWR, dmgStat.Mode())
	if err != nil {
		return fmt.Errorf("open dmg file: %w", err)
	}
	defer dmgFile.Close()

	super, err := codesign.ReadFromDMG[*super_blob.SuperBlob](dmgFile)
	if err != nil {
		return fmt.Errorf("read codesign from dmg: %w", err)
	}

	ticketBlob, _ := blobs.NewGeneric(codesign.MagicBlobWrapper, ticket.Value)
	if err = super.AddBlob(super_blob.SlotTicket, ticketBlob); err != nil {
		return fmt.Errorf("add ticket to super blob: %w", err)
	}

	if err = codesign.WriteToDMG(super, dmgFile); err != nil {
		return fmt.Errorf("write codesign to dmg: %w", err)
	}

	return nil
}
