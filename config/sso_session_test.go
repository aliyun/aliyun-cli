package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

func sharedSSOContext(t *testing.T) *cli.Context {
	t.Helper()
	oldLoad := hookLoadOrCreateConfiguration
	hookLoadOrCreateConfiguration = func(fn func(string) (*Configuration, error)) func(string) (*Configuration, error) { return fn }
	t.Cleanup(func() { hookLoadOrCreateConfiguration = oldLoad })
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	AddFlags(ctx.Flags())
	f := ConfigurePathFlag(ctx.Flags())
	f.SetValue(filepath.Join(t.TempDir(), "config.json"))
	f.SetAssigned(true)
	f = SSOSessionFlag(ctx.Flags())
	f.SetValue("company")
	f.SetAssigned(true)
	return ctx
}

func TestSharedSSOSessionStorageAndDelete(t *testing.T) {
	ctx := sharedSSOContext(t)
	cmd := NewConfigureSSOCommand()
	f := CloudSSOSignInUrlFlag(ctx.Flags())
	f.SetValue("https://example.invalid/device/login")
	f.SetAssigned(true)
	require.NoError(t, cmd.GetSubCommand("set").Run(ctx, nil))
	session, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Empty(t, session.AccessToken)
	session.AccessToken = "private-token"
	session.ExpiresAt = time.Now().Add(time.Hour).Unix()
	require.NoError(t, saveSSOSession(ctx, session))
	t.Run("owner-only permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix permission bits are not supported on Windows")
		}
		info, err := os.Stat(ssoSessionPath(ctx, "company"))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
	conf := &Configuration{Profiles: []Profile{{Name: "prod", Mode: CloudSSO, SSOSession: "company"}}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	require.ErrorContains(t, cmd.GetSubCommand("delete").Run(ctx, nil), "prod")
	require.NoError(t, cmd.GetSubCommand("set").Run(ctx, nil)) // Saving the same URL preserves login.
	session, err = loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Equal(t, "private-token", session.AccessToken)
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{}))
	require.NoError(t, cmd.GetSubCommand("delete").Run(ctx, nil))
	_, err = loadSSOSession(ctx, "company")
	require.ErrorContains(t, err, "does not exist")
}

func TestSharedSSOLoginRetainsProfileSTS(t *testing.T) {
	ctx := sharedSSOContext(t)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid"}))
	old := cloudssoGetAccessToken
	t.Cleanup(func() { cloudssoGetAccessToken = old })
	calls := 0
	cloudssoGetAccessToken = func(login *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		calls++
		require.Equal(t, "https://example.invalid", login.SignInUrl)
		return &cloudsso.AccessTokenResponse{AccessToken: "new", ExpiresIn: 3600}, nil
	}
	conf := &Configuration{Profiles: []Profile{{Name: "prod", Mode: CloudSSO, SSOSession: "company", StsToken: "old-sts"}}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	before, err := os.ReadFile(getConfigurePath(ctx))
	require.NoError(t, err)
	require.NoError(t, NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil))
	after, err := os.ReadFile(getConfigurePath(ctx))
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, 1, calls)
	session, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Equal(t, "new", session.AccessToken)
}

