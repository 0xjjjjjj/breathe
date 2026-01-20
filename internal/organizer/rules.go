package organizer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/0xjjjjjj/breathe/internal/config"
)

type RuleMatcher struct {
	rules []config.OrganizeRule
}

func NewRuleMatcher(rules []config.OrganizeRule) *RuleMatcher {
	return &RuleMatcher{rules: rules}
}

func (m *RuleMatcher) Match(filename string) string {
	for _, rule := range m.rules {
		patterns := expandBraces(rule.Match)
		for _, p := range patterns {
			matched, err := doublestar.PathMatch(p, filename)
			if err == nil && matched {
				return rule.Dest
			}
		}
	}
	return ""
}

// expandBraces expands {a,b,c} patterns into multiple patterns
func expandBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	if start == -1 {
		return []string{pattern}
	}

	end := strings.Index(pattern, "}")
	if end == -1 {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	options := strings.Split(pattern[start+1:end], ",")

	var result []string
	for _, opt := range options {
		expanded := expandBraces(prefix + opt + suffix)
		result = append(result, expanded...)
	}
	return result
}

type FilePlan struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
	Size   int64  `json:"size"`
}

type Plan struct {
	Files  []FilePlan            `json:"files"`
	ByDest map[string][]FilePlan `json:"by_dest"`
}

func (m *RuleMatcher) CreatePlan(sourcePath string) (*Plan, error) {
	plan := &Plan{
		ByDest: make(map[string][]FilePlan),
	}

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		dest := m.Match(e.Name())
		if dest == "" {
			continue
		}

		// Expand ~ to home directory
		if strings.HasPrefix(dest, "~") {
			home, _ := os.UserHomeDir()
			dest = filepath.Join(home, dest[1:])
		}

		fp := FilePlan{
			Source: filepath.Join(sourcePath, e.Name()),
			Dest:   filepath.Join(dest, e.Name()),
			Size:   info.Size(),
		}

		plan.Files = append(plan.Files, fp)
		plan.ByDest[dest] = append(plan.ByDest[dest], fp)
	}

	return plan, nil
}
