package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDB_RecordAndQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	op := Operation{
		Type:       OpMove,
		SourcePath: "/downloads/report.pdf",
		DestPath:   "/documents/report.pdf",
		FileSize:   1024,
		Reversible: true,
	}

	id, err := db.Record(op)
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero id")
	}

	results, err := db.Search("report")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestDB_SearchByDate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	db.Record(Operation{
		Type:       OpMove,
		SourcePath: "/old/file.txt",
		DestPath:   "/new/file.txt",
	})

	since := time.Now().Add(-1 * time.Hour)
	results, err := db.Since(since)
	if err != nil {
		t.Fatalf("Since() error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestDB_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	db.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}
