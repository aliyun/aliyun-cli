package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	"github.com/stretchr/testify/require"
)

func TestMCPAuthenticationErrorsRespectAIMode(t *testing.T) {
	cli.DisableExitCode()
	t.Cleanup(cli.EnableExitCode)
	t.Cleanup(func() { cli.SetNoColorOverride(false) })
	for _, tc := range []struct {
		name, env, integration string
		agent, saved, wantJSON bool
		flags                  []string
	}{
		{name: "ordinary"},
		{name: "saved", saved: true, wantJSON: true},
		{name: "environment", env: "1", wantJSON: true},
		{name: "environment off overrides saved", env: "0", saved: true},
		{name: "explicit on overrides environment", env: "0", flags: []string{"--cli-ai-mode"}, wantJSON: true},
		{name: "explicit off overrides environment and saved", env: "1", saved: true, flags: []string{"--no-cli-ai-mode"}},
		{name: "off wins after on", flags: []string{"--cli-ai-mode", "--no-cli-ai-mode"}},
		{name: "off wins before on", flags: []string{"--no-cli-ai-mode", "--cli-ai-mode"}},
		{name: "detected agent", agent: true, wantJSON: true},
		{name: "agent integration disabled", agent: true, integration: "disabled"},
		{name: "disabled integration preserves saved", agent: true, integration: "disabled", saved: true, wantJSON: true},
		{name: "environment off overrides agent", env: "0", agent: true},
		{name: "explicit off overrides agent", agent: true, flags: []string{"--no-cli-ai-mode"}},
		{name: "JSON output alone does not enable AI", flags: []string{"--cli-output", "json"}},
		{name: "JSON output does not override explicit off", env: "1", flags: []string{"--no-cli-ai-mode", "--cli-output", "json"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearAgentDetectionEnv(t)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HOMEDRIVE", "")
			t.Setenv("HOMEPATH", "")
			t.Setenv("USERPROFILE", home)
			t.Setenv("NO_COLOR", "1")
			t.Setenv(aimode.EnvAIMode, tc.env)
			t.Setenv(aimode.EnvAgentIntegration, tc.integration)
			if tc.agent {
				t.Setenv("CODEX_SHELL", "1")
			}
			if tc.saved {
				require.NoError(t, aimode.Save(filepath.Join(home, ".aliyun"), &aimode.AiConfig{Enabled: true}))
			}

			var stdout, stderr bytes.Buffer
			root := newRootCommand(config.NewProfile("default"), &stdout)
			ctx := newCommandContext(&stdout, &stderr)
			ctx.EnterCommand(root)
			args := append(append([]string(nil), tc.flags...), "utils", "mcp-proxy", "--skip-mcp-oauth", "invalid")
			root.Execute(ctx, args)

			require.Empty(t, stdout.String())
			if tc.wantJSON {
				require.JSONEq(t, `{
					"schemaVersion": "v1",
					"message": "--skip-mcp-oauth must be true or false",
					"recovery": {
						"action": "inspect_command_help",
						"hint": "Pass --skip-mcp-oauth without a value, or set it to true or false. Use `+"`aliyun utils mcp-proxy --help`"+` to inspect the command options."
					}
				}`, stderr.String())
			} else {
				require.Equal(t, "ERROR: --skip-mcp-oauth must be true or false\n", stderr.String())
			}
		})
	}
}
