package telemetry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/stretchr/testify/require"
)

func telemetryPluginFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := t.TempDir()
	t.Setenv(plugin.EnvPluginsDir, dir)
	manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-demo": {Name: "aliyun-cli-demo", Command: "demo", Version: "1.2.3", Type: plugin.PluginTypeMeta, Inner: true},
	}}
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600))
	return dir
}

func TestExecutionFlushPersistsOneEvent(t *testing.T) {
	telemetryPluginFixture(t)
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	for _, test := range []struct {
		name                   string
		args                   []string
		end                    func()
		result, code, pipeline string
	}{
		{"success", []string{"demo", "list", "--region-id", "private-value"}, func() { End(nil) }, "Success", "", PipelineInner},
		{"local_error", []string{"version", "--bad"}, func() { End(cli.NewInvalidFlagError("--bad", nil)) }, "UserFault", LocalErrorInvalidFlag, PipelinePublic},
		{"plugin_failure", []string{"plugin", "install", "--name", "unvalidated-name"}, func() { EndWithExitCode(7) }, "Failure", "", PipelinePublic},
		{"plugin_success", []string{"plugin", "update", "--name", "demo"}, func() { HookEnd(nil, nil, nil) }, "Success", "", PipelinePublic},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := collectionPanicConfigDir(t)
			args := append([]string(nil), test.args...)
			Begin(BeginInput{ConfigDir: dir, CommandToken: args[0], CommandArgs: args, AuthMode: "ak"})
			require.NotNil(t, activeExecution)
			// The caller may reuse its argument slice after Begin.
			args[0] = "mutated"
			activeExecution.start = time.Now().Add(-time.Second)
			test.end()
			End(nil)
			EndWithExitCode(0)
			entries, err := os.ReadDir(GetCacheDir(dir, test.pipeline))
			require.NoError(t, err)
			require.Len(t, entries, 1, "multiple end callbacks must not duplicate the event")
			data, err := os.ReadFile(filepath.Join(GetCacheDir(dir, test.pipeline), entries[0].Name()))
			require.NoError(t, err)
			var event Event
			require.NoError(t, json.Unmarshal(data, &event))
			require.Equal(t, test.result, event.Result)
			require.Equal(t, test.code, event.LocalErrorCode)
			require.Equal(t, test.pipeline, event.Pipeline)
			require.Equal(t, "AK", event.AuthMode)
			require.GreaterOrEqual(t, event.DurationMs, int64(1000))
			require.Equal(t, "c4160461-19cf-4ec9-bc28-b2430c1ad90e", event.InstanceID)
			require.NotContains(t, string(data), "private-value")
			require.NotContains(t, event.Command, "mutated")
			if test.name == "plugin_failure" {
				require.NotNil(t, event.PluginAction)
				require.Empty(t, event.PluginAction.PluginNames)
				require.NotContains(t, string(data), "unvalidated-name")
			}
			if test.name == "plugin_success" {
				require.Equal(t, []string{"aliyun-cli-demo"}, event.PluginAction.PluginNames)
				require.Equal(t, "1.2.3", event.PluginAction.ToVersion)
			}
		})
	}
}

func TestExecutionSkipsInvalidOrChangedStorage(t *testing.T) {
	t.Setenv(TelemetryDisableEnv, "0")
	t.Cleanup(resetExecutionState)
	for _, scenario := range []string{"bad_config_at_begin", "bad_config_at_end", "disabled_at_end", "invalid_id", "blocked_cache"} {
		t.Run(scenario, func(t *testing.T) {
			dir := collectionPanicConfigDir(t)
			if scenario == "bad_config_at_begin" {
				require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte("{"), 0600))
			}
			Begin(BeginInput{ConfigDir: dir, CommandToken: "version", CommandArgs: []string{"version"}})
			if scenario == "bad_config_at_begin" {
				require.Nil(t, activeExecution)
			} else {
				require.NotNil(t, activeExecution)
			}
			switch scenario {
			case "bad_config_at_end":
				require.NoError(t, os.WriteFile(GetConfigFilePath(dir), []byte("{"), 0600))
			case "disabled_at_end":
				require.NoError(t, Save(dir, &Config{Enabled: false}))
			case "invalid_id":
				require.NoError(t, os.WriteFile(GetInstanceIDFilePath(dir), []byte("invalid"), 0600))
			case "blocked_cache":
				require.NoError(t, os.WriteFile(filepath.Join(dir, CacheDirName), []byte("keep"), 0600))
			}
			End(nil)
			EndWithExitCode(0)
			require.NoDirExists(t, GetPublicCacheDir(dir))
			if scenario == "blocked_cache" {
				data, err := os.ReadFile(filepath.Join(dir, CacheDirName))
				require.NoError(t, err)
				require.Equal(t, "keep", string(data))
			}
		})
	}
}

