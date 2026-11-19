package updatecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/cli/upgrade"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestExplicitCheckTextAndJSONPreservePartialResults(t *testing.T) {
	previous := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	for _, cached := range []bool{false, true} {
		for _, useJSON := range []bool{false, true} {
			t.Run(fmtCase(cached, useJSON), func(t *testing.T) {
				s := testService(t)
				s.CLI = func(context.Context, string) (upgrade.UpdateResult, error) {
					return upgrade.UpdateResult{Current: "3.5.3", Latest: "3.9.0", Target: "3.6.0", Status: "update_available", UpdateAvailable: true, Command: "aliyun upgrade"}, nil
				}
				refreshErr := errors.New("plugin index refresh failed")
				s.Plugins = func(context.Context, string, plugin.UpdateScope) (plugin.UpdateReport, error) {
					r := plugin.UpdateReport{Status: "complete", Fresh: true, Plugins: []plugin.UpdateResult{{
						Name: "example", Current: "1.0.0", Latest: "2.0.0", Status: "update_available", UpdateAvailable: true,
						Command: "aliyun plugin update --name example", InstalledMinCliVersion: "3.0.0", MinCliVersion: "3.8.0", RequiresCLIUpgrade: true,
					}}}
					if cached {
						r.Status, r.Fresh = "cached", false
						return r, refreshErr
					}
					return r, nil
				}
				var out, stderr bytes.Buffer
				ctx := cli.NewCommandContext(&out, &stderr)
				ctx.EnterCommand(upgrade.NewUpgradeCommand())
				ctx.Flags().Add(&cli.Flag{Name: cli.CLIOutputFlagName, AssignedMode: cli.AssignedOnce})
				ctx.Flags().Get(cli.CLIOutputFlagName).SetAssigned(useJSON)
				ctx.Flags().Get(cli.CLIOutputFlagName).SetValue("json")
				err := s.runCheck(ctx)
				if cached {
					require.ErrorIs(t, err, refreshErr)
				} else {
					require.NoError(t, err)
				}
				require.Empty(t, stderr.String()) // The command framework renders returned errors.
				require.NoFileExists(t, s.path(".reminders.json"))
				if useJSON {
					var report Report
					require.NoError(t, json.Unmarshal(out.Bytes(), &report))
					require.Contains(t, out.String(), "\n  \"schema_version\": 1,")
					require.Equal(t, "3.6.0", report.CLI.Target)
					require.Len(t, report.Plugins.Plugins, 1)
					require.True(t, report.Plugins.Plugins[0].RequiresCLIUpgrade)
					if cached {
						require.Equal(t, "partial", report.Status)
						require.Equal(t, "cached", report.Plugins.Status)
					} else {
						require.Equal(t, "complete", report.Status)
					}
					return
				}
				require.False(t, json.Valid(out.Bytes()))
				for _, want := range []string{"Current version: 3.5.3", "Latest version: 3.9.0", "Upgrade target: 3.6.0", "Upgrade again", "Current version: 1.0.0; latest version: 2.0.0", "requires CLI >= 3.8.0", "Upgrade the CLI first", "aliyun plugin update --name example"} {
					require.Contains(t, out.String(), want)
				}
				if cached {
					require.Contains(t, out.String(), "Incomplete; some checks failed")
					require.Contains(t, out.String(), "Using verified cache; remote refresh failed")
				} else {
					require.Contains(t, out.String(), "Update check: Complete")
				}
			})
		}
	}
}

func fmtCase(cached, json bool) string {
	name := "complete"
	if cached {
		name = "cached"
	}
	if json {
		return name + "/json"
	}
	return name + "/text"
}

func TestCheckSummaryDistinguishesEmptyFailedAndUnsupportedPlugins(t *testing.T) {
	previous := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	for _, tc := range []struct {
		name, status, want string
		plugins            []plugin.UpdateResult
	}{
		{name: "empty", status: "complete", want: "No plugins installed."},
		{name: "failed", status: "failed", want: "Plugin update check failed."},
		{name: "unsupported", status: "complete", want: "Latest release does not support this platform", plugins: []plugin.UpdateResult{{Name: "other", Current: "1.0.0", Latest: "2.0.0", Status: "unsupported_platform"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			ctx := cli.NewCommandContext(&out, io.Discard)
			r := Report{Status: "partial", CLI: upgrade.UpdateResult{Status: "failed"}, Plugins: plugin.UpdateReport{Status: tc.status, Plugins: tc.plugins}}
			require.NoError(t, writeCheckReport(ctx, r))
			require.Contains(t, out.String(), tc.want)
			require.NotContains(t, out.String(), "Upgrade command:")
			if tc.status != "complete" {
				require.NotContains(t, out.String(), "No plugins installed.")
			}
		})
	}
}

func TestCheckSummaryChinese(t *testing.T) {
	previous := i18n.GetLanguage()
	i18n.SetLanguage("zh")
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	r := Report{Status: "complete", CLI: upgrade.UpdateResult{Current: "3.5.3", Latest: "3.5.3", Status: "up_to_date"}, Plugins: plugin.UpdateReport{Status: "complete"}}
	require.NoError(t, writeCheckReport(ctx, r))
	require.Contains(t, out.String(), "更新检查：完成")
	require.Contains(t, out.String(), "当前版本：3.5.3")
	require.Contains(t, out.String(), "已是最新版本")
	require.Contains(t, out.String(), "未安装插件。")
}
