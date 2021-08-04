// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
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
	Errorf(ctx.Stderr(), "ERROR: %v\n", err)
	Printf(ctx.Stderr(), "%s\n", suggestion)
}

func (c *Command) PrintTail(ctx *Context) {
	Printf(ctx.Writer(), "\nUse `%s --help` for more information.\n", c.Name)
}
