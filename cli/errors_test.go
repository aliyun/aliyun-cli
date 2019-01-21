/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorWithTip(t *testing.T) {
	err := NewErrorWithTip(errors.New("err test"), "%s-%d", "nicai", 1)
	e, ok := err.(*errorWithTip)
	assert.True(t, ok)
	assert.Equal(t, &errorWithTip{err: errors.New("err test"), tip: fmt.Sprintf("%s-%d", "nicai", 1)}, e)
	assert.Equal(t, e.Error(), "err test")
	assert.Equal(t, "nicai-1", e.GetTip("ch"))
}

func TestInvalidCommandError(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	err := NewInvalidCommandError("MrX", ctx)
	e, ok := err.(*InvalidCommandError)
	assert.True(t, ok)
	assert.Equal(t, "'MrX' is not a vaild command", e.Error())
	e.ctx.EnterCommand(&Command{Name: "oss", flags: NewFlagSet()})
	assert.Nil(t, e.GetSuggestions())
}

func TestInvalidFlagError(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	err := NewInvalidFlagError("MrX", ctx)
	e, ok := err.(*InvalidFlagError)
	assert.True(t, ok)
	assert.Equal(t, "invalid flag MrX", e.Error())

	e.ctx.EnterCommand(&Command{Name: "oss", flags: NewFlagSet()})
	assert.NotNil(t, e.GetSuggestions())
	assert.Len(t, e.GetSuggestions(), 0)
}
