package scanner

import (
	"encoding/json"
	"io"
)

type JSONOutput struct {
	Path       string      `json:"path"`
	TotalSize  int64       `json:"total_size"`
	TotalFiles int         `json:"total_files"`
	Children   []JSONEntry `json:"children"`
	Junk       []JunkGroup `json:"junk,omitempty"`
}

type JSONEntry struct {
	Path     string      `json:"path"`
	Name     string      `json:"name"`
	Size     int64       `json:"size"`
	IsDir    bool        `json:"is_dir"`
	Children []JSONEntry `json:"children,omitempty"`
}

func (t *Tree) ToJSON(w io.Writer, matcher *Matcher, maxDepth int) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	output := JSONOutput{
		Path:       t.root.Path,
		TotalSize:  t.root.Size,
		TotalFiles: t.FileCount(),
		Children:   t.childrenToJSON(t.root.Path, 0, maxDepth),
	}

	if matcher != nil {
		output.Junk = matcher.GroupJunk(t)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func (t *Tree) childrenToJSON(path string, depth, maxDepth int) []JSONEntry {
	if maxDepth > 0 && depth >= maxDepth {
		return nil
	}

	node, ok := t.nodes[path]
	if !ok {
		return nil
	}

	entries := make([]JSONEntry, 0, len(node.children))

	for _, child := range node.children {
		entry := JSONEntry{
			Path:  child.Path,
			Name:  child.Name,
			Size:  child.Size,
			IsDir: child.IsDir,
		}
		if child.IsDir {
			entry.Children = t.childrenToJSON(child.Path, depth+1, maxDepth)
		}
		entries = append(entries, entry)
	}

	return entries
}
