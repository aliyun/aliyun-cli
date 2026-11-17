//go:build linux || darwin || windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
	"github.com/stretchr/testify/require"
)

func TestMainTelemetrySkipsSensitiveToolParameterValues(t *testing.T) {
	const first = "--telemetry-private-credential-one"
	const second = "--telemetry-private-credential-two"
	const third = "--telemetry-private-credential-three"
	for _, test := range []struct {
		name      string
		extension bool
		leading   []string
		args      []string
		want      []string
	}{
		{"extension_sts_token", true, nil, []string{"--sts-token", first}, []string{"--sts-token"}},
		{"extension_access_key_id", true, nil, []string{"--access-key-id", first}, []string{"--access-key-id"}},
		{"extension_access_key_secret", true, nil, []string{"--access-key-secret", first}, []string{"--access-key-secret"}},
		{"extension_host_credentials_together", true, nil, []string{"--sts-token", first, "--access-key-id", second, "--access-key-secret", third, "--verbose"}, []string{"--sts-token", "--access-key-id", "--access-key-secret", "--verbose"}},
		{"extension_password", true, nil, []string{"--password", first, "--verbose"}, []string{"--password", "--verbose"}},
		{"extension_inline_host_token", true, nil, []string{"--sts-token=" + first, "--verbose"}, []string{"--sts-token", "--verbose"}},
		{"extension_inline_password_colon", true, nil, []string{"--password:" + first, "--verbose"}, []string{"--password", "--verbose"}},
		{"extension_empty_inline_password", true, nil, []string{"--password=", first, "--verbose"}, []string{"--password", "--verbose"}},
		{"extension_sensitive_before_delimiter", true, nil, []string{"--password", "--", first}, []string{"--password"}},
		{"go_password_case_and_repetition", false, nil, []string{"--Password", first, "--Password", second, "--verbose"}, []string{"--Password", "--Password", "--verbose"}},
		{"go_camel_case_credentials", false, nil, []string{"--AccessKeyId", first, "--AccessKeySecret", second, "--SecurityToken", third, "--verbose"}, []string{"--AccessKeyId", "--AccessKeySecret", "--SecurityToken", "--verbose"}},
		{"go_tool_specific_credentials", false, nil, []string{"--api_key", first, "--client-secret", second, "--database-password", third, "--verbose"}, []string{"--api_key", "--client-secret", "--database-password", "--verbose"}},
		{"go_inline_password_equals", false, nil, []string{"--Password=" + first, "--verbose"}, []string{"--Password", "--verbose"}},
		{"go_inline_password_colon", false, nil, []string{"--Password:" + first, "--verbose"}, []string{"--Password", "--verbose"}},
		{"go_inline_host_prefix", false, []string{"--access-key-id=" + first, "--access-key-secret=" + second}, []string{"--Password", third, "--verbose"}, []string{"--access-key-id", "--access-key-secret", "--Password", "--verbose"}},
		{"go_runtime_auth", false, nil, []string{"--auth", first, "--verbose"}, []string{"--auth", "--verbose"}},
		{"go_runtime_bearer", false, nil, []string{"--bearer", first, "--verbose"}, []string{"--bearer", "--verbose"}},
		{"go_runtime_signature_header", false, nil, []string{"--x-acs-signature", first, "--verbose"}, []string{"--x-acs-signature", "--verbose"}},
		{"go_runtime_underscore_header", false, nil, []string{"--x_acs_signature", first, "--verbose"}, []string{"--x_acs_signature", "--verbose"}},
		{"extension_runtime_header_case", true, nil, []string{"--X-Acs-Signature", first, "--verbose"}, []string{"--X-Acs-Signature", "--verbose"}},
		{"go_runtime_custom_header", false, nil, []string{"--x-custom-auth", first, "--X_Custom_Auth", second, "--verbose"}, []string{"--x-custom-auth", "--X_Custom_Auth", "--verbose"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			home, dir := telemetryExecutionHome(t)
			var args, env []string
			wantType := telemetry.CommandTypePluginGo
			if test.extension {
				// The bridge consumes host credential options and builds the
				// child's credential environment. All values are synthetic and
				// the child only prints fixed fixture output, without cloud I/O.
				profile := `{"current":"default","profiles":[{"name":"default","mode":"StsToken","access_key_id":"fixture-id","access_key_secret":"fixture-secret","sts_token":"fixture-token","region_id":"cn-hangzhou","language":"en"}]}`
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(profile), 0600))
				args = []string{"bl", "__telemetry-extension-test"}
				env = []string{"ALIBABA_CLOUD_BL_EXEC_PATH=" + os.Args[0], "ALIYUN_TEST_EXTENSION_EXIT_CODE=0"}
				wantType = telemetry.CommandTypeCLIExt
			} else {
				pluginsDir := filepath.Join(dir, "plugins")
				require.NoError(t, os.Mkdir(pluginsDir, 0700))
				required := false
				manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
					"aliyun-cli-sensitivefixture": {
						Name: "aliyun-cli-sensitivefixture", Version: "1.2.3", Command: "sensitivefixture",
						Type: plugin.PluginTypeGo, CmdNames: []string{"function list"}, ProfileRequired: &required,
						Path: filepath.Join(pluginsDir, "missing-executable"),
					},
				}}
				data, err := json.Marshal(manifest)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600))
				args = []string{"sensitivefixture", "function", "list"}
				env = []string{plugin.EnvPluginsDir + "=" + pluginsDir}
			}
			if test.name == "go_runtime_custom_header" {
				env = append(env, "ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY=x-custom-auth")
			}
			args = append(append(append([]string(nil), test.leading...), args...), test.args...)
			baseline := runTelemetryPermissionCommand(t, home, true, args, env...)
			if test.extension {
				require.Zero(t, baseline.exitCode, baseline.stderr)
				require.Equal(t, "extension stdout\n", baseline.stdout)
				require.Equal(t, "extension stderr\n", baseline.stderr)
			} else {
				require.Greater(t, baseline.exitCode, 0)
				require.Contains(t, baseline.stderr, "plugin binary not found", "the real Go route must have been selected")
			}
			got := runTelemetryPermissionCommand(t, home, false, args, env...)
			require.Equal(t, baseline, got, "credential suppression must not affect tool execution")
			event := readTelemetryExecutionEvent(t, dir)
			require.Equal(t, wantType, event.CommandType)
			require.Equal(t, test.want, event.Params)
			data, err := json.Marshal(event)
			require.NoError(t, err)
			for _, private := range []string{first, second, third, "fixture-id", "fixture-secret", "fixture-token", home} {
				require.NotContains(t, string(data), private)
			}
		})
	}
}
