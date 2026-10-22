package config

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

func TestSharedSSOConfigSymlinksReuseLogin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires Unix symlink support")
	}
	for _, kind := range []string{"default file", "custom file", "directory", "missing config in linked directory", "symlink parent traversal", "relative symlink parent traversal", "missing config after symlink parent traversal", "missing directory after symlink parent traversal"} {
		for _, loginViaLink := range []bool{true, false} {
			direction := "real to link"
			if loginViaLink {
				direction = "link to real"
			}
			t.Run(kind+"/"+direction, func(t *testing.T) {
				base := t.TempDir()
				home := filepath.Join(base, "home")
				realDir := filepath.Join(base, "real")
				require.NoError(t, os.MkdirAll(realDir, 0700))
				oldHome, oldLogin, oldRefresh, oldSave, oldLoad := hookGetHomePath, cloudssoGetAccessToken, tryRefreshStsTokenFunc, saveConfigurationFunc, hookLoadOrCreateConfiguration
				t.Cleanup(func() {
					hookGetHomePath, cloudssoGetAccessToken, tryRefreshStsTokenFunc, saveConfigurationFunc, hookLoadOrCreateConfiguration = oldHome, oldLogin, oldRefresh, oldSave, oldLoad
				})
				hookGetHomePath = func(func() string) func() string { return func() string { return home } }
				hookLoadOrCreateConfiguration = func(fn func(string) (*Configuration, error)) func(string) (*Configuration, error) { return fn }
				saveConfigurationFunc = SaveConfigurationWithContext
				realPath := filepath.Join(realDir, "config.json")
				linkPath := filepath.Join(base, "alias.json")
				conf := &Configuration{CurrentProfile: "prod", Profiles: []Profile{{Name: "prod", Mode: CloudSSO, RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en", SSOSession: "company", CloudSSOAccountId: "account", CloudSSOAccessConfig: "read"}}}
				missing := kind == "missing config in linked directory" || kind == "missing config after symlink parent traversal" || kind == "missing directory after symlink parent traversal"
				if !missing {
					require.NoError(t, SaveConfigurationWithContext(ssoPathContext(realPath), conf))
				}
				switch kind {
				case "default file":
					linkPath = filepath.Join(home, ".aliyun", configFile)
					require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0700))
					require.NoError(t, os.Symlink(realPath, linkPath))
				case "custom file":
					require.NoError(t, os.Symlink(realPath, linkPath))
				case "symlink parent traversal", "relative symlink parent traversal", "missing config after symlink parent traversal", "missing directory after symlink parent traversal":
					nested := filepath.Join(realDir, "nested")
					require.NoError(t, os.MkdirAll(nested, 0700))
					alias := filepath.Join(base, "alias")
					require.NoError(t, os.Symlink(nested, alias))
					// Do not use Join: it would remove alias/.. before the filesystem sees it.
					linkPath = alias + "/../config.json"
					if kind == "relative symlink parent traversal" {
						t.Chdir(base)
						linkPath = "alias/../config.json"
					}
					if kind == "missing directory after symlink parent traversal" {
						realPath = filepath.Join(realDir, "future", "config.json")
						linkPath = alias + "/../future/config.json"
					}
					if !missing {
						direct, err := os.Stat(realPath)
						require.NoError(t, err)
						viaLink, err := os.Stat(linkPath)
						require.NoError(t, err)
						require.True(t, os.SameFile(direct, viaLink))
					}
				default:
					aliasDir := filepath.Join(base, "alias")
					require.NoError(t, os.Symlink(realDir, aliasDir))
					linkPath = filepath.Join(aliasDir, "config.json")
				}
				realCtx, linkCtx := ssoPathContext(realPath), ssoPathContext(linkPath)
				if kind == "default file" {
					ConfigurePathFlag(linkCtx.Flags()).SetAssigned(false)
				}
				loginCtx, useCtx := realCtx, linkCtx
				if loginViaLink {
					loginCtx, useCtx = linkCtx, realCtx
				}
				SSOSessionFlag(loginCtx.Flags()).SetValue("company")
				SSOSessionFlag(loginCtx.Flags()).SetAssigned(true)
				CloudSSOSignInUrlFlag(loginCtx.Flags()).SetValue("https://example.invalid")
				CloudSSOSignInUrlFlag(loginCtx.Flags()).SetAssigned(true)
				cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
					return &cloudsso.AccessTokenResponse{AccessToken: "shared", ExpiresIn: 3600}, nil
				}
				require.NoError(t, NewConfigureSSOCommand().GetSubCommand("login").Run(loginCtx, nil))
				sessionPathBeforeSave := ssoSessionPath(loginCtx, "company")
				require.Equal(t, sessionPathBeforeSave, ssoSessionPath(useCtx, "company"))
				// The namespace must remain stable when a previously absent config is created.
				require.NoError(t, SaveConfigurationWithContext(realCtx, conf))
				require.Equal(t, sessionPathBeforeSave, ssoSessionPath(loginCtx, "company"))
				directInfo, err := os.Stat(realPath)
				require.NoError(t, err)
				aliasInfo, err := os.Stat(linkPath)
				require.NoError(t, err)
				require.True(t, os.SameFile(directInfo, aliasInfo))
				calls := 0
				tryRefreshStsTokenFunc = func(_ *string, token, ac, account *string, _ *http.Client) (*cloudsso.CloudCredentialResponse, error) {
					calls++
					require.Equal(t, "shared", *token)
					require.Equal(t, "read", *ac)
					require.Equal(t, "account", *account)
					return &cloudsso.CloudCredentialResponse{AccessKeyId: "key", AccessKeySecret: "secret", SecurityToken: "sts", ExpirationInt64: time.Now().Add(time.Hour).Unix()}, nil
				}
				p, err := LoadProfileWithContext(useCtx)
				require.NoError(t, err)
				cred, err := p.GetCredential(useCtx, nil)
				require.NoError(t, err)
				model, err := cred.GetCredential()
				require.NoError(t, err)
				require.Equal(t, "sts", tea.StringValue(model.SecurityToken))
				require.Equal(t, 1, calls)
				// An unrelated file in the same real directory remains isolated.
				_, err = loadSSOSession(ssoPathContext(filepath.Join(realDir, "other.json")), "company")
				require.ErrorContains(t, err, "does not exist")
			})
		}
	}
}

func ssoPathContext(path string) *cli.Context {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	return ctx
}
