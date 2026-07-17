// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/i18n"
)

var cmd = &Command{
	Name:  "oss",
	flags: NewFlagSet(),
}

func TestHelpFlag(t *testing.T) {
	fs := NewFlagSet()
	fs.Add(NewHelpFlag())
	f := HelpFlag(fs)
	assert.Equal(t, &Flag{Name: "help", Shorthand: 'h', Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone}, f)
}

func TestNewHelpFlag(t *testing.T) {
	f := NewHelpFlag()
	assert.Equal(t, &Flag{Name: "help", Shorthand: 'h', Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone}, f)
}

func TestContext_SetUnknownFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	ctx.SetUnknownFlags(NewFlagSet())
	assert.NotNil(t, ctx.unknownFlags)
}

func TestNewCommandContext(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	assert.Equal(t, &Context{
		flags:        NewFlagSet(),
		unknownFlags: nil,
		stdout:       w,
		stderr:       stderr,
		help:         false,
		command:      nil,
		completion:   nil,
		runtimeEnvs:  map[string]string{},
	}, ctx)
}

func TestCtx(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	assert.False(t, ctx.IsHelp())
	assert.Nil(t, ctx.Command())
	assert.Nil(t, ctx.Completion())
	assert.Equal(t, ctx.flags, ctx.Flags())
	assert.Equal(t, w, ctx.Stdout())
	assert.Nil(t, ctx.UnknownFlags())
	ctx.SetCompletion(&Completion{Current: "M", Args: []string{"GOOD", "BAD"}, line: "MrX", point: 2})
	assert.Equal(t, &Completion{Current: "M", Args: []string{"GOOD", "BAD"}, line: "MrX", point: 2}, ctx.Completion())

	//EnterCommand
	ctx.EnterCommand(cmd)
	assert.Nil(t, ctx.unknownFlags)
	ctx.EnterCommand(cmd)
	assert.Equal(t, &Flag{Name: "help", Shorthand: 'h', Short: i18n.T("print help", "打印帮助信息"), AssignedMode: AssignedNone}, ctx.flags.Get("help"))
}

func TestCheckFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	ctx.flags.AddByName("MrX")
	assert.Nil(t, ctx.CheckFlags())
	ctx.flags.flags[0].Required = true
	assert.EqualError(t, ctx.CheckFlags(), "missing flag --MrX")
	ctx.flags.flags[0].assigned = true
	ctx.flags.flags[0].Fields = []Field{{Key: "m", Required: true}}
	assert.EqualError(t, ctx.CheckFlags(), "bad flag format --MrX with field m= required")
	ctx.flags.flags[0].Fields[0].Required = false
	ctx.flags.flags[0].ExcludeWith = []string{"MrX"}
	ctx.flags.flags[0].value = "M"
	assert.EqualError(t, ctx.CheckFlags(), "flag --MrX is exclusive with --MrX")
}

func TestDetectFlag(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
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
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	ctx.flags.Add(&Flag{Name: "profile", Shorthand: 'p'})
	f, err := ctx.detectFlagByShorthand('p')
	assert.Equal(t, &Flag{Name: "profile", Shorthand: 'p'}, f)
	assert.Nil(t, err)
	f, err = ctx.detectFlagByShorthand('c')
	assert.Nil(t, f)
	assert.EqualError(t, err, "unknown flag -c")
}

func TestSetInConfigureMode(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)

	ctx.SetInConfigureMode(true)
	assert.True(t, ctx.InConfigureMode())

	ctx.SetInConfigureMode(false)
	assert.False(t, ctx.InConfigureMode())
}

func TestDetectFlagByShorthandEnableUnknown(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)
	ctx.command = newTestCmd()
	ctx.command.EnableUnknownFlag = true
	ctx.unknownFlags = NewFlagSet()
	f, err := ctx.detectFlagByShorthand('c')
	assert.NotNil(t, f)
	assert.Nil(t, err)
}

