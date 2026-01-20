package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Entry struct {
	Path  string
	Name  string
	Size  int64
	IsDir bool
}

type ScanResult struct {
	Entry Entry
	Err   error
}

func Scan(root string, results chan<- ScanResult) {
	defer close(results)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // Limit concurrent goroutines

	var walk func(path string)
	walk = func(path string) {
		defer wg.Done()

		entries, err := os.ReadDir(path)
		if err != nil {
			results <- ScanResult{Err: err}
			return
		}

		for _, e := range entries {
			fullPath := filepath.Join(path, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}

			entry := Entry{
				Path:  fullPath,
				Name:  e.Name(),
				IsDir: e.IsDir(),
			}

			if !e.IsDir() {
				entry.Size = info.Size()
			}

			results <- ScanResult{Entry: entry}

			if e.IsDir() && !isSymlink(info) {
				wg.Add(1)
				sem <- struct{}{}
				go func(p string) {
					defer func() { <-sem }()
					walk(p)
				}(fullPath)
			}
		}
	}

	wg.Add(1)
	walk(root)
	wg.Wait()
}

func isSymlink(info fs.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}
