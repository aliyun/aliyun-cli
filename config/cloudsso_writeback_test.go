package config

import (
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

func TestCloudSSOTargetOverrideDoesNotPolluteStoredCredentials(t *testing.T) {
	for _, shared := range []bool{false, true} {
		for _, target := range []string{"account", "access configuration", "sign-in URL"} {
			if shared && target == "sign-in URL" {
				continue
			}
			mode := "legacy"
			if shared {
				mode = "shared"
			}
			t.Run(mode+"/"+target, func(t *testing.T) {
				ctx := sharedSSOContext(t)
				SSOSessionFlag(ctx.Flags()).SetAssigned(false)
				ProfileFlag(ctx.Flags()).SetValue("prod")
				ProfileFlag(ctx.Flags()).SetAssigned(true)
				profile := Profile{Name: "prod", Mode: CloudSSO, RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en", CloudSSOSignInUrl: "https://one.invalid", CloudSSOAccountId: "account-a", CloudSSOAccessConfig: "read", AccessToken: "access", CloudSSOAccessTokenExpire: time.Now().Add(time.Hour).Unix(), AccessKeyId: "key-a", AccessKeySecret: "secret-a", StsToken: "sts-a", StsExpiration: time.Now().Add(time.Hour).Unix()}
				if shared {
					profile.SSOSession = "company"
					clearCloudSSOLogin(&profile)
					require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://one.invalid", AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour).Unix()}))
				}
				require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "prod", Profiles: []Profile{profile}}))
				before, err := os.ReadFile(getConfigurePath(ctx))
				require.NoError(t, err)
				expectedAccount, expectedConfig, expectedURL := "account-a", "read", "https://one.invalid"
				var flag *cli.Flag
				switch target {
				case "account":
					flag = CloudSSOAccountIdFlag(ctx.Flags())
					expectedAccount = "account-b"
					flag.SetValue(expectedAccount)
				case "access configuration":
					flag = CloudSSOAccessConfigFlag(ctx.Flags())
					expectedConfig = "admin"
					flag.SetValue(expectedConfig)
				case "sign-in URL":
					flag = CloudSSOSignInUrlFlag(ctx.Flags())
					expectedURL = "https://two.invalid"
					flag.SetValue(expectedURL)
				}
				flag.SetAssigned(true)
				oldRefresh, oldSave := tryRefreshStsTokenFunc, saveConfigurationFunc
				t.Cleanup(func() { tryRefreshStsTokenFunc, saveConfigurationFunc = oldRefresh, oldSave })
				saveConfigurationFunc = SaveConfigurationWithContext
				calls := 0
				tryRefreshStsTokenFunc = func(url, token, config, account *string, _ *http.Client) (*cloudsso.CloudCredentialResponse, error) {
					calls++
					require.Equal(t, expectedURL, *url)
					require.Equal(t, expectedAccount, *account)
					require.Equal(t, expectedConfig, *config)
					return &cloudsso.CloudCredentialResponse{AccessKeyId: "override-key", AccessKeySecret: "override-secret", SecurityToken: "override-sts", ExpirationInt64: time.Now().Add(time.Hour).Unix()}, nil
				}
				overridden, err := LoadProfileWithContext(ctx)
				require.NoError(t, err)
				overridden.OverwriteWithFlags(ctx)
				require.Empty(t, overridden.StsToken)
				credential, err := overridden.GetCredential(ctx, nil)
				require.NoError(t, err)
				model, err := credential.GetCredential()
				require.NoError(t, err)
				require.Equal(t, "override-sts", tea.StringValue(model.SecurityToken))
				after, err := os.ReadFile(getConfigurePath(ctx))
				require.NoError(t, err)
				require.Equal(t, string(before), string(after), "one-off overrides must not change the stored profile or its credentials")
				// A subsequent command without the override must keep using account A's cache.
				clean := cli.NewCommandContext(io.Discard, io.Discard)
				AddFlags(clean.Flags())
				ConfigurePathFlag(clean.Flags()).SetValue(getConfigurePath(ctx))
				ConfigurePathFlag(clean.Flags()).SetAssigned(true)
				restored, err := LoadProfileWithContext(clean)
				require.NoError(t, err)
				credential, err = restored.GetCredential(clean, nil)
				require.NoError(t, err)
				model, err = credential.GetCredential()
				require.NoError(t, err)
				require.Equal(t, "sts-a", tea.StringValue(model.SecurityToken))
				require.Equal(t, 1, calls)
			})
		}
	}
}
