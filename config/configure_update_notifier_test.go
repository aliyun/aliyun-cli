package config

import (
	"bytes"
	"encoding/json"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureUpdateNotifier(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(updatenotifier.DisableEnv, "0")
	cmd := NewConfigureUpdateNotifierCommand()
	for _, name := range []string{"disable", "enable", "show"} {
		var out, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&out, &stderr)
		require.NoError(t, cmd.GetSubCommand(name).Run(ctx, nil))
		require.Empty(t, stderr.String())
		cfg, err := updatenotifier.Load(GetConfigPath())
		require.NoError(t, err)
		if name == "disable" {
			require.False(t, cfg.Enabled)
		} else {
			require.True(t, cfg.Enabled)
		}
		if name == "show" {
			var result map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &result))
			require.Equal(t, false, result["environment_disabled"])
		} else {
			require.Empty(t, out.String())
		}
	}
	require.Nil(t, cmd.GetSubCommand("purge"))
}

func TestConfigureUpdateNotifierRejectsUnexpectedArgumentsAndBadSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(updatenotifier.DisableEnv, "0")
	cmd := NewConfigureUpdateNotifierCommand()
	for _, c := range []*cli.Command{cmd, cmd.GetSubCommand("show"), cmd.GetSubCommand("enable"), cmd.GetSubCommand("disable")} {
		var out, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&out, &stderr)
		ctx.EnterCommand(c)
		err := c.Run(ctx, []string{"unexpected"})
		require.ErrorContains(t, err, "unexpected")
		require.Empty(t, out.String())
	}
	dir := GetConfigPath()
	require.NoError(t, os.WriteFile(filepath.Join(dir, updatenotifier.ConfigFileName), []byte("broken"), 0600))
	var out, stderr bytes.Buffer
	require.Error(t, cmd.Run(cli.NewCommandContext(&out, &stderr), nil))
	require.Empty(t, out.String())
	require.NoError(t, updatenotifier.Save(dir, true))
	t.Setenv(updatenotifier.DisableEnv, "1")
	require.NoError(t, cmd.Run(cli.NewCommandContext(&out, &stderr), nil))
	var setting struct {
		Enabled             bool `json:"enabled"`
		EnvironmentDisabled bool `json:"environment_disabled"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &setting))
	require.True(t, setting.Enabled)
	require.True(t, setting.EnvironmentDisabled)
}
