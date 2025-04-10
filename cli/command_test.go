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
	"errors"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	cmd := &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		SuggestDistance:   2,
		Usage:             "aliyun [subcmd]",
		Short: i18n.T(
			"cmd Short",
			"",
		),
	}
	subcmd := &Command{
		Name:            "oss",
		SuggestDistance: 2,
		Usage:           "oss flag",
		Short: i18n.T(
			"oss Short",
			"",
		),
	}

	//AddSubCommand
	assert.Len(t, cmd.subCommands, 0)
	cmd.AddSubCommand(subcmd)
	assert.Len(t, cmd.subCommands, 1)

	//Flags
	fs := cmd.Flags()
	exfs := NewFlagSet()
	assert.Equal(t, exfs, fs)

	//Execute TODO

	//GetSubCommand
	testSubcmd := cmd.GetSubCommand("oo")
	assert.Nil(t, testSubcmd)
	testSubcmd = cmd.GetSubCommand("oss")
	assert.Equal(t, subcmd, testSubcmd)

	//GetSuggestions
	suggestions := cmd.GetSuggestions("A")
	assert.Nil(t, suggestions)
	assert.Len(t, cmd.GetSuggestions("o"), 1)
	assert.Equal(t, "oss", cmd.GetSuggestions("o")[0])

	//GetSuggestDistance
	cmd.SuggestDistance = -1
	assert.Equal(t, 0, cmd.GetSuggestDistance())
	cmd.SuggestDistance = 0
	assert.Equal(t, 2, cmd.GetSuggestDistance())
	cmd.SuggestDistance = 1
	assert.Equal(t, 1, cmd.GetSuggestDistance())

	//GetUsageWithParent
	usag := subcmd.GetUsageWithParent()
	assert.Equal(t, "aliyun oss flag", usag)

	//ExecuteComplete
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	ctx.completion = new(Completion)
	ctx.completion.Current = "--f"
	ctx.flags.flags = append(ctx.flags.flags, []*Flag{
		{Name: "ff1", Hidden: true},
		{Name: "ff2"},
	}...)
	cmd.ExecuteComplete(ctx, []string{})
	assert.Equal(t, "--ff2\n", w.String())

	ctx.completion.Current = "o"
	subcmd2 := &Command{
		Name:            "ecs",
		SuggestDistance: 2,
		Usage:           "ecs flag",
		Hidden:          true,
		Short: i18n.T(
			"ecs Short",
			"",
		),
	}
	subcmd3 := &Command{
		Name:            "ess",
		SuggestDistance: 2,
		Usage:           "ess flag",
		Short: i18n.T(
			"ess Short",
			"",
		),
	}
	cmd.AddSubCommand(subcmd2)
	cmd.AddSubCommand(subcmd3)
	w.Reset()
	cmd.ExecuteComplete(ctx, []string{})
	assert.Equal(t, "oss\n", w.String())

	//executeInner TODO

	err := cmd.executeInner(ctx, []string{"a", "b"})
	assert.Equal(t, &InvalidCommandError{"a", ctx}, err)
	ctx.flags.flags = append(ctx.flags.flags, &Flag{Name: "help", assigned: true})
	err = cmd.executeInner(ctx, []string{"help", "oss"})
	assert.Nil(t, err)

	cmd.subCommands[0].Hidden = false
	ctx.completion = nil
	err = cmd.executeInner(ctx, []string{"ess"})
	assert.Nil(t, err)

	ctx.help = false
	err = cmd.executeInner(ctx, []string{"ess"})
	assert.Nil(t, err)

	//processError can not test

	//executeHelp

}

func newAliyunCmd() *Command {
	return &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		SuggestDistance:   2,
		Usage:             "aliyun [subcmd]",
		Short: i18n.T(
			"cmd Short",
			"",
		),
	}
}
func newTestCmd() *Command {
	return &Command{
		Name:            "test",
		SuggestDistance: 2,
		Usage:           "test flag",
		Short: i18n.T(
			"test Short",
			"",
		),
	}
}
func TestAddSubCommand(t *testing.T) {
	cmd := newAliyunCmd()
	subCmd := newTestCmd()
	assert.Nil(t, subCmd.parent)
	cmd.AddSubCommand(subCmd)
	assert.Equal(t, cmd, subCmd.parent)
}

// Flags
func TestCmdFlags(t *testing.T) {
	cmd := newAliyunCmd()
	fs := cmd.Flags()
	exfs := NewFlagSet()
	assert.Equal(t, exfs, fs)
}

func TestGetSubCommand(t *testing.T) {
	cmd := newAliyunCmd()
	subCmd := newTestCmd()
	actual := cmd.GetSubCommand("test")
	assert.Nil(t, actual)
	cmd.AddSubCommand(subCmd)
	actual = cmd.GetSubCommand("test")
	assert.Equal(t, subCmd, actual)
}

func TestExecute(t *testing.T) {
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := NewCommandContext(buf, buf2)

	cmd := newAliyunCmd()
	subCmd := newTestCmd()
	ctx.command = subCmd
	// cmd.AddSubCommand(subCmd)
	ctx.completion = &Completion{
		Args: []string{"test"},
	}
	DisableExitCode()
	defer EnableExitCode()
	cmd.Execute(ctx, []string{})
	assert.Equal(t, "\x1b[1;31mERROR: 'test' is not a vaild command\n\x1b[0m", buf2.String())
}

func TestProcessError(t *testing.T) {
	DisableExitCode()
	defer EnableExitCode()
	cmd := newAliyunCmd()
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := NewCommandContext(buf, buf2)
	e := errors.New("test error tip")
	err := NewErrorWithTip(e, "")
	cmd.processError(ctx, err)
	assert.Equal(t, "\x1b[1;31mERROR: test error tip\n\x1b[0m\x1b[1;33m\n\n\x1b[0m", buf2.String())
}

func TestExecuteHelp(t *testing.T) {
	cmd := newAliyunCmd()
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := NewCommandContext(buf, buf2)
	cmd.Help = func(ctx *Context, args []string) error {
		fmt.Fprint(ctx.Stdout(), "test execute help")
		return nil
	}
	cmd.executeHelp(ctx, nil)
	assert.Equal(t, "test execute help", buf.String())

	buf.Reset()
	buf2.Reset()
	DisableExitCode()
	defer EnableExitCode()
	cmd.Help = func(ctx *Context, args []string) error {
		return errors.New("test help error")
	}
	cmd.executeHelp(ctx, nil)
	assert.Equal(t, "\x1b[1;31mERROR: test help error\n\x1b[0m", buf2.String())
}
