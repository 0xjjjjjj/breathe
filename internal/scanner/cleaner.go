package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xjjjjjj/breathe/internal/history"
)

// Paths that should never be deleted
var protectedPaths = []string{
	"/", "/usr", "/etc", "/var", "/tmp", "/opt",
	"/bin", "/sbin", "/lib", "/lib64",
	"/System", "/Library", "/Applications", // macOS
	"/Windows", "/Program Files",            // Windows
	"/home", "/root",                        // Linux
}

var ErrProtectedPath = errors.New("refusing to delete protected system path")
var ErrPathTraversal = errors.New("path contains directory traversal")

type Cleaner struct {
	db       *history.DB
	useTrash bool
}

func NewCleaner(db *history.DB, useTrash bool) *Cleaner {
	return &Cleaner{db: db, useTrash: useTrash}
}

// validatePath ensures the path is safe to delete
func validatePath(path string) error {
	// Must be absolute
	if !filepath.IsAbs(path) {
		return errors.New("path must be absolute")
	}

	// Clean the path and check for traversal attempts
	cleaned := filepath.Clean(path)
	if strings.Contains(path, "..") {
		return ErrPathTraversal
	}

	// Check against protected paths
	for _, protected := range protectedPaths {
		if cleaned == protected {
			return fmt.Errorf("%w: %s", ErrProtectedPath, path)
		}
	}

	// Don't allow deleting home directory itself
	home, _ := os.UserHomeDir()
	if cleaned == home {
		return fmt.Errorf("%w: home directory", ErrProtectedPath)
	}

	// Don't allow deleting first-level directories in root
	parts := strings.Split(cleaned, string(filepath.Separator))
	if len(parts) <= 2 && parts[0] == "" {
		return fmt.Errorf("%w: top-level directory", ErrProtectedPath)
	}

	return nil
}

func (c *Cleaner) Delete(path string) error {
	// Validate path before any operations
	if err := validatePath(path); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var size int64
	if info.IsDir() {
		size = c.dirSize(path)
	} else {
		size = info.Size()
	}

	opType := history.OpDelete
	var destPath string

	if c.useTrash {
		opType = history.OpTrash
		destPath, err = c.moveToTrash(path)
		if err != nil {
			return err
		}
	} else {
		if info.IsDir() {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
		if err != nil {
			return err
		}
	}

	if c.db != nil {
		c.db.Record(history.Operation{
			Type:       opType,
			SourcePath: path,
			DestPath:   destPath,
			FileSize:   size,
			Reversible: c.useTrash,
		})
	}

	return nil
}

func (c *Cleaner) moveToTrash(path string) (string, error) {
	home, _ := os.UserHomeDir()
	trashPath := filepath.Join(home, ".Trash", filepath.Base(path))

	// Handle duplicates in trash
	if _, err := os.Stat(trashPath); err == nil {
		trashPath = filepath.Join(home, ".Trash", fmt.Sprintf("%s_%d", filepath.Base(path), os.Getpid()))
	}

	return trashPath, os.Rename(path, trashPath)
}

func (c *Cleaner) dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
