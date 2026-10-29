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

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func testTelemetryContext(t *testing.T) (*cli.Context, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	t.Setenv(telemetry.TelemetryDisableEnv, "")
	configDir := filepath.Join(home, ".aliyun")
	stdout := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, new(bytes.Buffer))
	// The root command may still accept its global --config-path flag, but
	// telemetry is installation-wide and must ignore that profile override.
	AddFlags(ctx.Flags())
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(t.TempDir(), "config.json"))
	return ctx, stdout, configDir
}

func enterTelemetrySubcommand(t *testing.T, ctx *cli.Context, name string) *cli.Command {
	t.Helper()
	parent := NewConfigureTelemetryCommand()
	ctx.EnterCommand(parent)
	sub := parent.GetSubCommand(name)
	require.NotNil(t, sub)
	ctx.EnterCommand(sub)
	return sub
}

func TestConfigureTelemetryDoesNotInheritPersistentFlags(t *testing.T) {
	ctx, _, _ := testTelemetryContext(t)
	cmd := NewConfigureTelemetryCommand()
	require.True(t, cmd.DisablePersistentFlags)

	ctx.EnterCommand(cmd)
	for _, flag := range ctx.Flags().Flags() {
		require.Equal(t, "help", flag.Name)
	}
}

func TestConfigureTelemetryEnableAndDisable(t *testing.T) {
	ctx, _, configDir := testTelemetryContext(t)
	enable := enterTelemetrySubcommand(t, ctx, "enable")
	require.NoError(t, enable.Run(ctx, nil))

	cfg, err := telemetry.Load(configDir)
	require.NoError(t, err)
	require.True(t, cfg.Enabled)

	disable := NewConfigureTelemetryCommand().GetSubCommand("disable")
	require.NotNil(t, disable)
	require.NoError(t, disable.Run(ctx, nil))
	cfg, err = telemetry.Load(configDir)
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
}

func TestConfigureTelemetryShowIncludesState(t *testing.T) {
	ctx, stdout, configDir := testTelemetryContext(t)
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, telemetry.Save(configDir, &telemetry.Config{Enabled: true}))
	id, err := telemetry.EnsureInstanceID(configDir)
	require.NoError(t, err)
	show := enterTelemetrySubcommand(t, ctx, "show")
	require.NoError(t, show.Run(ctx, nil))
	require.Contains(t, stdout.String(), `"enabled": true`)
	require.Contains(t, stdout.String(), `"environment_disabled": false`)
	require.Contains(t, stdout.String(), `"public_cache_files": 0`)
	require.Contains(t, stdout.String(), `"instance_id_prefix": "`+id[:8]+`..."`)
	require.NotContains(t, stdout.String(), "last_uploaded_at")

	t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_DISABLE", "1")
	stdout.Reset()
	require.NoError(t, show.Run(ctx, nil))
	require.Contains(t, stdout.String(), `"enabled": true`)
	require.Contains(t, stdout.String(), `"environment_disabled": true`)
}

func TestConfigureTelemetryLifecycleDoesNotCollect(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, name := range []string{"", "show", "enable", "disable", "purge"} {
			t.Run(fmt.Sprintf("enabled=%v/%s", enabled, name), func(t *testing.T) {
				ctx, _, dir := testTelemetryContext(t)
				require.NoError(t, telemetry.Save(dir, &telemetry.Config{Enabled: enabled}))
				var id string
				if name == "purge" {
					var err error
					id, err = telemetry.EnsureInstanceID(dir)
					require.NoError(t, err)
					_, err = telemetry.WriteEvent(telemetry.GetPublicCacheDir(dir), telemetry.Event{EventID: "purge-test"})
					require.NoError(t, err)
				}
				args := []string{"configure", "telemetry"}
				cmd := NewConfigureTelemetryCommand()
				ctx.EnterCommand(cmd)
				if name != "" {
					args = append(args, name)
					cmd = cmd.GetSubCommand(name)
					ctx.EnterCommand(cmd)
				}
				telemetry.Begin(telemetry.BeginInput{ConfigDir: dir, CommandToken: "configure", CommandArgs: args})
				runErr := cmd.Run(ctx, nil)
				telemetry.End(runErr)
				require.NoError(t, runErr)
				cfg, err := telemetry.Load(dir)
				require.NoError(t, err)
				wantEnabled := enabled
				if name == "enable" {
					wantEnabled = true
				} else if name == "disable" {
					wantEnabled = false
				}
				require.Equal(t, wantEnabled, cfg.Enabled)
				require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir), "management commands must not create an ID")
				require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
				if id != "" {
					require.FileExists(t, filepath.Join(dir, "telemetry-instance-id.lock"))
				}
			})
		}
	}
}

func TestConfigureTelemetryEnableDisablePreservesInstanceID(t *testing.T) {
	ctx, _, dir := testTelemetryContext(t)
	id, err := telemetry.EnsureInstanceID(dir)
	require.NoError(t, err)
	for _, name := range []string{"disable", "enable"} {
		cmd := NewConfigureTelemetryCommand().GetSubCommand(name)
		require.NoError(t, cmd.Run(ctx, nil))
		got, err := telemetry.LoadInstanceID(dir)
		require.NoError(t, err)
		require.Equal(t, id, got)
	}
}

func TestConfigureTelemetryPurgeWithFlagsUsesActualCommandRoute(t *testing.T) {
	for _, args := range [][]string{
		{"configure", "--profile:foo", "telemetry", "purge"},
		{"--skip-secure-verify", "configure", "telemetry", "purge"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, _, dir := testTelemetryContext(t)
			require.NoError(t, telemetry.Save(dir, &telemetry.Config{Enabled: true}))
			_, err := telemetry.EnsureInstanceID(dir)
			require.NoError(t, err)
			root := &cli.Command{Name: "aliyun"}
			AddFlags(root.Flags())
			root.AddSubCommand(NewConfigureCommand())
			ended := false
			root.AfterExecute = func(ctx *cli.Context, args []string, err error) {
				require.NoError(t, err)
				telemetry.HookEnd(ctx, args, err)
				ended = true
			}
			ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
			ctx.EnterCommand(root)
			commandArgs := sysmock.StripLeadingGlobalFlags(args)
			telemetry.Begin(telemetry.BeginInput{ConfigDir: dir, CommandArgs: commandArgs, CommandToken: commandArgs[0]})
			root.Execute(ctx, args)
			require.True(t, ended)
			require.NoFileExists(t, telemetry.GetInstanceIDFilePath(dir))
			require.NoDirExists(t, filepath.Join(dir, telemetry.CacheDirName))
		})
	}
}