func TestSharedSSOProfilesUseOneLogin(t *testing.T) {
	ctx := sharedSSOContext(t)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid", AccessToken: "shared", ExpiresAt: time.Now().Add(time.Hour).Unix()}))
	oldRefresh, oldSave := tryRefreshStsTokenFunc, saveConfigurationFunc
	t.Cleanup(func() { tryRefreshStsTokenFunc = oldRefresh; saveConfigurationFunc = oldSave })
	saveConfigurationFunc = SaveConfigurationWithContext
	calls := 0
	tryRefreshStsTokenFunc = func(url, token, ac, account *string, client *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		calls++
		require.Equal(t, "shared", *token)
		return &cloudsso.CloudCredentialResponse{AccessKeyId: "key-" + *account, AccessKeySecret: "secret", SecurityToken: "sts-" + *ac, ExpirationInt64: time.Now().Add(time.Hour).Unix()}, nil
	}
	conf := &Configuration{Profiles: []Profile{
		{Name: "prod", Mode: CloudSSO, SSOSession: "company", CloudSSOAccountId: "a", CloudSSOAccessConfig: "read"},
		{Name: "test", Mode: CloudSSO, SSOSession: "company", CloudSSOAccountId: "b", CloudSSOAccessConfig: "admin"},
	}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	for _, p := range conf.Profiles {
		_, err := p.GetCredential(ctx, nil)
		require.NoError(t, err)
	}
	require.Equal(t, 2, calls)
	disk, err := LoadConfigurationWithContext(ctx)
	require.NoError(t, err)
	for _, p := range disk.Profiles {
		require.NotEmpty(t, p.StsToken)
		require.Empty(t, p.AccessToken)
		require.Empty(t, p.CloudSSOSignInUrl)
		_, err := p.GetCredential(ctx, nil)
		require.NoError(t, err)
	}
	require.Equal(t, 2, calls)
	session, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	session.ExpiresAt = 1
	require.NoError(t, saveSSOSession(ctx, session))
	p := disk.Profiles[0]
	_, err = p.GetCredential(ctx, nil)
	require.ErrorContains(t, err, "aliyun configure sso login")
	require.NoError(t, os.Remove(ssoSessionPath(ctx, "company")))
	_, err = p.GetCredential(ctx, nil)
	require.ErrorContains(t, err, "does not exist")
}

func TestSharedSSOTargetChangesInvalidateSTS(t *testing.T) {
	ctx := sharedSSOContext(t)
	p := Profile{Mode: CloudSSO, SSOSession: "company", CloudSSOAccountId: "a", CloudSSOAccessConfig: "read", AccessKeyId: "old", StsToken: "old", StsExpiration: 9999999999}
	f := CloudSSOAccountIdFlag(ctx.Flags())
	f.SetValue("b")
	f.SetAssigned(true)
	p.OverwriteWithFlags(ctx)
	require.Empty(t, p.StsToken)
	require.Zero(t, p.StsExpiration)
	disk := Profile{Mode: CloudSSO, SSOSession: "company", CloudSSOAccountId: "a"}
	require.Equal(t, disk, mergeProfileAfterCredentialRefresh(disk, &p))
	// The resolved runtime URL is not part of the target when a session is used.
	p = disk
	p.CloudSSOSignInUrl = "https://resolved.invalid"
	p.StsToken = "new"
	require.Equal(t, "new", mergeProfileAfterCredentialRefresh(disk, &p).StsToken)
	b, err := json.Marshal(disk)
	require.NoError(t, err)
	require.Contains(t, string(b), `"sso_session":"company"`)
}

func TestSharedSSOConfigureSetMigratesAndClearsTargetCredentials(t *testing.T) {
	ctx := sharedSSOContext(t)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid"}))
	legacy := Profile{Name: "prod", RegionId: "cn-hangzhou", Mode: CloudSSO, CloudSSOSignInUrl: "https://legacy.invalid", AccessToken: "legacy-token", CloudSSOAccessTokenExpire: 9999999999, CloudSSOAccountId: "a", CloudSSOAccessConfig: "read", AccessKeyId: "old", StsToken: "old", StsExpiration: 9999999999}
	conf := &Configuration{CurrentProfile: "prod", Profiles: []Profile{legacy}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	oldLoad, oldSave := hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext
	t.Cleanup(func() { hookLoadOrCreateConfiguration = oldLoad; hookSaveConfigurationWithContext = oldSave })
	hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
		return func(string) (*Configuration, error) { return conf, nil }
	}
	hookSaveConfigurationWithContext = func(fn func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error { return fn }
	require.NoError(t, doConfigureSet(ctx))
	disk, err := LoadConfigurationWithContext(ctx)
	require.NoError(t, err)
	p := disk.Profiles[0]
	require.Equal(t, "company", p.SSOSession)
	require.Empty(t, p.AccessToken)
	require.Empty(t, p.CloudSSOSignInUrl)
	require.Empty(t, p.StsToken)
	p.AccessKeyId, p.StsToken, p.StsExpiration = "cached", "cached", 9999999999
	conf.Profiles[0] = p
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	require.NoError(t, doConfigureSet(ctx))
	disk, err = LoadConfigurationWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "cached", disk.Profiles[0].StsToken)
	f := CloudSSOAccessConfigFlag(ctx.Flags())
	f.SetValue("admin")
	f.SetAssigned(true)
	require.NoError(t, doConfigureSet(ctx))
	disk, err = LoadConfigurationWithContext(ctx)
	require.NoError(t, err)
	require.Empty(t, disk.Profiles[0].StsToken)
}

func TestSharedSSONamespacesAndReferencedURLChange(t *testing.T) {
	ctx := sharedSSOContext(t)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://one.invalid", AccessToken: "old"}))
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{Profiles: []Profile{{Name: "prod", SSOSession: "company"}}}))
	f := CloudSSOSignInUrlFlag(ctx.Flags())
	f.SetValue("https://two.invalid")
	f.SetAssigned(true)
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("set").Run(ctx, nil), "referenced by profiles: prod")
	s, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Equal(t, "old", s.AccessToken)
	ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(GetConfigDir(ctx), "other.json"))
	_, err = loadSSOSession(ctx, "company")
	require.ErrorContains(t, err, "does not exist")
}

