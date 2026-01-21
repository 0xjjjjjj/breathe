package scanner

import (
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

			// Use e.Type() to check symlink - more reliable than info.Mode()
			if e.IsDir() && e.Type()&os.ModeSymlink == 0 {
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

