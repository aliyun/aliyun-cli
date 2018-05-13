package cli

import (
	"fmt"
	"text/tabwriter"
)

func (c *Command) PrintHead(ctx *Context) {
	Printf(ctx.Writer(), "%s\n", c.Short.Text())
	//if c.Long != nil {
	//	fmt.Printf("\n%s\n", c.Long.Text())
	//}
}

func (c *Command) PrintUsage(ctx *Context) {
	if c.Usage != "" {
		Printf(ctx.Writer(), "\nUsage:\n  %s\n", c.GetUsageWithParent())
	} else {
		c.PrintSubCommands(ctx)
	}
}

func (c *Command) PrintSample(ctx *Context) {
	if c.Sample != "" {
		Printf(ctx.Writer(), "\nSample:\n  %s\n", c.Sample)
	}
}

func (c *Command) PrintSubCommands(ctx *Context) {
	if len(c.subCommands) > 0 {
		Printf(ctx.Writer(), "\nCommands:\n")
		w := tabwriter.NewWriter(ctx.Writer(), 8, 0, 1, ' ', 0)
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
	Printf(ctx.Writer(), "\nFlags:\n")
	w := tabwriter.NewWriter(ctx.Writer(), 8, 0, 1, ' ', 0)
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

func (c *Command) PrintFailed(ctx *Context, err error, suggestion string) {
	Errorf(ctx.Writer(), "ERROR: %v\n", err)
	Printf(ctx.Writer(), "%s\n", suggestion)
}

func (c *Command) PrintTail(ctx *Context) {
	Printf(ctx.Writer(), "\nUse `%s --help` for more information.\n", c.Name)
}
