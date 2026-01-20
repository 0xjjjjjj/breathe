package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScan_CountsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "a", "file2.txt"), []byte("world!"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "a", "b", "file3.txt"), []byte("test"), 0644)

	results := make(chan ScanResult, 100)
	go Scan(tmpDir, results)

	var totalSize int64
	var fileCount int
	for r := range results {
		if r.Err != nil {
			t.Fatalf("scan error: %v", r.Err)
		}
		if !r.Entry.IsDir {
			totalSize += r.Entry.Size
			fileCount++
		}
	}

	if fileCount != 3 {
		t.Errorf("expected 3 files, got %d", fileCount)
	}
	// "hello" (5) + "world!" (6) + "test" (4) = 15
	if totalSize != 15 {
		t.Errorf("expected size 15, got %d", totalSize)
	}
}

func TestScan_StreamsResults(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("x"), 0644)

	results := make(chan ScanResult, 10)
	go Scan(tmpDir, results)

	// Should receive at least one result
	r := <-results
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}

	// Drain remaining
	for range results {
	}
}
