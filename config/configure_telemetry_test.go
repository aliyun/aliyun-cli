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
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func testTelemetryContext(t *testing.T) (*cli.Context, *bytes.Buffer, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
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

func TestConfigureTelemetryShowIncludesPublicState(t *testing.T) {
	ctx, stdout, configDir := testTelemetryContext(t)
	require.NoError(t, telemetry.Save(configDir, &telemetry.Config{
		Enabled: true,
	}))
	show := enterTelemetrySubcommand(t, ctx, "show")
	require.NoError(t, show.Run(ctx, nil))
	require.Contains(t, stdout.String(), `"public_enabled": true`)
	require.Contains(t, stdout.String(), `"public_cache_files": 0`)
}