func TestAuthenticationModeAllowlistAndState(t *testing.T) {
	t.Cleanup(resetExecutionState)
	resetExecutionState()
	SetEffectiveAuthenticationMode("AK")
	SetAuthenticationMode("AK", AuthModeSourceExplicit)
	require.Nil(t, activeExecution, "setters must not start collection")
	require.Zero(t, commandExecutionDurationMs(nil))
	for _, mode := range []string{"AK", "StsToken", "RamRoleArn", "EcsRamRole", "RsaKeyPair", "RamRoleArnWithRoleName", "ChainableRamRoleArn", "External", "CredentialsURI", "OIDC", "CloudSSO", "OAuth", "BearerToken", "Anonymous"} {
		t.Run(mode, func(t *testing.T) {
			beginCommandExecution("", Scope{}, nil, false)
			SetEffectiveAuthenticationMode(" " + strings.ToLower(mode) + " ")
			require.Equal(t, mode, activeExecution.authMode)
			for _, source := range []string{AuthModeSourceExplicit, AuthModeSourceDefault, AuthModeSourceExisting} {
				SetAuthenticationMode(mode, source)
				require.Equal(t, mode, activeExecution.authMode)
				require.Equal(t, source, activeExecution.authModeSource)
			}
			SetAuthenticationMode("customer-secret", AuthModeSourceExplicit)
			SetAuthenticationMode("AK", "customer-secret")
			require.Equal(t, mode, activeExecution.authMode)
			require.Equal(t, AuthModeSourceExisting, activeExecution.authModeSource)
		})
	}
}

func TestPluginManagementResolvesInstalledMetadata(t *testing.T) {
	telemetryPluginFixture(t)
	event := DetectPluginManagement([]string{"plugin", "update", "--name", "demo", "--version", "2.0.0"})
	require.Equal(t, &PluginManagementEvent{Operation: "update", PluginNames: []string{"aliyun-cli-demo"}, FromVersion: "1.2.3", ToVersion: "2.0.0", PluginType: plugin.PluginTypeMeta, Inner: boolPointer(true), SourceType: "catalog"}, event)
	CompletePluginManagement(event)
	require.Equal(t, "1.2.3", event.ToVersion, "record the installed version after success")
	inner := DetectInnerScope("demo")
	require.True(t, inner.Active)
	require.Equal(t, PipelineInner, inner.Pipeline)
	ok, scope := ShouldCollectInner("demo", DefaultConfig())
	require.True(t, ok)
	require.Equal(t, inner, scope)
	require.Empty(t, DetectInnerScope(""))
	require.Empty(t, DetectInnerScope("not-installed"))
	ok, _ = ShouldCollectInner("demo", &Config{Enabled: false})
	require.False(t, ok)

	require.Empty(t, packageSourceType(" "))
	require.Equal(t, strings.Repeat("a", 64), sanitizeVersion(strings.Repeat("a", 70)))
	require.Empty(t, sanitizeVersion("private/value"))
	require.Equal(t, []string{"demo"}, pluginFlagValues([]string{"--names=demo,demo"}, "names"))
	require.Equal(t, "", firstFlagValue([]string{"--version", "--name=demo"}, "version"))
}

func boolPointer(value bool) *bool { return &value }

func TestScopeFallsBackWhenPluginStorageFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(path, []byte("keep"), 0600))
	t.Setenv(plugin.EnvPluginsDir, path)
	t.Setenv(TelemetryDisableEnv, "0")
	ok, scope := SelectScope("demo", DefaultConfig())
	require.True(t, ok)
	require.Equal(t, Scope{Active: true, Pipeline: PipelinePublic, CommandToken: "demo"}, scope)
	require.Empty(t, DetectInnerScope("demo"))
}

type unknownRecoveryError struct{}

func (unknownRecoveryError) Error() string          { return "private error text" }
func (unknownRecoveryError) AIRecoveryEligible()    {}
func (unknownRecoveryError) LocalErrorCode() string { return "untrusted-code" }

func TestEventRejectsUntrustedErrorMetadata(t *testing.T) {
	event, err := BuildEvent(BuildInput{Err: unknownRecoveryError{}})
	require.NoError(t, err)
	require.Equal(t, LocalErrorOther, event.LocalErrorCode)
	require.Equal(t, "UserFault", event.Result)
	for _, code := range []string{"private value", strings.Repeat("a", 129), ""} {
		require.Empty(t, safeAPIErrorCode(code))
	}
	for _, status := range []int{-1, 99, 600} {
		require.Zero(t, safeHTTPStatus(status))
	}
	require.Equal(t, "Failure", classifyResult(errors.New("private error"), nil))
}
