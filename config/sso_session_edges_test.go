package config

import (
	"bytes"
	"io"
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

func TestSharedSSOStorageFailures(t *testing.T) {
	for _, data := range []string{"{", `{"name":"other","sign_in_url":"https://example.invalid"}`, `{"name":"company"}`} {
		t.Run(data, func(t *testing.T) {
			ctx := sharedSSOContext(t)
			path := ssoSessionPath(ctx, "company")
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
			require.NoError(t, os.WriteFile(path, []byte(data), 0600))
			_, err := loadSSOSession(ctx, "company")
			require.ErrorContains(t, err, "invalid SSO session")
			_, err = configureSSOSession(ctx, "company", false)
			require.ErrorContains(t, err, "invalid SSO session")
		})
	}
	t.Run("session path is directory", func(t *testing.T) {
		ctx := sharedSSOContext(t)
		require.NoError(t, os.MkdirAll(ssoSessionPath(ctx, "company"), 0700))
		_, err := loadSSOSession(ctx, "company")
		require.ErrorContains(t, err, "read SSO session")
	})
	t.Run("storage parent is a file", func(t *testing.T) {
		ctx := sharedSSOContext(t)
		parent := filepath.Dir(ssoSessionPath(ctx, "company"))
		require.NoError(t, os.MkdirAll(filepath.Dir(parent), 0700))
		require.NoError(t, os.WriteFile(parent, []byte("blocked"), 0600))
		require.Error(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://example.invalid"}))
		_, err := configureSSOSession(ctx, "company", false)
		require.Error(t, err)
	})
	t.Run("atomic write failure propagates", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires Unix symlink semantics")
		}
		ctx := sharedSSOContext(t)
		f := CloudSSOSignInUrlFlag(ctx.Flags())
		f.SetValue("https://example.invalid")
		f.SetAssigned(true)
		path := ssoSessionPath(ctx, "company")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		// A dangling symlink is absent to Stat, but cannot be resolved by the writer.
		require.NoError(t, os.Symlink(filepath.Join(t.TempDir(), "missing"), path))
		_, err := configureSSOSession(ctx, "company", false)
		require.ErrorContains(t, err, "symlink")
	})
}

func TestSharedSSOCommandInputAndReferenceFailures(t *testing.T) {
	for _, sub := range []string{"set", "login", "delete"} {
		t.Run(sub, func(t *testing.T) {
			ctx := sharedSSOContext(t)
			cmd := NewConfigureSSOCommand().GetSubCommand(sub)
			require.ErrorContains(t, cmd.Run(ctx, []string{"unexpected"}), "unexpected")
			SSOSessionFlag(ctx.Flags()).SetAssigned(false)
			require.Error(t, cmd.Run(ctx, nil))
		})
	}
	t.Run("missing flag definition", func(t *testing.T) {
		ctx := cli.NewCommandContext(io.Discard, io.Discard)
		_, err := ssoSessionName(ctx)
		require.ErrorContains(t, err, "missing --sso-session")
	})
	t.Run("invalid URL", func(t *testing.T) {
		ctx := sharedSSOContext(t)
		f := CloudSSOSignInUrlFlag(ctx.Flags())
		f.SetValue(":bad")
		f.SetAssigned(true)
		_, err := configureSSOSession(ctx, "company", false)
		require.Error(t, err)
		_, err = configureSSOSession(ctx, "../bad", false)
		require.ErrorContains(t, err, "invalid SSO session name")
	})
	t.Run("missing session deletion", func(t *testing.T) {
		ctx := sharedSSOContext(t)
		require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("delete").Run(ctx, nil), "does not exist")
	})
	t.Run("broken profile configuration", func(t *testing.T) {
		ctx := sharedSSOContext(t)
		require.NoError(t, os.WriteFile(getConfigurePath(ctx), []byte("{"), 0600))
		_, err := ssoSessionReferences(ctx, "company")
		require.Error(t, err)
		require.Error(t, NewConfigureSSOCommand().GetSubCommand("delete").Run(ctx, nil))
		require.NoError(t, saveSSOSession(ctx, &SSOSession{Name: "company", SignInURL: "https://old.invalid"}))
		f := CloudSSOSignInUrlFlag(ctx.Flags())
		f.SetValue("https://new.invalid")
		f.SetAssigned(true)
		_, err = configureSSOSession(ctx, "company", false)
		require.Error(t, err)
	})
	t.Run("profile config path blocked", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("requires Unix ENOTDIR semantics")
		}
		ctx := sharedSSOContext(t)
		blocker := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(blocker, []byte("blocked"), 0600))
		ConfigurePathFlag(ctx.Flags()).SetValue(filepath.Join(blocker, "config.json"))
		_, err := ssoSessionReferences(ctx, "company")
		require.Error(t, err)
	})
}

