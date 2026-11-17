package config

import (
	"io"
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

func TestSharedSSODefaultPathExplicitAndImplicitAreEquivalent(t *testing.T) {
	for _, explicitLogin := range []bool{false, true} {
		name := "implicit login"
		if explicitLogin {
			name = "explicit login"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			oldHome, oldToken, oldRefresh, oldSave, oldLoad := hookGetHomePath, cloudssoGetAccessToken, tryRefreshStsTokenFunc, saveConfigurationFunc, hookLoadOrCreateConfiguration
			t.Cleanup(func() {
				hookGetHomePath, cloudssoGetAccessToken, tryRefreshStsTokenFunc, saveConfigurationFunc, hookLoadOrCreateConfiguration = oldHome, oldToken, oldRefresh, oldSave, oldLoad
			})
			hookGetHomePath = func(func() string) func() string { return func() string { return home } }
			hookLoadOrCreateConfiguration = func(fn func(string) (*Configuration, error)) func(string) (*Configuration, error) { return fn }
			saveConfigurationFunc = SaveConfigurationWithContext
			newContext := func(explicit bool) *cli.Context {
				ctx := cli.NewCommandContext(io.Discard, io.Discard)
				AddFlags(ctx.Flags())
				if explicit {
					ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(home, ".aliyun", configFile))
					ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
				}
				return ctx
			}
			loginCtx, useCtx := newContext(explicitLogin), newContext(!explicitLogin)
			SSOSessionFlag(loginCtx.Flags()).SetValue("company")
			SSOSessionFlag(loginCtx.Flags()).SetAssigned(true)
			CloudSSOSignInUrlFlag(loginCtx.Flags()).SetValue("https://example.invalid")
			CloudSSOSignInUrlFlag(loginCtx.Flags()).SetAssigned(true)
			cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
				return &cloudsso.AccessTokenResponse{AccessToken: "shared", ExpiresIn: 3600}, nil
			}
			require.NoError(t, NewConfigureSSOCommand().GetSubCommand("login").Run(loginCtx, nil))
			// Saving the profile after login must not change the session namespace.
			conf := &Configuration{CurrentProfile: "prod", Profiles: []Profile{{Name: "prod", Mode: CloudSSO, RegionId: "cn-hangzhou", SSOSession: "company", CloudSSOAccountId: "account", CloudSSOAccessConfig: "read"}}}
			require.NoError(t, SaveConfigurationWithContext(loginCtx, conf))
			expected := filepath.Join(normalizeSSOConfigPath(home), ".aliyun", "sso", "company.json")
			require.Equal(t, expected, ssoSessionPath(loginCtx, "company"))
			require.Equal(t, expected, ssoSessionPath(useCtx, "company"))
			calls := 0
			tryRefreshStsTokenFunc = func(_ *string, token, ac, account *string, _ *http.Client) (*cloudsso.CloudCredentialResponse, error) {
				calls++
				require.Equal(t, "shared", *token)
				require.Equal(t, "account", *account)
				require.Equal(t, "read", *ac)
				return &cloudsso.CloudCredentialResponse{AccessKeyId: "key", AccessKeySecret: "secret", SecurityToken: "sts", ExpirationInt64: time.Now().Add(time.Hour).Unix()}, nil
			}
			p, err := LoadProfileWithContext(useCtx)
			require.NoError(t, err)
			credential, err := p.GetCredential(useCtx, nil)
			require.NoError(t, err)
			model, err := credential.GetCredential()
			require.NoError(t, err)
			require.Equal(t, "sts", tea.StringValue(model.SecurityToken))
			require.Equal(t, 1, calls)
		})
	}
}

func TestSharedSSONormalizedCustomConfigPaths(t *testing.T) {
	ctx := sharedSSOContext(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	absolute := getConfigurePath(ctx)
	// Windows cannot express a relative path across different drive letters.
	if filepath.VolumeName(cwd) != filepath.VolumeName(absolute) {
		t.Chdir(filepath.Dir(absolute))
		cwd, err = os.Getwd()
		require.NoError(t, err)
	}
	relative, err := filepath.Rel(cwd, absolute)
	require.NoError(t, err)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid", AccessToken: "shared"}))
	expected := ssoSessionPath(ctx, "company")
	for _, path := range []string{relative, filepath.Dir(absolute) + string(os.PathSeparator) + "." + string(os.PathSeparator) + filepath.Base(absolute), filepath.Dir(absolute) + string(os.PathSeparator) + "unused" + string(os.PathSeparator) + ".." + string(os.PathSeparator) + filepath.Base(absolute)} {
		ConfigurePathFlag(ctx.Flags()).SetValue(path)
		require.Equal(t, expected, ssoSessionPath(ctx, "company"))
		session, err := loadSSOSession(ctx, "company")
		require.NoError(t, err)
		require.Equal(t, "shared", session.AccessToken)
	}
	ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(filepath.Dir(absolute), "other.json"))
	require.NotEqual(t, expected, ssoSessionPath(ctx, "company"))
	_, err = loadSSOSession(ctx, "company")
	require.ErrorContains(t, err, "does not exist")
}
