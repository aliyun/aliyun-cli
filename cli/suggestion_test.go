/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSuggester(t *testing.T) {
	suggestion := NewSuggester("flags", 2)
	assert.Equal(t, &Suggester{suggestFor: "flags", distance: 2}, suggestion)
}

func TestApply(t *testing.T) {
	s := NewSuggester("aaa", 2)
	s.Apply("aab")
	s.Apply("aa2")
	s.Apply("aa2b")
	s.Apply("baa2b")

	result := s.GetResults()
	assert.Subset(t, result, []string{"aab", "aa2"})
	assert.Len(t, result, 2)
}

func TestPrintSuggestions(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	PrintSuggestions(ctx, "en", []string{"hello", "nihao"})
	assert.Equal(t, "\x1b[1;33m\nDid you mean:\n\x1b[0m\x1b[1;33m  hello\n\x1b[0m\x1b[1;33m  nihao\n\x1b[0m", w.String())

}
