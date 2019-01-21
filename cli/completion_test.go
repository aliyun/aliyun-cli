/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCompletion(t *testing.T) {
	cp := ParseCompletion("", "")
	assert.Nil(t, cp)

	cp = ParseCompletion("", "s")
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
	cp = ParseCompletion("name", "5")

}

func TestParseLineForCompletion(t *testing.T) {
	cl := parseLineForCompletion(`c\d'a'"hao" c\alok`, 18)
	assert.Subset(t, cl, []string{`c\d'a'"hao"`, `c\alok`})
	assert.Len(t, cl, 2)

	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	cl = parseLineForCompletion(`c\d'a'"hao" c\alok`, 20)
}
func TestCompletionGet(t *testing.T) {
	cp := ParseCompletion("name Mrx aa", "13")
	assert.Equal(t, "aa", cp.GetCurrent())
	assert.Equal(t, []string{"Mrx"}, cp.GetArgs())
}
