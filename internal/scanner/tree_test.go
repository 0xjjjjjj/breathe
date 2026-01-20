package scanner

import (
	"testing"
)

func TestTree_AddEntry(t *testing.T) {
	tree := NewTree("/root")

	tree.AddEntry(Entry{Path: "/root/a/file.txt", Name: "file.txt", Size: 100, IsDir: false})
	tree.AddEntry(Entry{Path: "/root/a", Name: "a", IsDir: true})
	tree.AddEntry(Entry{Path: "/root/b/c/file2.txt", Name: "file2.txt", Size: 200, IsDir: false})

	root := tree.Root()
	if root.Size != 300 {
		t.Errorf("expected root size 300, got %d", root.Size)
	}
}

func TestTree_Children(t *testing.T) {
	tree := NewTree("/root")

	tree.AddEntry(Entry{Path: "/root/a", Name: "a", IsDir: true})
	tree.AddEntry(Entry{Path: "/root/b", Name: "b", IsDir: true})
	tree.AddEntry(Entry{Path: "/root/file.txt", Name: "file.txt", Size: 50, IsDir: false})

	children := tree.Children("/root")
	if len(children) != 3 {
		t.Errorf("expected 3 children, got %d", len(children))
	}
}

func TestTree_SortedBySize(t *testing.T) {
	tree := NewTree("/root")

	tree.AddEntry(Entry{Path: "/root/small", Name: "small", IsDir: true})
	tree.AddEntry(Entry{Path: "/root/small/f.txt", Name: "f.txt", Size: 10, IsDir: false})
	tree.AddEntry(Entry{Path: "/root/large", Name: "large", IsDir: true})
	tree.AddEntry(Entry{Path: "/root/large/f.txt", Name: "f.txt", Size: 1000, IsDir: false})

	children := tree.Children("/root")
	if children[0].Name != "large" {
		t.Errorf("expected largest first, got %s", children[0].Name)
	}
}
