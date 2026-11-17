package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

var configurationDefaultLoadHook = hookLoadOrCreateConfiguration
var configurationDefaultSaveHook = hookSaveConfigurationWithContext
var configurationDefaultContextLoadHook = hookLoadConfigurationWithContext
var configurationDefaultDispatch = doConfigureProxy

func isolatedConfigurationLifecycle(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	oldHome, oldLoad, oldSave, oldContextLoad := hookGetHomePath, hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext, hookLoadConfigurationWithContext
	t.Cleanup(func() {
		hookGetHomePath, hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext, hookLoadConfigurationWithContext = oldHome, oldLoad, oldSave, oldContextLoad
	})
	hookGetHomePath = func(func() string) func() string { return func() string { return home } }
	hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext, hookLoadConfigurationWithContext = configurationDefaultLoadHook, configurationDefaultSaveHook, configurationDefaultContextLoadHook
	return home
}

func lifecycleContext(out io.Writer) *cli.Context {
	ctx := cli.NewCommandContext(out, io.Discard)
	AddFlags(ctx.Flags())
	return ctx
}

func assignLifecycleFlag(t *testing.T, ctx *cli.Context, name, value string) {
	t.Helper()
	flag := ctx.Flags().Get(name)
	require.NotNil(t, flag)
	flag.SetValue(value)
	flag.SetAssigned(true)
}

func TestSetPersistsNewAndChainedProfiles(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	for _, tc := range []struct {
		name  string
		mode  AuthenticateMode
		extra map[string]string
	}{
		{"new-ak", AK, map[string]string{AccessKeyIdFlagName: "new-id", AccessKeySecretFlagName: "new-secret"}},
		{"chain", ChainableRamRoleArn, map[string]string{SourceProfileFlagName: "source", RamRoleArnFlagName: "acs:ram::123:role/demo", RoleSessionNameFlagName: "session", ExternalIdFlagName: "external", ExpiredSecondsFlagName: "900"}},
		{"oauth", OAuth, map[string]string{OAuthSiteTypeName: "INTL"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := lifecycleContext(io.Discard)
			assignLifecycleFlag(t, ctx, ProfileFlagName, tc.name)
			assignLifecycleFlag(t, ctx, ModeFlagName, string(tc.mode))
			assignLifecycleFlag(t, ctx, RegionFlagName, "cn-hangzhou")
			for name, value := range tc.extra {
				assignLifecycleFlag(t, ctx, name, value)
			}
			require.NoError(t, NewConfigureSetCommand().Run(ctx, nil))
			conf, err := LoadConfigurationFromFile(filepath.Join(GetConfigPath(), configFile))
			require.NoError(t, err)
			stored, ok := conf.GetProfile(tc.name)
			require.True(t, ok)
			require.Equal(t, tc.mode, stored.Mode)
			if tc.mode == ChainableRamRoleArn {
				require.Equal(t, "source", stored.SourceProfile)
				require.Equal(t, "external", stored.ExternalId)
				require.Equal(t, 900, stored.ExpiredSeconds)
			}
			if tc.mode == OAuth {
				require.Equal(t, "INTL", stored.OAuthSiteType)
			}
		})
	}
}

func TestSetLoadsExplicitConfigurationAndReportsSaveFailures(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	profile := NewProfile("custom")
	profile.Mode = ""
	profile.RegionId = "cn-hangzhou"
	profile.AccessKeyId, profile.AccessKeySecret = "id", "secret"
	path := filepath.Join(t.TempDir(), "custom.json")
	require.NoError(t, atomicWriteFile(path, []byte(`{"current":"custom","profiles":[{"name":"custom","mode":"","region_id":"cn-hangzhou","access_key_id":"id","access_key_secret":"secret"}]}`), 0600))
	ctx := lifecycleContext(io.Discard)
	assignLifecycleFlag(t, ctx, ProfileFlagName, "custom")
	assignLifecycleFlag(t, ctx, ConfigurePathFlagName, path)
	require.NoError(t, doConfigureSet(ctx))
	conf, err := LoadConfigurationFromFile(path)
	require.NoError(t, err)
	stored, _ := conf.GetProfile("custom")
	require.Equal(t, AK, stored.Mode)
	assignLifecycleFlag(t, ctx, ConfigurePathFlagName, filepath.Join(t.TempDir(), "missing.json"))
	require.ErrorContains(t, doConfigureSet(ctx), "load configuration file failed")
	ctx = lifecycleContext(io.Discard)
	assignLifecycleFlag(t, ctx, ProfileFlagName, "custom")
	hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
		return func(string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "custom", Profiles: []Profile{profile}}, nil
		}
	}
	cause := errors.New("disk full")
	hookSaveConfigurationWithContext = func(func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error {
		return func(*cli.Context, *Configuration) error { return cause }
	}
	require.ErrorContains(t, doConfigureSet(ctx), "save configuration failed disk full")
}

