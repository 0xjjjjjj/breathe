package scanner

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/0xjjjjjj/breathe/internal/config"
)

type Matcher struct {
	patterns []config.JunkPattern
}

type Match struct {
	Name    string
	Pattern string
	Safe    bool
	Path    string
}

func NewMatcher(patterns []config.JunkPattern) *Matcher {
	return &Matcher{patterns: patterns}
}

func (m *Matcher) Match(path string) []Match {
	var matches []Match
	for _, p := range m.patterns {
		matched, err := doublestar.PathMatch(p.Pattern, path)
		if err != nil {
			continue
		}
		if matched {
			matches = append(matches, Match{
				Name:    p.Name,
				Pattern: p.Pattern,
				Safe:    p.Safe,
				Path:    path,
			})
		}
	}
	return matches
}

func (m *Matcher) FindJunk(tree *Tree) map[string][]Match {
	junk := make(map[string][]Match)

	var walk func(node *Node)
	walk = func(node *Node) {
		matches := m.Match(node.Path)
		if len(matches) > 0 {
			junk[node.Path] = matches
			return // Don't recurse into junk directories
		}
		for _, child := range tree.Children(node.Path) {
			walk(child)
		}
	}

	walk(tree.Root())
	return junk
}

type JunkGroup struct {
	Name    string
	Pattern string
	Safe    bool
	Paths   []string
	Total   int64
}

func (m *Matcher) GroupJunk(tree *Tree) []JunkGroup {
	junk := m.FindJunk(tree)

	groups := make(map[string]*JunkGroup)
	for path, matches := range junk {
		for _, match := range matches {
			if g, ok := groups[match.Name]; ok {
				g.Paths = append(g.Paths, path)
				if node := tree.Get(path); node != nil {
					g.Total += node.Size
				}
			} else {
				var size int64
				if node := tree.Get(path); node != nil {
					size = node.Size
				}
				groups[match.Name] = &JunkGroup{
					Name:    match.Name,
					Pattern: match.Pattern,
					Safe:    match.Safe,
					Paths:   []string{path},
					Total:   size,
				}
			}
		}
	}

	result := make([]JunkGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}
	return result
}
