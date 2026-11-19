package runtimehost

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/require"
)

func TestHostLoadsCredentialAndWireSettingsFromExplicitProfile(t *testing.T) {
	require.NoError(t, os.Unsetenv(config.EnvSignatureVersion))
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"test","profiles":[{"name":"test","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-shanghai","read_timeout":3,"connect_timeout":4,"retry_count":2,"endpoint_type":"internal","endpoint":"profile.local:8080","language":"zh","signature_version":"v4"}]}`), 0600))
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)
	config.SkipSecureVerify(ctx.Flags()).SetAssigned(true)
	host := &cliHost{ctx: ctx}
	require.Equal(t, "cn-shanghai", host.Region())
	settings := host.Settings()
	require.Equal(t, 3*time.Second, settings.ReadTimeout)
	require.Equal(t, 4*time.Second, settings.ConnectTimeout)
	require.Equal(t, 2, settings.RetryCount)
	require.Equal(t, "internal", settings.EndpointType)
	require.Equal(t, "profile.local:8080", settings.Endpoint)
	require.True(t, settings.SkipSecureVerify)
	require.Equal(t, config.SignatureVersionV4, settings.SignatureVersion)
	credential, err := host.Credential()
	require.NoError(t, err)
	model, err := credential.GetCredential()
	require.NoError(t, err)
	require.Equal(t, "id", *model.AccessKeyId)
	previousLanguage := i18n.GetLanguage()
	t.Cleanup(func() { i18n.SetLanguage(previousLanguage) })
	previousDispatch := engineDispatch
	t.Cleanup(func() { engineDispatch = previousDispatch })
	engineDispatch = func(request engine.Request) error {
		require.Equal(t, "zh", request.Lang)
		require.Equal(t, []string{"demo", "version"}, request.Args)
		return nil
	}
	require.NoError(t, Dispatch(ctx, []string{"demo", "version"}))
	require.Equal(t, "zh", i18n.GetLanguage())
}

func TestHostSettingsEndpointEnvAndFlagPrecedence(t *testing.T) {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"test","profiles":[{"name":"test","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-shanghai"}]}`), 0600))
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

	t.Setenv("ALIBABA_CLOUD_ENDPOINT", "env.local:9000")
	require.Equal(t, "env.local:9000", (&cliHost{ctx: ctx}).Settings().Endpoint)

	config.EndpointFlag(ctx.Flags()).SetAssigned(true)
	config.EndpointFlag(ctx.Flags()).SetValue("flag.local:9001")
	require.Equal(t, "flag.local:9001", (&cliHost{ctx: ctx}).Settings().Endpoint)
}

// TestDispatchKebabUsesProfileEndpoint covers the reported scenario end to end:
// a kebab-case metadata command resolves its request endpoint from the profile
// (including ALIBABA_CLOUD_ENDPOINT), and an explicit --endpoint still wins.
func TestDispatchKebabUsesProfileEndpoint(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ENDPOINT", "local.test:18080")
	var stdout, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&stdout, &stderr)
	config.AddFlags(ctx.Flags())
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"test","profiles":[{"name":"test","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-hangzhou"}]}`), 0600))
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(path)

	type dryRun struct {
		Product  string `json:"product"`
		Endpoint string `json:"endpoint"`
	}
	resolve := func() dryRun {
		t.Helper()
		var out dryRun
		require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &out), stdout.String())
		return out
	}

	require.NoError(t, Dispatch(ctx, []string{"ecs", "describe-regions", "--cli-dry-run-json"}))
	require.Equal(t, "ecs", resolve().Product)
	require.Equal(t, "local.test:18080", resolve().Endpoint)

	stdout.Reset()
	require.NoError(t, Dispatch(ctx, []string{"ecs", "describe-regions", "--endpoint", "explicit.test:18081", "--cli-dry-run-json"}))
	require.Equal(t, "explicit.test:18081", resolve().Endpoint)
}

func TestHelpLanguageAndSafetyReadOnlyBoundaries(t *testing.T) {
	require.False(t, flagAssigned(nil, "help"))
	require.NoError(t, checkSafetyPolicy(nil, nil))
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	ctx.Flags().Add(&cli.Flag{Name: "help"})
	for _, args := range [][]string{{"demo"}, {"demo", "version"}, {"demo", "delete", "-h"}, {"demo", "delete", "--help"}} {
		require.NoError(t, checkSafetyPolicy(ctx, args))
	}
	ctx.Flags().Get("help").SetAssigned(true)
	require.NoError(t, checkSafetyPolicy(ctx, []string{"demo", "delete"}))
	for _, args := range [][]string{{"demo", "--language", " zh "}, {"demo", "--language=zh"}} {
		require.Equal(t, "zh", runtimeHelpRequest(ctx, args, []string{"demo"}).Lang)
	}
	require.Equal(t, 1, nextRuntimeHelpPositional([]string{"--", "demo"}, 0))
	require.NoError(t, DispatchPluginHelp(ctx, []string{"unknown-product", "UnknownCommand"}))
	for _, mode := range []cli.AssignedMode{cli.AssignedRepeatable, cli.AssignedMode(255)} {
		require.Panics(t, func() { externalFlagMode(&cli.Flag{Name: "invalid", AssignedMode: mode}) })
	}
}
