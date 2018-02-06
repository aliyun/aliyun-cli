/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package cli

import (
	"fmt"
	"text/tabwriter"
	"os"
)

type Command struct {
	Parent *Command
	Name   string
	Short  string
	Usage  string
	Sample string
	EnableUnknownFlag bool

	Run func(ctx *Context, args []string) error
	Help func(ctx *Context, args[] string, err error)

	subCommands     []*Command
	flags        	*FlagSet
}

func (c *Command) AddSubCommand(cmd *Command) {
	cmd.Parent = c
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
	c.executeInner(ctx, args)
}

func (c *Command) executeInner(ctx *Context, args []string) {
	// fmt.Printf(">>> Execute Command: %s args=%v\n", c.Name, args)
	parser := NewParser(args, func(s string) (*Flag, error) {
		return ctx.DetectFlag(s)
	})

	nextArg, _, err := parser.ReadNextArg()
	if err != nil {
		c.ExecuteHelp(ctx, args, err)
	}

	if nextArg == "help" {
		ctx.help = true
		c.executeInner(ctx, parser.GetRemains())
	} else {
		subCommand := c.getSubCommand(nextArg)

		if subCommand != nil {
			ctx.EnterCommand(subCommand)
			subCommand.executeInner(ctx, parser.GetRemains())
		} else {
			if c.Run == nil {
				c.ExecuteHelp(ctx, args, fmt.Errorf("unknown command: %s", nextArg))
			} else {
				remainArgs, err := parser.ReadAll()
				if err != nil {
					c.ExecuteHelp(ctx, args, err)
				}
				err = ctx.CheckFlags()
				if err != nil {
					c.ExecuteHelp(ctx, args, err)
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
					c.ExecuteHelp(ctx, callArgs, nil)
				} else {
					c.Run(ctx, callArgs)
				}
			}
		}
	}
}

func (c *Command) ExecuteHelp(ctx *Context, args []string, err error) {
	if c.Help != nil {
		c.Help(ctx, args, err)
		return
	}

	c.PrintHead()
	c.PrintUsage()
	c.PrintSubCommands()
	c.PrintFlags()
	c.PrintTail()
}

func (c *Command) getSubCommand(s string) (*Command) {
	for _, cmd := range c.subCommands {
		if cmd.Name == s {
			return cmd
		}
	}
	return nil
}


func (c *Command) PrintHead(){
	fmt.Printf("%s\n", c.Short)
}

func (c *Command) PrintUsage() {
	if c.Usage != "" {
		fmt.Printf("\nUsage:\n  %s\n", c.Usage)
	} else {
		c.PrintSubCommands()
	}
}

func (c *Command) PrintSample() {
	if c.Sample != "" {
		fmt.Printf("\nSample:\n  %s\n", c.Sample)
	}
}

func (c *Command) PrintSubCommands() {
	fmt.Printf("\nCommands:\n")

	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	if len(c.subCommands) > 0 {
		for _, cmd := range c.subCommands {
			fmt.Fprintf(w, "  %s\t%s\n", cmd.Name, cmd.Usage)
		}
	} else {
		fmt.Printf("  %s\n", c.Usage)
	}
	w.Flush()
}

func (c *Command) PrintFlags() {
	if len(c.flags.Flags()) == 0 {
		return
	}

	fmt.Printf("\nFlags:\n")
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, flag := range c.Flags().Flags() {
		if flag.Hidden {
			continue
		}
		fmt.Fprintf(w, "  --%s\t%s\n", flag.Name, flag.Usage)
	}
	w.Flush()
}

func (c *Command) PrintFailed(err error, suggestion string) {
	Errorf("ERROR: %v\n", err)
	fmt.Printf("%s\n", suggestion)
}

func (c *Command) PrintTail() {
	fmt.Printf("\nUse `%s --help` for more information.\n", c.Name)
}
