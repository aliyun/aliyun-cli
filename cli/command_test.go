/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/i18n"
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
	ctx := NewCommandContext(w)
	ctx.completion = new(Completion)
	ctx.completion.Current = "--f"
	ctx.flags.flags = append(ctx.flags.flags, []*Flag{
		&Flag{Name: "ff1", Hidden: true},
		&Flag{Name: "ff2"},
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
