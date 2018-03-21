/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"os"
)

type Command struct {
	// Command Name
	Name   string

	// Short is the short description shown in the 'help' output.
	Short  *i18n.Text

	// Long is the long message shown in the 'help <this-command>' output.
	Long   *i18n.Text

	// Syntax for usage
	Usage  string

	// Sample command
	Sample string

	// Enable unknown flags
	EnableUnknownFlag bool

	// enable suggest distance,
	// disable -1
	// 0: default distance
	SuggestDistance int

	// Hidden command
	Hidden bool

	// Run, command error will be catch
	Run func(ctx *Context, args []string) error

	// Help
	Help func(ctx *Context, args []string) error

	// auto compete
	AutoComplete func(ctx *Context) []string

	suggestDistance int
	parent			*Command
	subCommands     []*Command
	flags        	*FlagSet
}

func (c *Command) AddSubCommand(cmd *Command) {
	cmd.parent = c
	c.subCommands = append(c.subCommands, cmd)
}

func (c *Command) Flags() (*FlagSet) {
	if c.flags == nil {
		c.flags = NewFlagSet()
	}
	return c.flags
}

func (c *Command) Execute(args []string) {
	ctx := NewCommandContext()
	ctx.EnterCommand(c)
	ctx.completion = NewCompletion()

	//
	// if
	if ctx.completion != nil {
		args = ctx.completion.GetArgs()
	}

	err := c.executeInner(ctx, args)
	if err != nil {
		c.processError(err)
	}
}

func (c *Command) GetSubCommand(s string) (*Command) {
	for _, cmd := range c.subCommands {
		if cmd.Name == s {
			return cmd
		}
	}
	return nil
}

func (c *Command) GetSuggestions(s string) []string {
	sr := NewSuggester(s, c.GetSuggestDistance())
	for _, cmd := range c.subCommands {
		sr.Apply(cmd.Name)
	}
	return sr.GetResults()
}

func (c *Command) GetSuggestDistance() int {
	if c.SuggestDistance < 0 {
		return 0
	} else if c.SuggestDistance == 0 {
		return DefaultSuggestDistance
	} else {
		return c.SuggestDistance
	}
}

func (c *Command) GetUsageWithParent() string {
	usage := c.Usage
	for p := c.parent; p != nil; p = p.parent {
		usage = p.Name + " " + usage
	}
	return usage
}

func (c *Command) executeInner(ctx *Context, args []string) error {
	//
	// fmt.Printf(">>> Execute Command: %s args=%v\n", c.Name, args)
	parser := NewParser(args, func(s string) (*Flag, error) {
		return ctx.DetectFlag(s)
	})

	//
	// get next arg
	nextArg, _, err := parser.ReadNextArg()
	if err != nil {
		return err
	}

	//
	// if next arg is help, run help
	if nextArg == "help" {
		ctx.help = true
		return c.executeInner(ctx, parser.GetRemains())
	}

	//
	// if next args is not empty, try find sub commands
	if nextArg != "" {
		//
		// if has sub command, run it
		subCommand := c.GetSubCommand(nextArg)
		if subCommand != nil {
			ctx.EnterCommand(subCommand)
			return subCommand.executeInner(ctx, parser.GetRemains())
		}

		//
		// no sub command and command.Run == nil
		// raise error
		if c.Run == nil {
			// c.executeHelp(ctx, args, fmt.Errorf("unknown command: %s", nextArg))
			return NewInvalidCommandError(nextArg, ctx)
		}
	}

	//
	// parse remain args
	remainArgs, err := parser.ReadAll()
	if err != nil {
		return fmt.Errorf("parse failed %s", err)
	}

	//
	// check flags
	err = ctx.CheckFlags()
	if err != nil {
		return err
	}

	if ctx.flags.IsAssigned("help") {
		ctx.help = true
	}
	callArgs := make([]string, 0)
	if nextArg != "" {
		callArgs = append(callArgs, nextArg)
	}
	for _, s := range remainArgs {
		if s != "help" {
			callArgs = append(callArgs, s)
		} else {
			ctx.help = true
		}
	}

	if ctx.help {
		c.executeHelp(ctx, callArgs)
		return nil
	} else if c.Run == nil {
		c.executeHelp(ctx, callArgs)
		return nil
	} else {
		return c.Run(ctx, callArgs)
	}
}

func (c *Command) processError(err error) {
	//
	// process error
	//if e, ok := err.(PrintableError); ok {
	//	Errorf("error: %s\n", e.GetText(i18n.GetLanguage()))
	//} else {
		Errorf("ERROR: %s\n", err.Error())
//	}

	if e, ok := err.(SuggestibleError); ok {
		ss := e.GetSuggestions()
		if len(ss) > 0 {
			Noticef("\ndid you mean: \n  %s \n", ss[0])
		}
	}
}

func (c *Command) executeHelp(ctx *Context, args []string)  {
	if c.Help != nil {
		err := c.Help(ctx, args)
		if err != nil {
			Errorf("Error: %s\n", err)
		}
		if se, ok := err.(SuggestibleError); ok {
			ss := se.GetSuggestions()
			if len(ss) > 0 {
				Noticef("\nDid you mean:\n")
				for _, s := range ss {
					Noticef("  %s\n", s)
				}
			}
		}
		return
	}

	c.PrintHead()
	c.PrintUsage()
	c.PrintSubCommands()
	c.PrintFlags(ctx)
	c.PrintTail()
}