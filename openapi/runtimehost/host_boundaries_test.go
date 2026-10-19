package runtimehost

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-openapi-runtime/engine"
	"github.com/stretchr/testify/require"
)

func TestHostLoadsCredentialAndWireSettingsFromExplicitProfile(t *testing.T) {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"current":"test","profiles":[{"name":"test","mode":"AK","access_key_id":"id","access_key_secret":"secret","region_id":"cn-shanghai","read_timeout":3,"connect_timeout":4,"retry_count":2,"endpoint_type":"internal","language":"zh"}]}`), 0600))
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
	require.True(t, settings.SkipSecureVerify)
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
