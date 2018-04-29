package cli

import (
	"fmt"
	"text/tabwriter"
)

func (c *Command) PrintHead() {
	Printf("%s\n", c.Short.Text())
	//if c.Long != nil {
	//	fmt.Printf("\n%s\n", c.Long.Text())
	//}
}

func (c *Command) PrintUsage() {
	if c.Usage != "" {
		Printf("\nUsage:\n  %s\n", c.GetUsageWithParent())
	} else {
		c.PrintSubCommands()
	}
}

func (c *Command) PrintSample() {
	if c.Sample != "" {
		Printf("\nSample:\n  %s\n", c.Sample)
	}
}

func (c *Command) PrintSubCommands() {
	if len(c.subCommands) > 0 {
		Printf("\nCommands:\n")
		w := tabwriter.NewWriter(GetOutputWriter(), 8, 0, 1, ' ', 0)
		for _, cmd := range c.subCommands {
			if cmd.Hidden {
				continue
			}
			fmt.Fprintf(w, "  %s\t%s\n", cmd.Name, cmd.Short.Text())
		}
		w.Flush()
	}
}

func (c *Command) PrintFlags(ctx *Context) {
	if len(c.Flags().Flags()) == 0 {
		return
	}
	Printf("\nFlags:\n")
	w := tabwriter.NewWriter(GetOutputWriter(), 8, 0, 1, ' ', 0)
	fs := c.Flags()
	if ctx != nil {
		fs = ctx.Flags()
	}
	for _, flag := range fs.Flags() {
		if flag.Hidden {
			continue
		}
		s := "--" + flag.Name
		if flag.Shorthand != 0 {
			s = s + ",-" + string(flag.Shorthand)
		}
		fmt.Fprintf(w, "  %s\t%s\n", s, flag.Short.Text())
	}
	w.Flush()
}

func (c *Command) PrintFailed(err error, suggestion string) {
	Errorf("ERROR: %v\n", err)
	Printf("%s\n", suggestion)
}

func (c *Command) PrintTail() {
	Printf("\nUse `%s --help` for more information.\n", c.Name)
}
