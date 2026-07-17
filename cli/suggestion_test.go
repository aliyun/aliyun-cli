// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	PrintSuggestions(ctx, "en", []string{"hello", "nihao"})
	assert.Equal(t, "\x1b[1;33m\nDid you mean:\n\x1b[0m\x1b[1;33m  hello\n\x1b[0m\x1b[1;33m  nihao\n\x1b[0m", w.String())

}

func TestStripNonAlphanumeric(t *testing.T) {
	assert.Equal(t, "abc123", stripNonAlphanumeric("abc123"))
	assert.Equal(t, "describeregions", stripNonAlphanumeric("describe-regions"))
	assert.Equal(t, "listinstances", stripNonAlphanumeric("list_instances"))
	assert.Equal(t, "helloworld", stripNonAlphanumeric("hello world!@#"))
	assert.Equal(t, "", stripNonAlphanumeric("---"))
	assert.Equal(t, "", stripNonAlphanumeric(""))
}

func TestUnifyApply(t *testing.T) {
	t.Run("Match ignoring hyphens and case", func(t *testing.T) {
		s := NewSuggester("describe-regions", 2)
		s.UnifyApply("DescribeRegions")
		results := s.GetResults()
		assert.Contains(t, results, "DescribeRegions")
	})

	t.Run("Preserves original string in results", func(t *testing.T) {
		s := NewSuggester("list-instances", 2)
		s.UnifyApply("ListInstances")
		results := s.GetResults()
		assert.Equal(t, []string{"ListInstances"}, results)
	})

	t.Run("No match when too far", func(t *testing.T) {
		s := NewSuggester("abc", 2)
		s.UnifyApply("xyz-totally-different")
		results := s.GetResults()
		assert.Empty(t, results)
	})

	t.Run("Closer match replaces farther", func(t *testing.T) {
		s := NewSuggester("describe-region", 3)
		s.UnifyApply("DescribeInstances")
		farResults := s.GetResults()

		s.UnifyApply("DescribeRegions")
		closerResults := s.GetResults()

		if len(farResults) > 0 {
			assert.LessOrEqual(t, len(closerResults), len(farResults)+1)
		}
		assert.Contains(t, closerResults, "DescribeRegions")
	})

	t.Run("Multiple matches at same distance", func(t *testing.T) {
		s := NewSuggester("ab", 2)
		s.UnifyApply("abc")
		s.UnifyApply("abd")
		results := s.GetResults()
		assert.Contains(t, results, "abc")
		assert.Contains(t, results, "abd")
	})
}