func TestContext_EnterCommand_DisablePersistentFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)

	ctx.flags.Add(&Flag{Name: "persistent", Persistent: true})

	// Subcommand with DisablePersistentFlags = true
	subCmd := &Command{
		Name:                   "sub",
		DisablePersistentFlags: true,
	}

	ctx.EnterCommand(subCmd)

	assert.Nil(t, ctx.flags.Get("persistent"))
	assert.NotNil(t, ctx.flags.Get("help"))
}

func TestContext_EnterCommand_EnablePersistentFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)

	ctx.flags.Add(&Flag{Name: "persistent", Persistent: true})
	ctx.flags.Add(&Flag{Name: "local", Persistent: false})

	// Subcommand with DisablePersistentFlags = false (default)
	subCmd := &Command{
		Name:                   "sub",
		DisablePersistentFlags: false,
	}

	ctx.EnterCommand(subCmd)

	assert.NotNil(t, ctx.flags.Get("persistent"))
	assert.Nil(t, ctx.flags.Get("local"))
}

func TestContext_SetCommand_DisablePersistentFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)

	ctx.flags.Add(&Flag{Name: "persistent", Persistent: true})

	// Command with DisablePersistentFlags = true
	cmd := &Command{
		Name:                   "test",
		DisablePersistentFlags: true,
	}

	ctx.SetCommand(cmd)

	assert.Nil(t, ctx.flags.Get("persistent"))
	assert.NotNil(t, ctx.flags.Get("help"))
}

func TestContext_SetCommand_EnablePersistentFlags(t *testing.T) {
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := NewCommandContext(w, stderr)

	ctx.flags.Add(&Flag{Name: "persistent", Persistent: true})
	ctx.flags.Add(&Flag{Name: "local", Persistent: false})

	// Command with DisablePersistentFlags = false (default)
	cmd := &Command{
		Name:                   "test",
		DisablePersistentFlags: false,
	}

	ctx.SetCommand(cmd)

	assert.NotNil(t, ctx.flags.Get("persistent"))
	assert.Nil(t, ctx.flags.Get("local"))
}

func TestIsPluginSubCommandArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		// plugin commands (all-lowercase subcommand)
		{"fc list-tag-resources", []string{"fc", "list-tag-resources", "--resource-id", "a"}, true},
		{"ecs describe-instances", []string{"ecs", "describe-instances"}, true},
		{"cs get-cluster", []string{"cs", "get-cluster"}, true},
		{"fc version", []string{"fc", "version"}, true},

		// with help prefix
		{"help fc list-tag-resources", []string{"help", "fc", "list-tag-resources"}, true},

		// OpenAPI PascalCase commands — NOT plugin
		{"ecs DescribeInstances", []string{"ecs", "DescribeInstances"}, false},
		{"vpc CreateVpc", []string{"vpc", "CreateVpc"}, false},
		{"rds DescribeDBInstances", []string{"rds", "DescribeDBInstances"}, false},

		// HTTP method subcommands — NOT plugin
		{"ecs GET", []string{"ecs", "GET"}, false},
		{"sls POST", []string{"sls", "POST"}, false},
		{"ecs put", []string{"ecs", "put"}, false},
		{"ecs delete", []string{"ecs", "delete"}, false},
		{"ecs get", []string{"ecs", "get"}, false},

		// first arg starts with dash (global flag before product)
		{"--profile before product", []string{"--profile", "default", "fc", "list-tag-resources"}, false},
		{"--region before product", []string{"--region", "cn-hangzhou", "ecs", "describe-instances"}, false},

		// subcommand starts with dash
		{"product then flag", []string{"fc", "--help"}, false},

		// too few args
		{"only product", []string{"fc"}, false},
		{"empty args", []string{}, false},
		{"only help", []string{"help"}, false},
		{"help with one arg", []string{"help", "fc"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPluginSubCommandArgs(tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Plugin-style invocations (product + lowercase subcommand) must still register unknown
// long flags so the following token is consumed as the flag value, not as a third positional.
func TestPluginSubcommand_unknownLongFlagValueNotPositional(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"aliyun", "config", "create-compliance-pack", "--api-version", "2020-09-07"}

	var gotArgs []string
	cmdRoot := &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		Run: func(ctx *Context, args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}
	ctx := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	ctx.EnterCommand(cmdRoot)
	cmdRoot.Execute(ctx, []string{"config", "create-compliance-pack", "--api-version", "2020-09-07"})

	assert.Equal(t, []string{"config", "create-compliance-pack"}, gotArgs)
	uf := ctx.UnknownFlags().Get("api-version")
	assert.NotNil(t, uf)
	v, ok := uf.GetValue()
	assert.True(t, ok)
	assert.Equal(t, "2020-09-07", v)
	assert.True(t, uf.allowRepeatedUnknown)
}

// Flags may appear between product and lowercase subcommand (os.Args[1] is not the subcommand),
// so HasPluginSubCommand is false; unknown long flags must still consume the next argv as value.
func TestPluginSubcommand_unknownFlagBetweenProductAndSubcommand(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"aliyun", "config", "--api-version", "2020-09-07", "create-compliance-pack"}

	var gotArgs []string
	cmdRoot := &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		Run: func(ctx *Context, args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}
	ctx := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	ctx.EnterCommand(cmdRoot)
	cmdRoot.Execute(ctx, []string{"config", "--api-version", "2020-09-07", "create-compliance-pack"})

	assert.Equal(t, []string{"config", "create-compliance-pack"}, gotArgs)
	assert.False(t, ctx.HasPluginSubCommand())
	uf := ctx.UnknownFlags().Get("api-version")
	assert.NotNil(t, uf)
	v, ok := uf.GetValue()
	assert.True(t, ok)
	assert.Equal(t, "2020-09-07", v)
	assert.False(t, uf.allowRepeatedUnknown)
}

func TestDetectFlagByShorthand_unknownPluginSubcommand(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"aliyun", "fc", "list-tag-resources", "-x", "a"}

	ctx := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	cmdRoot := &Command{Name: "aliyun", EnableUnknownFlag: true}
	ctx.EnterCommand(cmdRoot)

	assert.True(t, ctx.HasPluginSubCommand())

	f1, err := ctx.detectFlagByShorthand('x')
	assert.NoError(t, err)
	assert.NotNil(t, f1)
	assert.Equal(t, "x", f1.Name)
	assert.True(t, f1.allowRepeatedUnknown, "plugin unknown shorthand should allow repeat")

	f2, err := ctx.detectFlagByShorthand('x')
	assert.NoError(t, err)
	assert.Equal(t, f1, f2, "second -x should reuse same Flag via Get")

	// Non-plugin shape: second argv token is not the lowercase subcommand
	os.Args = []string{"aliyun", "ecs", "DescribeInstances"}
	ctx2 := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	ctx2.EnterCommand(cmdRoot)
	assert.False(t, ctx2.HasPluginSubCommand())

	fy, err := ctx2.detectFlagByShorthand('y')
	assert.NoError(t, err)
	assert.NotNil(t, fy)
	assert.False(t, fy.allowRepeatedUnknown)
}

func TestPluginSubcommand_repeatedUnknownFlagValues(t *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"aliyun", "fc", "list-tag-resources", "--foo", "a", "--foo", "b"}

	var gotArgs []string
	cmdRoot := &Command{
		Name:              "aliyun",
		EnableUnknownFlag: true,
		Run: func(ctx *Context, args []string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}
	ctx := NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	ctx.EnterCommand(cmdRoot)
	cmdRoot.Execute(ctx, []string{"fc", "list-tag-resources", "--foo", "a", "--foo", "b"})

	assert.Equal(t, []string{"fc", "list-tag-resources"}, gotArgs)
	uf := ctx.UnknownFlags().Get("foo")
	assert.NotNil(t, uf)
	v, ok := uf.GetValue()
	assert.True(t, ok)
	assert.Equal(t, "b", v)
}
