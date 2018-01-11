package cli

import (
	"io"
	"fmt"
	"os"
)

//
// CLI Tool set
// why not choose a open source library
// because aliyun cli need process unknown flags,
// and the following popular library did not support this feature
// - https://github.com/spf13/cobra,
// - https://github.com/urfave/cli
type Command struct {
	Parent *Command
	Name   string
	Short  string
	Usage  string

	Run  func(c *Command, args []string) error
	Help func(c *Command, args []string, writer io.Writer)

	subCommands     []*Command
	flags        	*FlagSet
	unknownFlags	*FlagSet
}

func (c *Command) AddSubCommand(cmd *Command) {
	cmd.Parent = c
	cmd.mergeParentFlags()
	c.subCommands = append(c.subCommands, cmd)
}

func (c *Command) Flags() (*FlagSet) {
	if c.flags == nil {
		c.flags = NewFlagSet()
	}
	return c.flags
}

func (c *Command) EnableUnknownFlags() {
	c.unknownFlags = NewFlagSet()
}

func (c *Command) UnknownFlags() (*FlagSet) {
	return c.unknownFlags
}

func (c *Command) Execute(args []string) {
	// fmt.Printf(">>> Execute %s\n", c.Name)

	sub := GetFirstArgs(args)
	if sub == "help" {
		c.ExecuteHelp(args[1:])
	}

	if len(c.subCommands) > 0 {
		if sub != "" {
			c2 := c.findSubCommand(sub)
			if c2 != nil {
				c2.Execute(args[1:])
				return
			}
		}
	}

	if c.Run == nil {
		c.PrintHelp(fmt.Errorf("command not implemented"))
		return
	}

	args, err := c.flags.ParseArgs(args, c.UnknownFlags())
	if err != nil {
		c.PrintHelp(err)
		return
	}

	err = c.Run(c, args)
	if err != nil {
		c.PrintHelp(err)
	}
}

func (c *Command) ExecuteHelp(args []string) {
	sub := GetFirstArgs(args)

	if len(c.subCommands) > 0 {
		if sub != "" {
			c2 := c.findSubCommand(sub)
			if c2 != nil {
				c2.ExecuteHelp(args[1:])
				return
			}
		}
	}

	if c.Help != nil {
		c.Help(c, args[1:], os.Stdout)
		return
	} else {
		c.PrintHelp(nil)
	}
	c.Run(c, args)
}

func (c *Command) PrintHelp(err error) {
	if err != nil {
		fmt.Printf("failed: %v\n", err)
	}
	fmt.Println("Alibaba Cloud CLI v0.12")
	fmt.Println("Usage:")
	fmt.Println("\taliyun configure --profile ...")
	fmt.Printf("\taliyun [Product] [Api] --parameter1 value1 --parametere2 value2 ...\n")
	fmt.Printf("\tSample: aliyun Ecs DescribeRegions \n")
}

func (c *Command) findSubCommand(s string) (*Command) {
	for _, cmd := range c.subCommands {
		if cmd.Name == s {
			return cmd
		}
	}
	return nil
}

func (c *Command) mergeParentFlags() {
	for p := c.Parent; p != nil; p = p.Parent {
		for _, f := range p.Flags().Items() {
			if f.Persistent {
				c.flags.Add(f)
			}
		}
	}
}
