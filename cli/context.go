// Copyright 1999-2019 Alibaba Group Holding Limited
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
	"fmt"
	"io"

	"github.com/aliyun/aliyun-cli/i18n"
)

//
// default help flag

func HelpFlag(fs *FlagSet) *Flag {
	return fs.Get("help")
}

func NewHelpFlag() *Flag {
	return &Flag{
		Name:         "help",
		Short:        i18n.T("print help", "打印帮助信息"),
		AssignedMode: AssignedNone,
	}
}

//
// CLI Command Context
type Context struct {
	help         bool
	flags        *FlagSet
	unknownFlags *FlagSet
	command      *Command
	completion   *Completion
	writer       io.Writer
}

func NewCommandContext(w io.Writer) *Context {
	return &Context{
		flags:        NewFlagSet(),
		unknownFlags: nil,
		writer:       w,
	}
}

func (ctx *Context) SetUnknownFlags(flags *FlagSet) {
	ctx.unknownFlags = flags
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

func (ctx *Context) Writer() io.Writer {
	return ctx.writer
}

func (ctx *Context) UnknownFlags() *FlagSet {
	return ctx.unknownFlags
}

func (ctx *Context) SetCompletion(completion *Completion) {
	ctx.completion = completion
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

	ctx.flags = cmd.flags.mergeWith(ctx.flags, func(f *Flag) bool {
		return f.Persistent
	})
	ctx.flags.Add(NewHelpFlag())
}

func (ctx *Context) CheckFlags() error {
	for _, f := range ctx.flags.Flags() {
		if !f.IsAssigned() {
			if f.Required {
				return fmt.Errorf("missing flag --%s", f.Name)
			}
		} else {
			if err := f.checkFields(); err != nil {
				return err
			}
			if len(f.ExcludeWith) > 0 {
				for _, es := range f.ExcludeWith {
					if _, ok := ctx.flags.GetValue(es); ok {
						return fmt.Errorf("flag --%s is exclusive with --%s", f.Name, es)
					}
				}
			}
		}
	}
	return nil
}

func (ctx *Context) detectFlag(name string) (*Flag, error) {
	flag := ctx.flags.Get(name)

	if flag != nil {
		return flag, nil
	} else if ctx.unknownFlags != nil {
		return ctx.unknownFlags.AddByName(name)
	} else {
		return nil, NewInvalidFlagError(name, ctx)
	}
}

func (ctx *Context) detectFlagByShorthand(ch rune) (*Flag, error) {
	flag := ctx.flags.GetByShorthand(ch)
	if flag != nil {
		return flag, nil
	}
	return nil, fmt.Errorf("unknown flag -%s", string(ch))
}
