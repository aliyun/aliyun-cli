/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
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
