package scanner

import (
	"testing"

	"github.com/0xjjjjjj/breathe/internal/config"
)

func TestMatcher_MatchesNodeModules(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
	}
	m := NewMatcher(patterns)

	matches := m.Match("/home/user/project/node_modules")
	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Name != "node_modules" {
		t.Errorf("expected 'node_modules', got %s", matches[0].Name)
	}
}

func TestMatcher_NoMatch(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
	}
	m := NewMatcher(patterns)

	matches := m.Match("/home/user/project/src")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestMatcher_MultiplePatternsMatch(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "dist", Pattern: "**/dist", Safe: true},
		{Name: "build output", Pattern: "**/dist/**", Safe: true},
	}
	m := NewMatcher(patterns)

	matches := m.Match("/project/dist")
	if len(matches) < 1 {
		t.Errorf("expected at least 1 match, got %d", len(matches))
	}
}

func TestMatcher_SafeFlag(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
		{Name: ".git", Pattern: "**/.git", Safe: false},
	}
	m := NewMatcher(patterns)

	// Safe pattern
	matches := m.Match("/project/node_modules")
	if len(matches) != 1 || !matches[0].Safe {
		t.Error("node_modules should be safe to delete")
	}

	// Unsafe pattern
	matches = m.Match("/project/.git")
	if len(matches) != 1 || matches[0].Safe {
		t.Error(".git should NOT be safe to delete")
	}
}

func TestMatcher_FindJunk(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
		{Name: "__pycache__", Pattern: "**/__pycache__", Safe: true},
	}
	m := NewMatcher(patterns)

	tree := NewTree("/project")
	tree.Add("/project/src", false, 0)
	tree.Add("/project/node_modules", true, 1000)
	tree.Add("/project/node_modules/lodash", true, 500)
	tree.Add("/project/src/__pycache__", true, 200)

	junk := m.FindJunk(tree)

	// Should find node_modules and __pycache__
	if len(junk) != 2 {
		t.Errorf("expected 2 junk entries, got %d", len(junk))
	}

	if _, ok := junk["/project/node_modules"]; !ok {
		t.Error("should have found node_modules")
	}
	if _, ok := junk["/project/src/__pycache__"]; !ok {
		t.Error("should have found __pycache__")
	}

	// Should NOT recurse into junk (lodash is inside node_modules)
	if _, ok := junk["/project/node_modules/lodash"]; ok {
		t.Error("should not recurse into junk directories")
	}
}

func TestMatcher_GroupJunk(t *testing.T) {
	patterns := []config.JunkPattern{
		{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
	}
	m := NewMatcher(patterns)

	tree := NewTree("/")
	tree.Add("/project1/node_modules", true, 1000)
	tree.Add("/project2/node_modules", true, 2000)

	groups := m.GroupJunk(tree)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]
	if g.Name != "node_modules" {
		t.Errorf("expected group name 'node_modules', got %s", g.Name)
	}
	if len(g.Paths) != 2 {
		t.Errorf("expected 2 paths in group, got %d", len(g.Paths))
	}
	if g.Total != 3000 {
		t.Errorf("expected total size 3000, got %d", g.Total)
	}
	if !g.Safe {
		t.Error("expected group to be safe")
	}
}

func TestMatcher_InvalidPattern(t *testing.T) {
	// doublestar should handle invalid patterns gracefully
	patterns := []config.JunkPattern{
		{Name: "invalid", Pattern: "[invalid", Safe: true},
	}
	m := NewMatcher(patterns)

	// Should not panic, just return no matches
	matches := m.Match("/some/path")
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for invalid pattern, got %d", len(matches))
	}
}
