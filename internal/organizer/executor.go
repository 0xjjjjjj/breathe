package organizer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/0xjjjjjj/breathe/internal/history"
)

type Executor struct {
	db     *history.DB
	dryRun bool
}

func NewExecutor(db *history.DB, dryRun bool) *Executor {
	return &Executor{db: db, dryRun: dryRun}
}

func (e *Executor) Execute(plan *Plan) error {
	for _, fp := range plan.Files {
		if err := e.moveFile(fp); err != nil {
			return fmt.Errorf("failed to move %s: %w", fp.Source, err)
		}
	}
	return nil
}

func (e *Executor) moveFile(fp FilePlan) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(fp.Dest)
	if !e.dryRun {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
	}

	// Handle duplicates
	dest := fp.Dest
	if _, err := os.Stat(dest); err == nil {
		// File exists, add timestamp
		ext := filepath.Ext(dest)
		base := dest[:len(dest)-len(ext)]
		dest = fmt.Sprintf("%s_%s%s", base, time.Now().Format("2006-01-02"), ext)
	}

	if e.dryRun {
		fmt.Printf("[DRY RUN] %s -> %s\n", fp.Source, dest)
		return nil
	}

	// Calculate hash before move
	hash, _ := fileHash(fp.Source)

	// Move file
	if err := os.Rename(fp.Source, dest); err != nil {
		return err
	}

	// Record in history
	if e.db != nil {
		e.db.Record(history.Operation{
			Type:       history.OpMove,
			SourcePath: fp.Source,
			DestPath:   dest,
			FileSize:   fp.Size,
			FileHash:   hash,
			Reversible: true,
			Metadata:   map[string]string{"original_name": filepath.Base(fp.Source)},
		})
	}

	return nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
