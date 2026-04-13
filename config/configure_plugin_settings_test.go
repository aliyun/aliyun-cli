package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPluginSettingsIsolatedHome(t *testing.T) (ctx *cli.Context, stdout *bytes.Buffer, aliyunDir string) {
	t.Helper()
	home := t.TempDir()
	// These tests do not use t.Parallel(); t.Setenv restores env after the test (Go 1.17+).
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
		t.Setenv("HOMEDRIVE", "")
		t.Setenv("HOMEPATH", "")
	}
	aliyunDir = filepath.Join(home, ".aliyun")
	stdout = new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx = cli.NewCommandContext(stdout, stderr)
	return ctx, stdout, aliyunDir
}

func TestConfigurePluginSettings_SourceBaseFlagOnSetOnly(t *testing.T) {
	root := NewConfigurePluginSettingsCommand()
	assert.Nil(t, root.Flags().Get("source-base"))
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	assert.NotNil(t, set.Flags().Get("source-base"))
}

func TestConfigurePluginSettings_Show_Default(t *testing.T) {
	ctx, w, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	show := root.GetSubCommand("show")
	require.NotNil(t, show)
	ctx.EnterCommand(show)
	require.NoError(t, show.Run(ctx, nil))
	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Bytes(), &m))
	assert.Equal(t, "", m["source_base"])
}

func TestConfigurePluginSettings_Set_ViaCommand(t *testing.T) {
	ctx, _, aliyunDir := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	f := ctx.Flags().Get("source-base")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("https://mirror.example.com/plugins")
	require.NoError(t, set.Run(ctx, nil))

	cfg, err := pluginsettings.Load(aliyunDir)
	require.NoError(t, err)
	assert.Equal(t, "https://mirror.example.com/plugins", cfg.SourceBase)
}

func TestConfigurePluginSettings_Clear(t *testing.T) {
	ctx, _, aliyunDir := testPluginSettingsIsolatedHome(t)
	require.NoError(t, os.MkdirAll(aliyunDir, 0755))
	require.NoError(t, pluginsettings.Save(aliyunDir, &pluginsettings.PluginSettings{
		SourceBase: "https://old.example/plugins",
	}))
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	clear := root.GetSubCommand("clear")
	require.NotNil(t, clear)
	ctx.EnterCommand(clear)
	require.NoError(t, clear.Run(ctx, nil))
	cfg, err := pluginsettings.Load(aliyunDir)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.SourceBase)
}

func TestConfigurePluginSettings_Set_MissingFlag(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	ctx.EnterCommand(set)
	err := set.Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--source-base")
}

func TestConfigurePluginSettings_ParentRun_Shows(t *testing.T) {
	ctx, w, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	require.NoError(t, root.Run(ctx, nil))
	assert.Contains(t, w.String(), "source_base")
}

func TestConfigurePluginSettings_Root_InvalidArg(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	err := root.Run(ctx, []string{"nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")
	assert.Contains(t, err.Error(), "not a vaild command")
}

func TestConfigurePluginSettings_Show_InvalidArg(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	show := root.GetSubCommand("show")
	require.NotNil(t, show)
	ctx.EnterCommand(show)
	err := show.Run(ctx, []string{"extra"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extra")
	assert.Contains(t, err.Error(), "not a vaild command")
}

func TestConfigurePluginSettings_Set_InvalidArg(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	err := set.Run(ctx, []string{"junk"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "junk")
	assert.Contains(t, err.Error(), "not a vaild command")
}

func TestConfigurePluginSettings_Set_EmptySourceBase(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	f := ctx.Flags().Get("source-base")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("")
	err := set.Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestConfigurePluginSettings_Set_WhitespaceSourceBase(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	f := ctx.Flags().Get("source-base")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("   \t  ")
	err := set.Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestConfigurePluginSettings_Set_SourceBaseNotHTTP(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	f := ctx.Flags().Get("source-base")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("ftp://mirror.example/plugins")
	err := set.Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http://")
}

func TestConfigurePluginSettings_Set_SourceBaseBareHost(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	set := root.GetSubCommand("set")
	require.NotNil(t, set)
	ctx.EnterCommand(set)
	f := ctx.Flags().Get("source-base")
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("mirror.example/plugins")
	err := set.Run(ctx, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http://")
}

func TestConfigurePluginSettings_Clear_InvalidArg(t *testing.T) {
	ctx, _, _ := testPluginSettingsIsolatedHome(t)
	root := NewConfigurePluginSettingsCommand()
	ctx.EnterCommand(root)
	clear := root.GetSubCommand("clear")
	require.NotNil(t, clear)
	ctx.EnterCommand(clear)
	err := clear.Run(ctx, []string{"oops"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oops")
	assert.Contains(t, err.Error(), "not a vaild command")
}
