// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	err := NewInvalidCommandError("MrX", ctx)
	e, ok := err.(*InvalidCommandError)
	assert.True(t, ok)
	assert.Equal(t, "'MrX' is not a valid command", e.Error())
	e.ctx.EnterCommand(&Command{Name: "oss", flags: NewFlagSet()})
	assert.Nil(t, e.GetSuggestions())
}

func TestInvalidFlagError(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	err := NewInvalidFlagError("MrX", ctx)
	e, ok := err.(*InvalidFlagError)
	assert.True(t, ok)
	assert.Equal(t, "invalid flag --MrX", e.Error())
	assert.Nil(t, e.GetSuggestions())

	e.ctx.EnterCommand(&Command{Name: "oss", flags: NewFlagSet()})
	// help flag is always added; no close match for MrX → list available flags
	assert.Contains(t, e.Error(), "invalid flag --MrX; available flags:")
	assert.Contains(t, e.Error(), "--help")
	assert.Nil(t, e.GetSuggestions())
}

func TestInvalidFlagError_DidYouMean(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	cmd := &Command{Name: "uninstall", flags: NewFlagSet()}
	cmd.Flags().Add(&Flag{Name: "name", AssignedMode: AssignedOnce})
	ctx.EnterCommand(cmd)

	err := NewInvalidFlagError("names", ctx)
	e, ok := err.(*InvalidFlagError)
	assert.True(t, ok)
	// "names" vs "--name" should suggest via UnifyApply
	assert.Equal(t, "invalid flag --names, did you mean --name?", e.Error())
	assert.Nil(t, e.GetSuggestions())
}

func TestInvalidFlagError_DidYouMeanOr(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	cmd := &Command{Name: "cmd", flags: NewFlagSet()}
	cmd.Flags().Add(&Flag{Name: "foo", AssignedMode: AssignedOnce})
	cmd.Flags().Add(&Flag{Name: "fox", AssignedMode: AssignedOnce})
	ctx.EnterCommand(cmd)

	err := NewInvalidFlagError("fo", ctx)
	e := err.(*InvalidFlagError)
	msg := e.Error()
	assert.Contains(t, msg, "invalid flag --fo, did you mean")
	assert.Contains(t, msg, "--foo")
	assert.Contains(t, msg, "--fox")
	assert.Contains(t, msg, " or ")
}

func TestInvalidFlagError_AlreadyPrefixed(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	err := NewInvalidFlagError("--MrX", ctx)
	e := err.(*InvalidFlagError)
	assert.Equal(t, "invalid flag --MrX", e.Error())
}

func TestInvalidFlagError_NilContext(t *testing.T) {
	e := &InvalidFlagError{Flag: "x", ctx: nil}
	assert.Equal(t, "invalid flag --x", e.Error())
	assert.Nil(t, e.closeSuggestions())
	assert.Nil(t, e.availableFlags())
}
