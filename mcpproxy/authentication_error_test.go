package mcpproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

func TestMCPProxyAuthenticationErrorOutput(t *testing.T) {
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	for _, tc := range []struct {
		name, message, action, hint string
		mode                        config.AuthenticateMode
		options                     []string
		missing, invalid, exchange  bool
	}{
		{name: "missing profile", missing: true, message: "profile configuration failed", action: "check_configuration", hint: "aliyun configure list"},
		{name: "invalid profile", invalid: true, message: "profile configuration failed", action: "check_configuration", hint: "aliyun configure list"},
		{name: "unknown mode", mode: "unknown-sensitive-mode", message: "profile configuration failed", action: "check_configuration", hint: "aliyun configure list"},
		{name: "BearerToken", mode: config.BearerToken, message: `profile mode "BearerToken" is not supported for MCP credential authentication`, action: "check_configuration", hint: "--profile"},
		{name: "Anonymous", mode: config.Anonymous, message: `profile mode "Anonymous" is not supported for MCP credential authentication`, action: "check_configuration", hint: "--profile"},
		{name: "invalid skip value", options: []string{"--skip-mcp-oauth", "invalid"}, message: "--skip-mcp-oauth must be true or false", action: "inspect_command_help", hint: "true or false"},
		{name: "OAuth application conflict", options: []string{"--skip-mcp-oauth", "--oauth-app-name", "test-app"}, message: "--oauth-app-name and --scope apply to MCP OAuth; omit --skip-mcp-oauth to use that flow", action: "fix_option_combination", hint: "Remove --oauth-app-name and --scope"},
		{name: "OAuth scope conflict", options: []string{"--skip-mcp-oauth", "--scope", "custom-scope"}, message: "--oauth-app-name and --scope apply to MCP OAuth; omit --skip-mcp-oauth to use that flow", action: "fix_option_combination", hint: "Remove --oauth-app-name and --scope"},
		{name: "token exchange failure", exchange: true, message: "failed to authenticate MCP with CLI credentials: GenerateAccessToken failed; verify CLI credentials, AliyunOpenAPIMCPServerStaticCredentialAccess permission, and the account's MCP static-credential pre-check", action: "check_mcp_authentication", hint: "AliyunOpenAPIMCPServerStaticCredentialAccess"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixtureCtx := newMCPCredentialTestContext(t)
			profile := mcpTestAKProfile("selected", "test-key")
			if tc.mode != "" {
				profile.Mode = tc.mode
			}
			profile.BearerTokenValue = "sensitive-bearer"
			if tc.invalid {
				profile.AccessKeySecret = ""
			}
			writeMCPCredentialTestConfig(t, fixtureCtx, mcpTestAKProfile("default", "fallback-key"), profile)
			forbidMCPCommandTestNetwork(t)
			calls := 0
			if tc.exchange {
				http.DefaultTransport = credentialRoundTripper(func(req *http.Request) (*http.Response, error) {
					calls++
					require.Equal(t, "GenerateAccessToken", req.URL.Query().Get("Action"))
					return credentialResponse(http.StatusForbidden, `{"Code":"Forbidden","Message":"sensitive-response"}`), nil
				})
			}
			root := &cli.Command{Name: "aliyun"}
			config.AddFlags(root.Flags())
			utils := &cli.Command{Name: "utils"}
			utils.AddSubCommand(NewMCPProxyCommand())
			root.AddSubCommand(utils)
			stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
			ctx := cli.NewCommandContext(stdout, stderr)
			ctx.EnterCommand(root)
			profileName := "selected"
			if tc.missing {
				profileName = "missing"
			}
			args := []string{"utils", "mcp-proxy", "--profile", profileName}
			if tc.options == nil {
				args = append(args, "--skip-mcp-oauth")
			} else {
				args = append(args, tc.options...)
			}

			root.Execute(ctx, args)

			require.Empty(t, stdout.String())
			var envelope map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope), "%s", stderr.String())
			require.Len(t, envelope, 3)
			require.JSONEq(t, `"v1"`, string(envelope["schemaVersion"]))
			var message string
			require.NoError(t, json.Unmarshal(envelope["message"], &message))
			require.Equal(t, tc.message, message)
			var recovery map[string]string
			require.NoError(t, json.Unmarshal(envelope["recovery"], &recovery))
			require.Len(t, recovery, 2)
			require.Equal(t, tc.action, recovery["action"])
			require.NotEmpty(t, strings.TrimSpace(recovery["hint"]))
			require.Contains(t, recovery["hint"], tc.hint)
			for _, secret := range []string{"test-secret", "test-key", "fallback-key", "sensitive-bearer", "sensitive-response", "unknown-sensitive-mode"} {
				require.NotContains(t, stderr.String(), secret)
			}
			if tc.exchange {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}

type mcpAuthenticationErrorWriter struct{ err error }

func (w mcpAuthenticationErrorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestMCPAuthenticationErrorRenderingFailure(t *testing.T) {
	err := newMCPAuthenticationError("profile configuration failed", "check_configuration", "Check the selected profile.")
	require.EqualError(t, err, "profile configuration failed")
	var structured cli.StructuredError
	require.ErrorAs(t, err, &structured)
	require.Equal(t, 1, structured.ExitCode())
	writeErr := errors.New("stderr write failed")
	require.ErrorIs(t, structured.RenderError(mcpAuthenticationErrorWriter{err: writeErr}), writeErr)
}
