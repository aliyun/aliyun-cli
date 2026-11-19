package upgrade

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestReadOnlyCommandOutputFormats(t *testing.T) {
	previous, previousLanguage := httpClient, i18n.GetLanguage()
	t.Cleanup(func() { httpClient = previous; i18n.SetLanguage(previousLanguage) })
	i18n.SetLanguage("en")
	httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := "3.9.0"
		if req.URL.Path == "/versions" {
			body = `{"format":"aliyun-cli-versions v1","versions":[{"version":"3.9.0","public_keys":["1"]}]}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	for _, useJSON := range []bool{false, true} {
		var out, stderr bytes.Buffer
		ctx := cli.NewCommandContext(&out, &stderr)
		root := &cli.Command{Name: "aliyun"}
		root.Flags().Add(&cli.Flag{Name: cli.CLIOutputFlagName, AssignedMode: cli.AssignedOnce, Persistent: true})
		ctx.EnterCommand(root)
		cmd := NewUpgradeCommand()
		ctx.EnterCommand(cmd)
		ctx.Flags().Get("check").SetAssigned(true)
		ctx.Flags().Get(cli.CLIOutputFlagName).SetAssigned(useJSON)
		ctx.Flags().Get(cli.CLIOutputFlagName).SetValue("json")
		require.NoError(t, cmd.Run(ctx, nil))
		require.Empty(t, stderr.String())
		if useJSON {
			var result UpdateResult
			require.NoError(t, json.Unmarshal(out.Bytes(), &result))
			require.Equal(t, "3.9.0", result.Latest)
			require.Contains(t, out.String(), "\n  \"current\":")
		} else {
			require.Contains(t, out.String(), "Latest version: 3.9.0")
			require.False(t, json.Valid(out.Bytes()))
		}
	}
}

func TestInvalidCheckOutputIsRejectedBeforeChecking(t *testing.T) {
	var out bytes.Buffer
	ctx := cli.NewCommandContext(&out, io.Discard)
	cmd := NewUpgradeCommand(func(*cli.Context) error {
		t.Fatal("an invalid output format must not start a check")
		return nil
	})
	ctx.EnterCommand(cmd)
	ctx.Flags().Add(&cli.Flag{Name: cli.CLIOutputFlagName, AssignedMode: cli.AssignedOnce})
	ctx.Flags().Get(cli.CLIOutputFlagName).SetAssigned(true)
	ctx.Flags().Get(cli.CLIOutputFlagName).SetValue("yaml")
	ctx.Flags().Get("check").SetAssigned(true)
	err := cmd.Run(ctx, nil)
	var optionErr *cli.HelpOptionError
	require.ErrorAs(t, err, &optionErr)
	require.Equal(t, cli.HelpOptionInvalidOutput, optionErr.Code)
	require.Nil(t, cmd.Flags().Get("json"))
	require.Empty(t, out.String())
}

func TestCheckSummaryDescribesUnavailableAndStagedUpgrades(t *testing.T) {
	previous := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	for _, tc := range []struct{ status, want string }{
		{"failed", "Check failed"},
		{"up_to_date", "Up to date"},
		{"development_build", "version comparison skipped"},
		{"no_compatible_release", "No compatible upgrade release"},
	} {
		var out bytes.Buffer
		require.NoError(t, WriteCheckSummary(&out, UpdateResult{Status: tc.status}))
		require.Contains(t, out.String(), tc.want)
		require.NotContains(t, out.String(), "Upgrade command:")
	}
	var out bytes.Buffer
	require.NoError(t, WriteCheckSummary(&out, UpdateResult{Current: "3.5.3", Latest: "3.9.0", Target: "3.6.0",
		Status: "update_available", UpdateAvailable: true, Command: "aliyun upgrade"}))
	require.Contains(t, out.String(), "Upgrade target: 3.6.0")
	require.Contains(t, out.String(), "Upgrade command: aliyun upgrade")
	require.Contains(t, out.String(), "Upgrade again")
}
