/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
)

//
// default help flag
var HelpFlag = &Flag{
	Name:         "help",
	Usage:        i18n.T("print help", "打印帮助信息"),
	AssignedMode: AssignedNone,
}

//
// CLI Command Context
type Context struct {
	help         bool
	flags        *FlagSet
	unknownFlags *FlagSet
	command      *Command
	completion   *Completion
}

func NewCommandContext() *Context {
	return &Context{
		flags:        NewFlagSet(),
		unknownFlags: nil,
	}
}

func (ctx *Context) IsHelp() bool {
	return ctx.help
}

func (ctx *Context) Command() *Command {
	return ctx.command
}

func (ctx *Context) Completion() *Completion {
	return ctx.completion
}

func (ctx *Context) Flags() *FlagSet {
	return ctx.flags
}

func (ctx *Context) UnknownFlags() *FlagSet {
	return ctx.unknownFlags
}

//
// Before go into the sub command, we need traverse flags and merge with parent
func (ctx *Context) EnterCommand(cmd *Command) {
	ctx.command = cmd
	if !cmd.EnableUnknownFlag {
		ctx.unknownFlags = nil
	} else if ctx.unknownFlags == nil {
		ctx.unknownFlags = NewFlagSet()
	}

	parentFlags := ctx.flags
	ctx.flags = cmd.flags
	ctx.flags.mergeWith(parentFlags, func(f *Flag) bool {
		return f.Persistent
	})
	ctx.flags.Add(HelpFlag)
}

func (ctx *Context) CheckFlags() error {
	for _, f := range ctx.flags.Flags() {
		if f.Required && !f.IsAssigned() {
			if !f.useDefaultValue() {
				return fmt.Errorf("missing flag --%s", f.Name)
			}
		}
	}
	return nil
}

func (ctx *Context) DetectFlag(name string) (*Flag, error) {
	flag := ctx.flags.Get(name)

	if flag != nil {
		return flag, nil
	} else if ctx.unknownFlags != nil {
		return ctx.unknownFlags.AddByName(name)
	} else {
		return nil, NewInvalidFlagError(name, ctx)
	}
}

func (ctx *Context) DetectFlagByShorthand(ch rune) (*Flag, error) {
	flag := ctx.flags.GetByShorthand(ch)
	if flag != nil {
		return flag, nil
	} else {
		return nil, fmt.Errorf("unknown flag -%s", string(ch))
	}
}
