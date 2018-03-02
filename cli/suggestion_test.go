package cli

import (
	"testing"
)

func TestSuggestion(t *testing.T) {
	s := NewSuggester("aaa", 2)

	s.Apply("aab")
	s.Apply("aa2")
	s.Apply("aa2")
	s.Apply("aa2b")
	s.Apply("baa2b")

	for _, r := range s.results {
		t.Logf("%s\n", r)
	}
}