func TestListShowsIndirectCredentialsWithoutSecrets(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	conf := &Configuration{CurrentProfile: "chain"}
	for _, mode := range []AuthenticateMode{RamRoleArnWithEcs, ChainableRamRoleArn, External, CredentialsURI, OIDC, OAuth} {
		p := NewProfile(string(mode))
		p.Mode = mode
		p.RegionId = "cn-hangzhou"
		p.AccessKeyId = "private-access-key"
		p.SourceProfile = "source"
		p.RamRoleArn = "role-arn"
		p.ExternalId = "private-external-id"
		p.ProcessCommand = "credential-helper"
		p.CredentialsURI = "https://credentials.example.invalid"
		p.OIDCProviderARN = "private-provider-arn"
		p.OIDCTokenFile = "private-token-path"
		p.OAuthAccessToken = "private-access-token"
		p.OAuthSiteType = "CN"
		conf.Profiles = append(conf.Profiles, p)
	}
	require.NoError(t, SaveConfiguration(conf))
	var out bytes.Buffer
	require.NoError(t, NewConfigureListCommand().Run(lifecycleContext(&out), nil))
	for _, want := range []string{"arn:***key", "ChainableRamRoleArn:source:role-arn", "ProcessCommand:credential-helper", "CredentialsURI:https://credentials.example.invalid", "OIDC:***", "OAuth:", "@CN"} {
		require.Contains(t, out.String(), want)
	}
	for _, secret := range []string{"private-access-key", "private-external-id", "private-provider-arn", "private-token-path", "private-access-token"} {
		require.NotContains(t, out.String(), secret)
	}
}

