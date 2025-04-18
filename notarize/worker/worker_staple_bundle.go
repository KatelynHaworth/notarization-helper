package worker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KatelynHaworth/notarization-helper/v2/notarize/api"
)

var (
	bundleExtensions = []string{
		".app",
		".kext",
	}
)

func (worker *Worker) stapleToBundle(ticket api.SignedTicket) error {
	if stat, err := os.Stat(filepath.Join(worker.target.File, "Contents")); os.IsNotExist(err) || !stat.IsDir() {
		return errors.New("invalid directory provided, no 'Contents' directory")
	}

	if err := os.WriteFile(filepath.Join(worker.target.File, "Contents", "CodeResources"), ticket.Value, 0644); err != nil {
		return fmt.Errorf("write ticket to 'CodeResources' file in directory: %w", err)
	}

	return nil
}
