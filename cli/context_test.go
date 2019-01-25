/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/i18n"
)

var cmd = &Command{
	Name:  "oss",
	flags: NewFlagSet(),
}

func TestHelpFlag(t *testing.T) {
	fs := NewFlagSet()
	fs.Add(NewHelpFlag())
	f := HelpFlag(fs)
	assert.Equal(t, &Flag{Name: "help", Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone, formation: "--help"}, f)
}

func TestNewHelpFlag(t *testing.T) {
	f := NewHelpFlag()
	assert.Equal(t, &Flag{Name: "help", Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone}, f)
}

func TestContext_SetUnknownFlags(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	ctx.SetUnknownFlags(NewFlagSet())
	assert.NotNil(t, ctx.unknownFlags)
}

func TestNewCommandContext(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	assert.Equal(t, &Context{
		flags:        NewFlagSet(),
		unknownFlags: nil,
		writer:       w,
		help:         false,
		command:      nil,
		completion:   nil,
	}, ctx)
}

func TestCtx(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	assert.False(t, ctx.IsHelp())
	assert.Nil(t, ctx.Command())
	assert.Nil(t, ctx.Completion())
	assert.Equal(t, ctx.flags, ctx.Flags())
	assert.Equal(t, w, ctx.Writer())
	assert.Nil(t, ctx.UnknownFlags())
	ctx.SetCompletion(&Completion{Current: "M", Args: []string{"GOOD", "BAD"}, line: "MrX", point: 2})
	assert.Equal(t, &Completion{Current: "M", Args: []string{"GOOD", "BAD"}, line: "MrX", point: 2}, ctx.Completion())

	//EnterCommand
	ctx.EnterCommand(cmd)
	assert.Nil(t, ctx.unknownFlags)
	ctx.EnterCommand(cmd)
	assert.Equal(t, &Flag{Name: "help", Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone, formation: "--help"}, ctx.flags.Get("help"))
}

func TestCheckFlags(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	ctx.flags.AddByName("MrX")
	assert.Nil(t, ctx.CheckFlags())
	ctx.flags.flags[0].Required = true
	assert.EqualError(t, ctx.CheckFlags(), "missing flag --MrX")
	ctx.flags.flags[0].assigned = true
	ctx.flags.flags[0].Fields = []Field{Field{Key: "m", Required: true}}
	assert.EqualError(t, ctx.CheckFlags(), "bad flag format --MrX with field m= required")
	ctx.flags.flags[0].Fields[0].Required = false
	ctx.flags.flags[0].ExcludeWith = []string{"MrX"}
	ctx.flags.flags[0].value = "M"
	assert.EqualError(t, ctx.CheckFlags(), "flag --MrX is exclusive with --MrX")
}

func TestDetectFlag(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	ctx.flags.AddByName("MrX")
	f, err := ctx.detectFlag("mrx")
	assert.Nil(t, f)
	assert.NotNil(t, err)
	f, err = ctx.detectFlag("MrX")
	assert.NotNil(t, f)
	assert.Nil(t, err)
	ctx.unknownFlags = NewFlagSet()
	f, err = ctx.detectFlag("mrx")
	assert.NotNil(t, f)
	assert.Nil(t, err)
}

func TestDetectFlagByShorthand(t *testing.T) {
	w := new(bytes.Buffer)
	ctx := NewCommandContext(w)
	ctx.flags.Add(&Flag{Name: "profile", Shorthand: 'p'})
	f, err := ctx.detectFlagByShorthand('p')
	assert.Equal(t, &Flag{Name: "profile", Shorthand: 'p', formation: "-p"}, f)
	assert.Nil(t, err)
	f, err = ctx.detectFlagByShorthand('c')
	assert.Nil(t, f)
	assert.EqualError(t, err, "unknown flag -c")
}
