package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestPluginMinimumWhitespacePreservesExecutionAndInstallValidation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX plugin executable fixture")
	}
	previous := cli.Version
	cli.Version = "3.5.3"
	t.Cleanup(func() { cli.Version = previous })
	home := t.TempDir()
	cleanup := setTestHomeDir(t, home)
	t.Cleanup(cleanup)
	root := filepath.Join(home, ".aliyun", "plugins")
	path := filepath.Join(root, "aliyun-cli-whitespace")
	require.NoError(t, os.MkdirAll(path, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(path, "aliyun-cli-whitespace"), []byte("#!/bin/sh\nprintf 'plugin result\\n'\n"), 0700))
	for _, minimum := range []string{"3.5.0", " \t3.5.0\n", " \t\n", " \t3.6.0\n"} {
		t.Run(minimum, func(t *testing.T) {
			local := LocalPlugin{Name: "aliyun-cli-whitespace", Version: "1.0.0", Command: "whitespace", Path: path, MinCliVersion: minimum}
			body, err := json.Marshal(LocalManifest{Plugins: map[string]LocalPlugin{local.Name: local}})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), body, 0600))
			var out, stderr bytes.Buffer
			ctx := cli.NewCommandContext(&out, &stderr)
			found, err := ExecutePlugin("whitespace", nil, ctx)
			require.True(t, found)
			require.NoError(t, err)
			require.Equal(t, "plugin result\n", out.String())
			require.Empty(t, stderr.String())
			info := &PluginInfo{Versions: map[string]VersionInfo{"1.0.0": {
				Metadata:  &VersionMetadata{MinCliVersion: minimum},
				Platforms: map[string]PlatformInfo{GetCurrentPlatform(): {URL: "https://example.invalid/plugin"}},
			}}}
			platform, err := (&Manager{}).validateVersionAndPlatform(ctx, info, "1.0.0", local.Name)
			if minimum == " \t3.6.0\n" {
				require.ErrorContains(t, err, "requires CLI version 3.6.0 or higher")
				require.Nil(t, platform)
			} else {
				require.NoError(t, err)
				require.NotNil(t, platform)
			}
		})
	}
}

func TestPluginCompatibilityNoticeUsesTrimmedMinimum(t *testing.T) {
	previous := cli.Version
	cli.Version = "3.5.3"
	t.Cleanup(func() { cli.Version = previous })
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	var out, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&out, &stderr)
	local := &LocalPlugin{Name: "example", MinCliVersion: " \t3.6.0\n"}
	require.NoError(t, CheckLocalPluginCliVersion(ctx, "example", local))
	cmd := &cli.Command{Name: "aliyun", Run: func(*cli.Context, []string) error { return errors.New("original failure") }}
	cmd.Execute(ctx, nil)
	require.Empty(t, out.String())
	require.Contains(t, stderr.String(), "ERROR: original failure")
	require.Contains(t, stderr.String(), "requires CLI >= 3.6.0; current CLI is 3.5.3")
	require.Equal(t, " \t3.6.0\n", local.MinCliVersion)
}

func TestPluginCompatibilityValidationAndLookupFailures(t *testing.T) {
	previous := cli.Version
	cli.Version = "3.5.3"
	t.Cleanup(func() { cli.Version = previous })
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	require.ErrorContains(t, CheckLocalPluginCliVersion(ctx, "example", nil), "plugin is nil")
	require.ErrorContains(t, CheckLocalPluginCliVersion(ctx, "", &LocalPlugin{Name: "example", MinCliVersion: "invalid"}), "example has an invalid minimum CLI version")
	home := t.TempDir()
	t.Cleanup(setTestHomeDir(t, home))
	root := filepath.Join(home, ".aliyun", "plugins")
	require.Error(t, CheckPluginCliVersion(ctx, "missing"))
	body, err := json.Marshal(LocalManifest{Plugins: map[string]LocalPlugin{
		"example": {Name: "example", Command: "demo", MinCliVersion: "3.4.0"},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), body, 0600))
	require.NoError(t, CheckPluginCliVersion(ctx, "example"))
	blocked := filepath.Join(home, "blocked-plugins")
	require.NoError(t, os.WriteFile(blocked, []byte("file"), 0600))
	t.Setenv(EnvPluginsDir, blocked)
	require.Error(t, CheckPluginCliVersion(ctx, "demo"))
}
