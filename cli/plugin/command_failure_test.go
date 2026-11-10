package plugin

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestPluginCommandsReportStorageFailures(t *testing.T) {
	for _, name := range []string{"list", "list-remote", "search", "show", "install", "install-all", "install-custom", "uninstall", "update"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "not-a-directory")
			require.NoError(t, os.WriteFile(root, []byte("preserve"), 0600))
			t.Setenv(EnvPluginsDir, root)
			ctx := newTestContext()
			command := NewPluginCommand().GetSubCommand(name)
			ctx.EnterCommand(command)
			if flag := ctx.Flags().Get("name"); flag != nil {
				flag.SetValue("demo")
				flag.SetAssigned(true)
			}
			require.Error(t, command.Run(ctx, []string{"demo"}))
			data, err := os.ReadFile(root)
			require.NoError(t, err)
			require.Equal(t, "preserve", string(data))
		})
	}
}

func TestPluginSourceOverrideRejectsInvalidScheme(t *testing.T) {
	t.Setenv(EnvPluginsDir, t.TempDir())
	ctx := newTestContext()
	command := newListRemoteCommand()
	ctx.EnterCommand(command)
	flag := ctx.Flags().Get("source-base")
	flag.SetValue("ftp://mirror.example/plugins")
	flag.SetAssigned(true)
	require.ErrorContains(t, command.Run(ctx, nil), "plugin URL must use https://")
}

func TestHiddenInstallCommandsPropagateIndexErrors(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = managerBoundaryTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("trust offline")
	})
	for _, command := range []*cli.Command{newInstallAllCommand(), newInstallCustomCommand()} {
		t.Run(command.Name, func(t *testing.T) {
			t.Setenv(EnvPluginsDir, t.TempDir())
			ctx := newTestContext()
			ctx.EnterCommand(command)
			ctx.Flags().Get("enable-pre").SetAssigned(true)
			require.ErrorContains(t, command.Run(ctx, nil), "failed to verify plugin index")
		})
	}
}

func TestUninstallCommandRejectsMissingAndUnknownNames(t *testing.T) {
	t.Setenv(EnvPluginsDir, t.TempDir())
	ctx := newTestContext()
	command := newUninstallCommand()
	ctx.EnterCommand(command)
	require.ErrorContains(t, command.Run(ctx, nil), "plugin name is required")
	flag := ctx.Flags().Get("name")
	flag.SetValue("not-installed")
	flag.SetAssigned(true)
	require.ErrorContains(t, command.Run(ctx, nil), "not installed")
}
