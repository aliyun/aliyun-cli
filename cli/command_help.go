package cli

import (
	"fmt"
	"text/tabwriter"
	"os"
	"github.com/aliyun/aliyun-cli/i18n"
)

func (c *Command) PrintHead(){
	fmt.Printf("%s\n", c.Short.Get(i18n.GetLanguage()))
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
			if cmd.Hidden {
				continue
			}
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
		fmt.Fprintf(w, "  --%s\t%s\n", flag.Name, flag.Usage.Get(i18n.GetLanguage()))
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
