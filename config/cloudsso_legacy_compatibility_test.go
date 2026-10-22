package config

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

func legacyCloudSSOProfileForTest() Profile {
	return Profile{Name: "legacy", Mode: CloudSSO, RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en", CloudSSOSignInUrl: "https://legacy.invalid", CloudSSOAccountId: "account-a", CloudSSOAccessConfig: "read", AccessToken: "inline-token", CloudSSOAccessTokenExpire: time.Now().Add(time.Hour).Unix(), AccessKeyId: "cached-key", AccessKeySecret: "cached-secret", StsToken: "cached-sts", StsExpiration: time.Now().Add(time.Hour).Unix()}
}

func TestLegacyCloudSSONonTargetOverridesPreserveCacheAndRefresh(t *testing.T) {
	for _, expired := range []bool{false, true} {
		name := "cached"
		if expired {
			name = "expired"
		}
		t.Run(name, func(t *testing.T) {
			ctx := sharedSSOContext(t)
			SSOSessionFlag(ctx.Flags()).SetAssigned(false)
			ctx.SetInConfigureMode(true)
			original := legacyCloudSSOProfileForTest()
			if expired {
				original.StsExpiration = 1
			}
			require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "legacy", Profiles: []Profile{original}}))
			before, err := os.ReadFile(getConfigurePath(ctx))
			require.NoError(t, err)
			for _, item := range []struct {
				flag  *cli.Flag
				value string
			}{{RegionFlag(ctx.Flags()), "cn-beijing"}, {EndpointFlag(ctx.Flags()), "override.invalid"}, {ReadTimeoutFlag(ctx.Flags()), "30"}, {CloudSSOAccountIdFlag(ctx.Flags()), original.CloudSSOAccountId}, {CloudSSOAccessConfigFlag(ctx.Flags()), original.CloudSSOAccessConfig}, {CloudSSOSignInUrlFlag(ctx.Flags()), original.CloudSSOSignInUrl}} {
				item.flag.SetValue(item.value)
				item.flag.SetAssigned(true)
			}
			oldRefresh, oldSave := tryRefreshStsTokenFunc, saveConfigurationFunc
			t.Cleanup(func() { tryRefreshStsTokenFunc, saveConfigurationFunc = oldRefresh, oldSave })
			saveConfigurationFunc = SaveConfigurationWithContext
			calls := 0
			tryRefreshStsTokenFunc = func(url, token, ac, account *string, _ *http.Client) (*cloudsso.CloudCredentialResponse, error) {
				calls++
				require.Equal(t, original.CloudSSOSignInUrl, *url)
				require.Equal(t, original.AccessToken, *token)
				require.Equal(t, original.CloudSSOAccessConfig, *ac)
				require.Equal(t, original.CloudSSOAccountId, *account)
				return &cloudsso.CloudCredentialResponse{AccessKeyId: "new-key", AccessKeySecret: "new-secret", SecurityToken: "new-sts", ExpirationInt64: time.Now().Add(time.Hour).Unix()}, nil
			}
			current, err := LoadProfileWithContext(ctx)
			require.NoError(t, err)
			require.Equal(t, original.StsToken, current.StsToken)
			credential, err := current.GetCredential(ctx, nil)
			require.NoError(t, err)
			model, err := credential.GetCredential()
			require.NoError(t, err)
			if expired {
				require.Equal(t, 1, calls)
				require.Equal(t, "new-sts", tea.StringValue(model.SecurityToken))
			} else {
				require.Zero(t, calls)
				require.Equal(t, "cached-sts", tea.StringValue(model.SecurityToken))
				after, err := os.ReadFile(getConfigurePath(ctx))
				require.NoError(t, err)
				require.Equal(t, before, after)
			}
			saved, err := LoadConfigurationWithContext(ctx)
			require.NoError(t, err)
			stored, ok := saved.GetProfile("legacy")
			require.True(t, ok)
			expected := original
			if expired {
				expected.AccessKeyId = "new-key"
				expected.AccessKeySecret = "new-secret"
				expected.StsToken = "new-sts"
				expected.StsExpiration = stored.StsExpiration
				require.Greater(t, stored.StsExpiration, time.Now().Unix())
			}
			require.Equal(t, expected, stored, "refresh must preserve the stored target, inline login, and noncredential settings")
			_, err = os.Stat(filepath.Join(GetConfigDir(ctx), "sso"))
			require.True(t, os.IsNotExist(err), "legacy profiles must not need or create session storage")
		})
	}
}

func TestLegacyCloudSSOConfigureSetInvalidatesOnlyChangedTargets(t *testing.T) {
	for _, name := range []string{"region", "same account", "same access configuration", "same URL", "changed account", "changed access configuration", "changed URL"} {
		t.Run(name, func(t *testing.T) {
			ctx := sharedSSOContext(t)
			SSOSessionFlag(ctx.Flags()).SetAssigned(false)
			original := legacyCloudSSOProfileForTest()
			require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "legacy", Profiles: []Profile{original}}))
			oldLoad, oldSave := hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext
			t.Cleanup(func() { hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext = oldLoad, oldSave })
			hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
				return func(string) (*Configuration, error) { return LoadConfigurationFromFile(getConfigurePath(ctx)) }
			}
			hookSaveConfigurationWithContext = func(fn func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error { return fn }
			expected := original
			var flag *cli.Flag
			value := ""
			switch name {
			case "region":
				flag = RegionFlag(ctx.Flags())
				value = "cn-beijing"
				expected.RegionId = value
			case "same account":
				flag = CloudSSOAccountIdFlag(ctx.Flags())
				value = original.CloudSSOAccountId
			case "same access configuration":
				flag = CloudSSOAccessConfigFlag(ctx.Flags())
				value = original.CloudSSOAccessConfig
			case "same URL":
				flag = CloudSSOSignInUrlFlag(ctx.Flags())
				value = original.CloudSSOSignInUrl
			case "changed account":
				flag = CloudSSOAccountIdFlag(ctx.Flags())
				value = "account-b"
				expected.CloudSSOAccountId = value
				clearCloudSSOSTS(&expected)
			case "changed access configuration":
				flag = CloudSSOAccessConfigFlag(ctx.Flags())
				value = "admin"
				expected.CloudSSOAccessConfig = value
				clearCloudSSOSTS(&expected)
			case "changed URL":
				flag = CloudSSOSignInUrlFlag(ctx.Flags())
				value = "https://other.invalid"
				expected.CloudSSOSignInUrl = value
				clearCloudSSOSTS(&expected)
			}
			flag.SetValue(value)
			flag.SetAssigned(true)
			require.NoError(t, doConfigureSet(ctx))
			conf, err := LoadConfigurationWithContext(ctx)
			require.NoError(t, err)
			stored, ok := conf.GetProfile("legacy")
			require.True(t, ok)
			// configure set has always normalized the site field when saving.
			expected.Site = "china"
			require.Equal(t, expected, stored)
		})
	}
}