func TestConfigurationMigrationAndReadErrors(t *testing.T) {
	home := isolatedConfigurationLifecycle(t)
	legacy := filepath.Join(home, ".aliyuncli")
	require.NoError(t, os.Mkdir(legacy, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "credentials"), []byte("[default]\naliyun_access_key_id=id\naliyun_access_key_secret=secret\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "configure"), []byte("[default]\nregion=cn-hangzhou\n"), 0600))
	conf, err := LoadOrCreateConfiguration(filepath.Join(t.TempDir(), "not-created.json"))
	require.NoError(t, err)
	profile, _ := conf.GetProfile("default")
	require.Equal(t, "cn-hangzhou", profile.RegionId)
	path := filepath.Join(GetConfigPath(), configFile)
	require.FileExists(t, path)
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Mkdir(path, 0700))
	_, err = LoadOrCreateConfiguration(filepath.Join(t.TempDir(), "not-created.json"))
	require.ErrorContains(t, err, "save failed")
	_, err = LoadOrCreateConfiguration(path)
	require.ErrorContains(t, err, "reading config")
	_, err = LoadConfigurationFromFile(path)
	require.ErrorContains(t, err, "reading config")
	ctx := lifecycleContext(io.Discard)
	assignLifecycleFlag(t, ctx, ConfigurePathFlagName, path)
	_, err = LoadProfileWithContext(ctx)
	require.Error(t, err)
}

func TestInteractiveConfigureDefaultsOAuthAndCustomPaths(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	oldInput, oldStart, oldExchange, oldDispatch := stdin, oauthStartOauthFlow, oauthExchangeFromOAuth, doConfigureProxy
	t.Cleanup(func() {
		stdin, oauthStartOauthFlow, oauthExchangeFromOAuth, doConfigureProxy = oldInput, oldStart, oldExchange, oldDispatch
	})
	doConfigureProxy = configurationDefaultDispatch
	profile := NewProfile("selected")
	profile.Mode = OAuth
	profile.RegionId = "cn-hangzhou"
	profile.OAuthSiteType = "CN"
	require.NoError(t, SaveConfiguration(&Configuration{CurrentProfile: "selected", Profiles: []Profile{profile}}))
	stop := errors.New("save stopped")
	hookSaveConfigurationWithContext = func(func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error {
		return func(_ *cli.Context, conf *Configuration) error {
			p, _ := conf.GetProfile("selected")
			require.Equal(t, OAuth, p.Mode)
			require.Equal(t, "id", p.AccessKeyId)
			return stop
		}
	}
	oauthStartOauthFlow = func(io.Writer, *Profile) error { return nil }
	oauthExchangeFromOAuth = func(_ io.Writer, p *Profile) error { p.AccessKeyId = "id"; return nil }
	var out bytes.Buffer
	ctx := lifecycleContext(&out)
	stdin = strings.NewReader("")
	require.ErrorIs(t, NewConfigureCommand().Run(ctx, nil), stop)
	require.Contains(t, out.String(), "OAuth configuration completed")
	require.ErrorIs(t, doConfigure(ctx, "", ""), stop)
	assignLifecycleFlag(t, ctx, ConfigurePathFlagName, filepath.Join(GetConfigPath(), configFile))
	require.ErrorIs(t, NewConfigureCommand().Run(ctx, nil), stop)
	oauthStartOauthFlow = func(io.Writer, *Profile) error { return stop }
	require.ErrorIs(t, doConfigure(ctx, "selected", string(OAuth)), stop)
	require.ErrorContains(t, doConfigure(ctx, "selected", "unsupported"), "unexcepted authenticate mode")
	assignLifecycleFlag(t, ctx, ConfigurePathFlagName, GetConfigPath())
	require.Error(t, doConfigure(ctx, "selected", string(AK)))
}

func TestConfigureDeleteUpdatesCurrentProfileAndGetReadsOAuthSite(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	for _, keepOther := range []bool{false, true} {
		conf := &Configuration{CurrentProfile: "selected", Profiles: []Profile{{Name: "selected", Mode: OAuth, RegionId: "cn-hangzhou", OAuthSiteType: "INTL"}}}
		if keepOther {
			conf.Profiles = append(conf.Profiles, Profile{Name: "other"})
		}
		require.NoError(t, SaveConfiguration(conf))
		var out bytes.Buffer
		ctx := lifecycleContext(&out)
		assignLifecycleFlag(t, ctx, ProfileFlagName, "selected")
		require.NoError(t, NewConfigureGetCommand().Run(ctx, []string{OAuthSiteTypeName}))
		require.Contains(t, out.String(), "oauth-site-type=INTL")
		require.NoError(t, NewConfigureDeleteCommand().Run(ctx, nil))
		stored, err := LoadConfigurationFromFile(filepath.Join(GetConfigPath(), configFile))
		require.NoError(t, err)
		if keepOther {
			require.Equal(t, "other", stored.CurrentProfile)
		} else {
			require.Equal(t, DefaultConfigProfileName, stored.CurrentProfile)
		}
		_, exists := stored.GetProfile("selected")
		require.False(t, exists)
	}
	var profile Profile
	require.Error(t, json.Unmarshal([]byte(`{"read_timeout":"invalid"}`), &profile))
	profile = Profile{Mode: BearerToken, BearerTokenValue: "token", BearerTokenHeaderKey: "X-Token"}
	profile.InjectBearerTokenHeader(nil)
}

func TestInteractiveConfigureUsesDefaultProfileAndReportsLoadErrors(t *testing.T) {
	isolatedConfigurationLifecycle(t)
	oldInput := stdin
	t.Cleanup(func() { stdin = oldInput })
	stdin = &configureLineReader{lines: []string{"id\n", "secret\n", "cn-hangzhou\n", "en\n"}}
	cause := errors.New("save interrupted")
	hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
		return func(string) (*Configuration, error) { return &Configuration{}, nil }
	}
	hookSaveConfigurationWithContext = func(func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error {
		return func(_ *cli.Context, conf *Configuration) error {
			stored, ok := conf.GetProfile("default")
			require.True(t, ok)
			require.Equal(t, AK, stored.Mode)
			require.Equal(t, "id", stored.AccessKeyId)
			return cause
		}
	}
	require.ErrorIs(t, doConfigure(lifecycleContext(io.Discard), "", ""), cause)
	hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
		return func(string) (*Configuration, error) { return nil, cause }
	}
	require.ErrorIs(t, doConfigure(lifecycleContext(io.Discard), "", ""), cause)
}