func TestSharedSSOLoginThroughProfile(t *testing.T) {
	ctx := sharedSSOContext(t)
	SSOSessionFlag(ctx.Flags()).SetAssigned(false)
	conf := &Configuration{CurrentProfile: "prod", Profiles: []Profile{{Name: "prod", SSOSession: "company"}}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	oldToken, oldInput := cloudssoGetAccessToken, stdin
	t.Cleanup(func() { cloudssoGetAccessToken, stdin = oldToken, oldInput })
	stdin = strings.NewReader("https://example.invalid\n")
	cloudssoGetAccessToken = func(*cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{AccessToken: "via-profile", ExpiresIn: 3600}, nil
	}
	require.NoError(t, NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil))
	saved, err := loadSSOSession(ctx, "company")
	require.NoError(t, err)
	require.Equal(t, "via-profile", saved.AccessToken)
	conf.Profiles[0].SSOSession = ""
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	require.ErrorContains(t, NewConfigureSSOCommand().GetSubCommand("login").Run(ctx, nil), "missing --sso-session")
}

func TestSharedSSOInteractiveProfileConfiguration(t *testing.T) {
	ctx := sharedSSOContext(t)
	p := Profile{Name: "prod", Mode: CloudSSO, SSOSession: "company", RegionId: "cn-hangzhou", Language: "en"}
	require.NoError(t, SaveConfigurationWithContext(ctx, &Configuration{CurrentProfile: "prod", Profiles: []Profile{p}}))
	_, err := loadSSOSession(ctx, "company")
	require.Error(t, err)
	require.ErrorContains(t, configureSharedCloudSSO(ctx, io.Discard, &p), "does not exist")
	session := &SSOSession{Name: "company", SignInURL: "https://example.invalid", AccessToken: "shared", ExpiresAt: 1}
	require.NoError(t, saveSSOSession(ctx, session))
	require.ErrorContains(t, configureSharedCloudSSO(ctx, io.Discard, &p), "configure sso login")
	session.ExpiresAt = time.Now().Add(time.Hour).Unix()
	require.NoError(t, saveSSOSession(ctx, session))
	oldUsers, oldAC, oldSTS, oldInput, oldSave := cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken, stdin, hookSaveConfigurationWithContext
	t.Cleanup(func() {
		cloudssoListAllUsers, cloudssoListAllAccessConfigurations, cloudssoTryRefreshStsToken, stdin, hookSaveConfigurationWithContext = oldUsers, oldAC, oldSTS, oldInput, oldSave
	})
	stdin = strings.NewReader("\n\n\n")
	hookSaveConfigurationWithContext = func(fn func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error { return fn }
	cloudssoListAllUsers = func(param *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		require.Equal(t, "shared", param.AccessToken)
		return []cloudsso.AccountDetailResponse{{AccountId: "account", DisplayName: "Production"}}, nil
	}
	cloudssoListAllAccessConfigurations = func(param *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		require.Equal(t, "account", req.AccountId)
		return []cloudsso.AccessConfiguration{{AccessConfigurationId: "read"}}, nil
	}
	cloudssoTryRefreshStsToken = func(url, token, ac, account *string, _ *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		require.Equal(t, "read", *ac)
		require.Equal(t, "shared", *token)
		return &cloudsso.CloudCredentialResponse{AccessKeyId: "key", AccessKeySecret: "secret", SecurityToken: "sts", Expiration: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}, nil
	}
	require.NoError(t, doConfigure(ctx, "prod", "CloudSSO"))
	conf, err := LoadConfigurationWithContext(ctx)
	require.NoError(t, err)
	saved, ok := conf.GetProfile("prod")
	require.True(t, ok)
	require.Equal(t, "company", saved.SSOSession)
	require.Equal(t, "account", saved.CloudSSOAccountId)
	require.Equal(t, "read", saved.CloudSSOAccessConfig)
	require.Equal(t, "sts", saved.StsToken)
	require.Empty(t, saved.AccessToken)
	require.Empty(t, saved.CloudSSOSignInUrl)
	require.Zero(t, saved.CloudSSOAccessTokenExpire)
	cloudssoListAllUsers = func(*cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return nil, io.ErrUnexpectedEOF
	}
	require.ErrorContains(t, configureSharedCloudSSO(ctx, io.Discard, &p), "list account failed")
	require.Empty(t, p.AccessToken)
	require.Empty(t, p.CloudSSOSignInUrl)
	require.Zero(t, p.CloudSSOAccessTokenExpire)
}

func TestSharedSSOConfigureGetSessionReference(t *testing.T) {
	ctx := sharedSSOContext(t)
	conf := &Configuration{CurrentProfile: "prod", Profiles: []Profile{{Name: "prod", Mode: CloudSSO, SSOSession: "company"}}}
	require.NoError(t, SaveConfigurationWithContext(ctx, conf))
	oldLoad := hookLoadConfigurationWithContext
	t.Cleanup(func() { hookLoadConfigurationWithContext = oldLoad })
	hookLoadConfigurationWithContext = func(fn func(*cli.Context) (*Configuration, error)) func(*cli.Context) (*Configuration, error) {
		return fn
	}
	require.NoError(t, doConfigureGet(ctx, []string{SSOSessionFlagName}))
	require.Contains(t, ctx.Stdout().(*bytes.Buffer).String(), "sso-session=company")
}
