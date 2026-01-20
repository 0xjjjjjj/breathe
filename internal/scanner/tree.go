package scanner

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Node struct {
	Path     string
	Name     string
	Size     int64
	IsDir    bool
	children map[string]*Node
}

type Tree struct {
	root  *Node
	nodes map[string]*Node
	mu    sync.RWMutex
}

func NewTree(rootPath string) *Tree {
	root := &Node{
		Path:     rootPath,
		Name:     filepath.Base(rootPath),
		IsDir:    true,
		children: make(map[string]*Node),
	}
	return &Tree{
		root:  root,
		nodes: map[string]*Node{rootPath: root},
	}
}

func (t *Tree) AddEntry(e Entry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Ensure parent directories exist
	t.ensureParents(e.Path)

	node, exists := t.nodes[e.Path]
	if !exists {
		node = &Node{
			Path:     e.Path,
			Name:     e.Name,
			IsDir:    e.IsDir,
			children: make(map[string]*Node),
		}
		t.nodes[e.Path] = node

		// Add to parent
		parentPath := filepath.Dir(e.Path)
		if parent, ok := t.nodes[parentPath]; ok {
			parent.children[e.Name] = node
		}
	}

	if !e.IsDir {
		node.Size = e.Size
		t.propagateSize(e.Path, e.Size)
	}
}

func (t *Tree) ensureParents(path string) {
	parts := strings.Split(strings.TrimPrefix(path, t.root.Path), string(filepath.Separator))
	current := t.root.Path

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		current = filepath.Join(current, parts[i])
		if _, exists := t.nodes[current]; !exists {
			node := &Node{
				Path:     current,
				Name:     parts[i],
				IsDir:    true,
				children: make(map[string]*Node),
			}
			t.nodes[current] = node
			parentPath := filepath.Dir(current)
			if parent, ok := t.nodes[parentPath]; ok {
				parent.children[parts[i]] = node
			}
		}
	}
}

func (t *Tree) propagateSize(path string, size int64) {
	for {
		parentPath := filepath.Dir(path)
		if parentPath == path || parentPath == "." {
			break
		}
		if parent, ok := t.nodes[parentPath]; ok {
			parent.Size += size
		}
		path = parentPath
	}
}

func (t *Tree) Root() *Node {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.root
}

func (t *Tree) Children(path string) []*Node {
	t.mu.RLock()
	defer t.mu.RUnlock()

	node, ok := t.nodes[path]
	if !ok {
		return nil
	}

	children := make([]*Node, 0, len(node.children))
	for _, child := range node.children {
		children = append(children, child)
	}

	// Sort by size descending
	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	return children
}

func (t *Tree) Get(path string) *Node {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.nodes[path]
}

func (t *Tree) FileCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	count := 0
	for _, n := range t.nodes {
		if !n.IsDir {
			count++
		}
	}
	return count
}

// Add is a convenience method for tests - adds a node with given path, isDir, and size
func (t *Tree) Add(path string, isDir bool, size int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Ensure parent directories exist
	t.ensureParents(path)

	node, exists := t.nodes[path]
	if !exists {
		node = &Node{
			Path:     path,
			Name:     filepath.Base(path),
			IsDir:    isDir,
			children: make(map[string]*Node),
		}
		t.nodes[path] = node

		// Add to parent
		parentPath := filepath.Dir(path)
		if parent, ok := t.nodes[parentPath]; ok {
			parent.children[filepath.Base(path)] = node
		}
	}

	// Set size directly (for testing purposes, allows setting dir sizes)
	node.Size = size
}
