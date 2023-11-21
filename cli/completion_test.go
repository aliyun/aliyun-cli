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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCompletionForShell(t *testing.T) {
	cp := ParseCompletionForShell()
	assert.Nil(t, cp)
}

func TestParseCompletion(t *testing.T) {
	cp := ParseCompletion("", "")
	assert.Nil(t, cp)

	cp = ParseCompletion("", "s")
	assert.Nil(t, cp)

	cp = ParseCompletion("line", "invalid number")
	assert.Nil(t, cp)

	cp = ParseCompletion("cdn ", "5")
	assert.Equal(t, &Completion{Current: "", Args: []string{""}, line: "cdn ", point: 4}, cp)

	cp = ParseCompletion(" ", "5")
	assert.Equal(t, &Completion{Current: "", Args: []string{}, line: " ", point: 1}, cp)

	cp = ParseCompletion("name Mrx aa", "13")
	assert.Equal(t, &Completion{Current: "aa", Args: []string{"Mrx"}, line: "name Mrx aa", point: 11}, cp)

	defer func() {
		reerr := recover()
		err, ok := reerr.(error)
		assert.True(t, ok)
		assert.EqualError(t, err, "unexcepted args [name] for line 'name'")
	}()
	ParseCompletion("name", "5")
}

func TestParseLineForCompletion(t *testing.T) {
	cl := parseLineForCompletion(`c\d'a'"hao" c\alok`, 18)
	assert.Subset(t, cl, []string{`c\d'a'"hao"`, `c\alok`})
	assert.Len(t, cl, 2)

	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	parseLineForCompletion(`c\d'a'"hao" c\alok`, 20)
}
func TestCompletionGet(t *testing.T) {
	cp := ParseCompletion("name Mrx aa", "13")
	assert.Equal(t, "aa", cp.GetCurrent())
	assert.Equal(t, []string{"Mrx"}, cp.GetArgs())
}
