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
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/require"
)

func TestMCPProxyAuthenticationErrorOutput(t *testing.T) {
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	t.Cleanup(func() { cli.SetNoColorOverride(false) })
	t.Setenv("NO_COLOR", "1")
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
		for _, mode := range []string{"0", "1"} {
			t.Run(tc.name+"/ai="+mode, func(t *testing.T) {
				t.Setenv(aimode.EnvAIMode, mode)
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
				stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
				root := &cli.Command{Name: "aliyun"}
				config.AddFlags(root.Flags())
				openapi.AddFlags(root.Flags())
				openapi.NewCommando(stdout, profile).InitWithCommand(root)
				utils := &cli.Command{Name: "utils"}
				utils.AddSubCommand(NewMCPProxyCommand())
				root.AddSubCommand(utils)
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
				if mode == "0" {
					require.Equal(t, "ERROR: "+tc.message+"\n", stderr.String())
				} else {
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
				}
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
}

func TestMCPAuthenticationErrorConversion(t *testing.T) {
	err := newMCPAuthenticationError("profile configuration failed", "check_configuration", "Check the selected profile.")
	require.EqualError(t, err, "profile configuration failed")
	var structured cli.StructuredError
	require.False(t, errors.As(err, &structured))
	var agentErr *cli.AgentError
	require.False(t, errors.As(err, &agentErr))
	require.False(t, cli.IsAIRecoveryEligible(err))
	converted := err.(*mcpAuthenticationError).AIError()
	require.Equal(t, 1, converted.ExitCode())
	require.ErrorIs(t, converted, err)
	require.Equal(t, err.Error(), converted.Error())
	require.Equal(t, "check_configuration", converted.Envelope().Recovery.Action)
	require.Equal(t, "Check the selected profile.", converted.Envelope().Recovery.Hint)
}
