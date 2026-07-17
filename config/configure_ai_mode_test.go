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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAiModeContext returns a context whose GetConfigDir points to configDir (via --config-path <configDir>/config.json).
func testAiModeContext(t *testing.T, configDir string) (*cli.Context, *bytes.Buffer) {
	t.Helper()
	configFile := filepath.Join(configDir, "config.json")
	require.NoError(t, os.WriteFile(configFile, []byte(`{"current":"default","profiles":[]}`), 0600))
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	ConfigurePathFlag(ctx.Flags()).SetValue(configFile)
	return ctx, w
}

func enterAiModeSub(t *testing.T, ctx *cli.Context, name string) *cli.Command {
	t.Helper()
	ai := NewConfigureAiModeCommand()
	ctx.EnterCommand(ai)
	sub := ai.GetSubCommand(name)
	require.NotNil(t, sub, "subcommand %q", name)
	ctx.EnterCommand(sub)
	return sub
}

func TestConfigureAiMode_Show_Default(t *testing.T) {
	dir := t.TempDir()
	ctx, w := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "show")
	require.NoError(t, sub.Run(ctx, []string{}))
	assert.Contains(t, w.String(), `"enabled": false`)
	assert.Contains(t, w.String(), aimode.DefaultUserAgent)
}

func TestConfigureAiMode_ParentRun_DefaultShows(t *testing.T) {
	dir := t.TempDir()
	ctx, w := testAiModeContext(t, dir)
	ai := NewConfigureAiModeCommand()
	ctx.EnterCommand(ai)
	require.NoError(t, ai.Run(ctx, []string{}))
	assert.Contains(t, w.String(), `"enabled": false`)
}

func TestConfigureAiMode_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)

	enable := enterAiModeSub(t, ctx, "enable")
	require.NoError(t, enable.Run(ctx, []string{}))
	cfg, err := aimode.Load(dir)
	require.NoError(t, err)
	assert.True(t, cfg.Enabled)

	ctx2, _ := testAiModeContext(t, dir)
	disable := enterAiModeSub(t, ctx2, "disable")
	require.NoError(t, disable.Run(ctx2, []string{}))
	cfg2, err := aimode.Load(dir)
	require.NoError(t, err)
	assert.False(t, cfg2.Enabled)
}

func TestConfigureAiMode_SetResetUserAgent(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "set-user-agent")
	ua := ctx.Flags().Get("user-agent")
	require.NotNil(t, ua)
	ua.SetAssigned(true)
	ua.SetValue("MyAgent/1")
	require.NoError(t, sub.Run(ctx, []string{}))
	cfg, err := aimode.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "MyAgent/1", cfg.UserAgent)

	ctx2, _ := testAiModeContext(t, dir)
	reset := enterAiModeSub(t, ctx2, "reset-user-agent")
	require.NoError(t, reset.Run(ctx2, []string{}))
	cfg2, err := aimode.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "", cfg2.UserAgent)
}

func TestConfigureAiMode_SetUserAgent_MissingFlag(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "set-user-agent")
	err := sub.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--user-agent is required")
}

func TestConfigureAiMode_SetResetOssutil(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "set-ossutil")
	f := ctx.Flags().Get("ossutil")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue(`{"k":1,"s":"x"}`)
	require.NoError(t, sub.Run(ctx, []string{}))
	cfg, err := aimode.Load(dir)
	require.NoError(t, err)
	m, ok := cfg.PluginSpecialOSSUTIL.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), m["k"])
	assert.Equal(t, "x", m["s"])

	ctx2, _ := testAiModeContext(t, dir)
	reset := enterAiModeSub(t, ctx2, "reset-ossutil")
	require.NoError(t, reset.Run(ctx2, []string{}))
	cfg2, err := aimode.Load(dir)
	require.NoError(t, err)
	assert.Nil(t, cfg2.PluginSpecialOSSUTIL)
}

func TestConfigureAiMode_SetOssutil_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "set-ossutil")
	f := ctx.Flags().Get("ossutil")
	f.SetAssigned(true)
	f.SetValue(`not-json`)
	err := sub.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestConfigureAiMode_SetOssutil_MissingFlag(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "set-ossutil")
	err := sub.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--ossutil is required")
}

func TestConfigureAiMode_Subcommand_ExtraArgs(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "show")
	err := sub.Run(ctx, []string{"garbage"})
	require.Error(t, err)
}

func TestConfigureAiMode_ParentRun_ExtraArgs(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	ai := NewConfigureAiModeCommand()
	ctx.EnterCommand(ai)
	err := ai.Run(ctx, []string{"nope"})
	require.Error(t, err)
}

func TestConfigureAiMode_Show_IncludesOssutilInOutput(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, aimode.Save(dir, &aimode.AiConfig{
		Enabled:  true,
		UserAgent: "ua",
		PluginSpecialOSSUTIL: map[string]any{"a": true},
	}))
	ctx, w := testAiModeContext(t, dir)
	sub := enterAiModeSub(t, ctx, "show")
	require.NoError(t, sub.Run(ctx, []string{}))
	var out struct {
		Ossutil any `json:"ossutil"`
	}
	lines := strings.TrimSpace(w.String())
	require.NoError(t, json.Unmarshal([]byte(lines), &out))
	m, ok := out.Ossutil.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, m["a"])
}
