package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/stretchr/testify/require"
)

func TestSettingsCommandsRejectUnexpectedArgumentsAndUnreadableFiles(t *testing.T) {
	for _, family := range []struct {
		name, filename, wantError string
		command                   func() *cli.Command
	}{
		{name: "ai-mode", filename: aimode.AiConfigFileName, wantError: "load ai-mode config failed", command: NewConfigureAiModeCommand},
		{name: "safety-policy", filename: safety.SafetyPolicyFileName, wantError: "load", command: NewConfigureSafetyPolicyCommand},
	} {
		root := family.command()
		for _, name := range append([]string{""}, root.SubCommandNames()...) {
			t.Run(family.name+"/"+name, func(t *testing.T) {
				dir := t.TempDir()
				ctx, _ := testAiModeContext(t, dir)
				parent := family.command()
				ctx.EnterCommand(parent)
				command := parent
				if name != "" {
					command = parent.GetSubCommand(name)
					ctx.EnterCommand(command)
				}
				err := command.Run(ctx, []string{"unexpected"})
				require.ErrorContains(t, err, "unexpected")
				require.NoError(t, os.Mkdir(filepath.Join(dir, family.filename), 0700))
				require.ErrorContains(t, command.Run(ctx, nil), family.wantError)
				info, err := os.Stat(filepath.Join(dir, family.filename))
				require.NoError(t, err)
				require.True(t, info.IsDir())
			})
		}
	}
}

func TestPluginSettingsCommandsRejectUnreadableConfig(t *testing.T) {
	for _, name := range []string{"", "show", "set", "clear"} {
		t.Run(name, func(t *testing.T) {
			ctx, _, dir := testPluginSettingsIsolatedHome(t)
			root := NewConfigurePluginSettingsCommand()
			ctx.EnterCommand(root)
			command := root
			if name != "" {
				command = root.GetSubCommand(name)
				ctx.EnterCommand(command)
			}
			require.ErrorContains(t, command.Run(ctx, []string{"unexpected"}), "unexpected")
			require.NoError(t, os.MkdirAll(dir, 0700))
			require.NoError(t, os.Mkdir(filepath.Join(dir, "plugin-settings.json"), 0700))
			if name == "set" {
				flag := ctx.Flags().Get("source-base")
				flag.SetValue("https://mirror.example")
				flag.SetAssigned(true)
			}
			require.ErrorContains(t, command.Run(ctx, nil), "load plugin-settings failed")
		})
	}
}
