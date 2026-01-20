package organizer

import (
	"testing"

	"github.com/0xjjjjjj/breathe/internal/config"
)

func TestMatcher_MatchesPDF(t *testing.T) {
	rules := []config.OrganizeRule{
		{Match: "*.pdf", Dest: "~/Documents"},
		{Match: "*", Dest: "~/Unsorted"},
	}
	m := NewRuleMatcher(rules)

	dest := m.Match("report.pdf")
	if dest != "~/Documents" {
		t.Errorf("expected ~/Documents, got %s", dest)
	}
}

func TestMatcher_FallsBackToDefault(t *testing.T) {
	rules := []config.OrganizeRule{
		{Match: "*.pdf", Dest: "~/Documents"},
		{Match: "*", Dest: "~/Unsorted"},
	}
	m := NewRuleMatcher(rules)

	dest := m.Match("random.xyz")
	if dest != "~/Unsorted" {
		t.Errorf("expected ~/Unsorted, got %s", dest)
	}
}

func TestMatcher_BraceExpansion(t *testing.T) {
	rules := []config.OrganizeRule{
		{Match: "*.{jpg,png,gif}", Dest: "~/Pictures"},
	}
	m := NewRuleMatcher(rules)

	for _, ext := range []string{"photo.jpg", "image.png", "anim.gif"} {
		dest := m.Match(ext)
		if dest != "~/Pictures" {
			t.Errorf("expected ~/Pictures for %s, got %s", ext, dest)
		}
	}
}

func TestExpandBraces_Single(t *testing.T) {
	result := expandBraces("*.txt")
	if len(result) != 1 || result[0] != "*.txt" {
		t.Errorf("expected [*.txt], got %v", result)
	}
}

func TestExpandBraces_Multiple(t *testing.T) {
	result := expandBraces("*.{a,b,c}")
	expected := []string{"*.a", "*.b", "*.c"}
	if len(result) != 3 {
		t.Errorf("expected 3 patterns, got %d", len(result))
	}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("expected %s at index %d, got %s", exp, i, result[i])
		}
	}
}