func TestSharedSSOSessionNames(t *testing.T) {
	ctx := sharedSSOContext(t)
	for _, name := range []string{"company", "company-admin", "user_1", strings.Repeat("a", 64)} {
		require.NoError(t, validateSSOSessionName(name))
		require.Equal(t, name+".json", filepath.Base(ssoSessionPath(ctx, name)))
	}
	for _, name := range []string{"", "../company", "a/b", "a\\b", ".", "..", "Company", "with space", "中文", strings.Repeat("a", 65)} {
		require.Error(t, validateSSOSessionName(name))
		_, err := loadSSOSession(ctx, name)
		require.ErrorContains(t, err, "invalid SSO session name")
		require.ErrorContains(t, saveSSOSession(ctx, &SSOSession{Name: name}), "invalid SSO session name")
	}
	SSOSessionFlag(ctx.Flags()).SetValue("../company")
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("set").Run(ctx, nil), "invalid SSO session name")
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("delete").Run(ctx, nil), "invalid SSO session name")
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil), "invalid SSO session name")
}

func TestSharedSSOFirstLoginAndRetry(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		existing, explicit, fail bool
	}{
		{name: "first login with URL", explicit: true},
		{name: "first login prompts"},
		{name: "existing login reuses URL", existing: true},
		{name: "first login failure keeps configuration", explicit: true, fail: true},
		{name: "existing login failure keeps token", existing: true, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sharedSSOContext(t)
			oldInput, oldToken := stdin, cloudssoGetAccessToken
			t.Cleanup(func() { stdin, cloudssoGetAccessToken = oldInput, oldToken })
			stdin = strings.NewReader("https://example.invalid\n")
			if tc.existing {
				require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid", AccessToken: "prior", ExpiresAt: 9999999999}))
				stdin = strings.NewReader("invalid-if-prompted\n")
			}
			if tc.explicit {
				f := CloudSSOSignInUrlFlag(ctx.Flags())
				f.SetValue("https://example.invalid")
				f.SetAssigned(true)
				stdin = strings.NewReader("invalid-if-prompted\n")
			}
			calls := 0
			cause := errors.New("authorization denied")
			cloudssoGetAccessToken = func(login *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
				calls++
				require.Equal(t, "https://example.invalid", login.SignInUrl)
				saved, err := loadSSOSession(ctx, "company")
				require.NoError(t, err)
				require.Equal(t, login.SignInUrl, saved.SignInURL)
				if tc.fail {
					return nil, cause
				}
				return &cloudsso.AccessTokenResponse{AccessToken: "fresh", ExpiresIn: 3600}, nil
			}
			err := NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil)
			if tc.fail {
				require.ErrorIs(t, err, cause)
				require.ErrorContains(t, err, "aliyun configure sso login")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, calls)
			saved, err := loadSSOSession(ctx, "company")
			require.NoError(t, err)
			if !tc.fail {
				require.Equal(t, "fresh", saved.AccessToken)
			} else if tc.existing {
				require.Equal(t, "prior", saved.AccessToken)
				require.EqualValues(t, 9999999999, saved.ExpiresAt)
			} else {
				require.Empty(t, saved.AccessToken)
			}
		})
	}
}

func TestSharedSSOCommandRoutingAndSetDoesNotLogin(t *testing.T) {
	ctx := sharedSSOContext(t)
	oldToken := cloudssoGetAccessToken
	t.Cleanup(func() { cloudssoGetAccessToken = oldToken })
	cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		t.Fatal("set must not log in")
		return nil, nil
	}
	root := &cli.Command{Name: "aliyun"}
	AddFlags(root.Flags())
	root.AddSubCommand(NewConfigureCommand())
	require.Nil(t, root.GetSubCommand("sso"))
	require.Nil(t, root.GetSubCommand("configure").GetSubCommand("sso-session"))
	group := root.GetSubCommand("configure").GetSubCommand("sso")
	require.ElementsMatch(t, []string{"set", "login", "delete"}, group.SubCommandNames())
	execCtx := cli.NewCommandContext(new(bytes.Buffer), os.Stderr)
	execCtx.EnterCommand(root)
	root.Execute(execCtx, []string{"configure", "sso", "set", "--sso-session", "company", "--cloud-sso-sign-in-url", "https://example.invalid", "--config-path", getConfigurePath(ctx)})
	saved, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Empty(t, saved.AccessToken)
	require.ErrorContains(t, group.GetSubCommand("set").Run(ctx, []string{"unexpected"}), "unexpected")
}

func TestSharedSSOLoginRejectsReferencedURLChange(t *testing.T) {
	ctx := sharedSSOContext(t)
	require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://one.invalid", AccessToken: "prior"}))
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{Profiles: []Profile{{Name: "prod", SSOSession: "company"}}}))
	f := CloudSSOSignInUrlFlag(ctx.Flags())
	f.SetValue("https://two.invalid")
	f.SetAssigned(true)
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil), "referenced by profiles: prod")
	saved, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Equal(t, "prior", saved.AccessToken)
	require.Equal(t, "https://one.invalid", saved.SignInURL)
}
