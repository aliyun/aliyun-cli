package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// SSOSession stores both durable configuration and the current local login.
// STS credentials remain in profiles, scoped to their account/access configuration.
type SSOSession struct {
	Name        string `json:"name"`
	SignInURL   string `json:"sign_in_url"`
	AccessToken string `json:"access_token,omitempty"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

func validateSSOSessionName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("invalid SSO session name %q: use 1-64 lowercase letters, digits, '-' or '_'", name)
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return fmt.Errorf("invalid SSO session name %q: use 1-64 lowercase letters, digits, '-' or '_'", name)
		}
	}
	return nil
}

func normalizeSSOConfigPath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		return absolute
	}
	return filepath.Clean(path)
}

func ssoSessionPath(ctx *cli.Context, name string) string {
	// Namespace by the actual configuration path, not how it was specified.
	path := normalizeSSOConfigPath(getConfigurePath(ctx))
	defaultPath := normalizeSSOConfigPath(filepath.Join(hookGetHomePath(GetHomePath)()+configPath, configFile))
	dir := filepath.Join(filepath.Dir(path), "sso")
	if path != defaultPath {
		namespace := sha256.Sum256([]byte(path))
		dir = filepath.Join(dir, hex.EncodeToString(namespace[:]))
	}
	return filepath.Join(dir, name+".json")
}

func loadSSOSession(ctx *cli.Context, name string) (*SSOSession, error) {
	if err := validateSSOSessionName(name); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(ssoSessionPath(ctx, name))
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("SSO session %q does not exist; configure it with `aliyun configure sso set --sso-session %s`", name, name)
	}
	if err != nil {
		return nil, fmt.Errorf("read SSO session %q: %w", name, err)
	}
	var session SSOSession
	if json.Unmarshal(b, &session) != nil || session.Name != name || session.SignInURL == "" {
		return nil, fmt.Errorf("invalid SSO session %q", name)
	}
	return &session, nil
}

func saveSSOSession(ctx *cli.Context, session *SSOSession) error {
	if err := validateSSOSessionName(session.Name); err != nil {
		return err
	}
	path := ssoSessionPath(ctx, session.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, b, 0600)
}

func ssoSessionName(ctx *cli.Context) (string, error) {
	if f := SSOSessionFlag(ctx.Flags()); f != nil {
		if name, ok := f.GetValue(); ok && strings.TrimSpace(name) != "" {
			return name, validateSSOSessionName(name)
		}
	}
	return "", fmt.Errorf("missing --sso-session <name>")
}

// configureSSOSession persists configuration before any optional browser login.
// Existing login state survives saving the same entry point and failed logins.
func configureSSOSession(ctx *cli.Context, name string, prompt bool) (*SSOSession, error) {
	if err := validateSSOSessionName(name); err != nil {
		return nil, err
	}
	session := &SSOSession{Name: name}
	if _, err := os.Stat(ssoSessionPath(ctx, name)); err == nil {
		session, err = loadSSOSession(ctx, name)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	signInURL := CloudSSOSignInUrlFlag(ctx.Flags()).GetStringOrDefault(session.SignInURL)
	if !CloudSSOSignInUrlFlag(ctx.Flags()).IsAssigned() && (prompt || signInURL == "") {
		cli.Printf(ctx.Stdout(), "CloudSSO Sign In Url [%s]: ", signInURL)
		signInURL = ReadInput(signInURL)
	}
	if _, err := cloudsso.ParseUrl(&signInURL); err != nil {
		return nil, err
	}
	if signInURL != session.SignInURL {
		if session.SignInURL != "" {
			refs, err := ssoSessionReferences(ctx, name)
			if err != nil {
				return nil, err
			}
			if len(refs) > 0 {
				return nil, fmt.Errorf("cannot change SSO session %q URL: referenced by profiles: %s", name, strings.Join(refs, ", "))
			}
		}
		session.AccessToken, session.ExpiresAt = "", 0
	}
	session.SignInURL = signInURL
	if err := saveSSOSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func NewConfigureSSOCommand() *cli.Command {
	cmd := &cli.Command{Name: "sso", Short: i18n.T("Manage shared CloudSSO sessions", "管理共享 CloudSSO 会话")}
	cmd.AddSubCommand(&cli.Command{Name: "set", Usage: "set --sso-session <name> [--cloud-sso-sign-in-url <url>]", Short: i18n.T("Save CloudSSO session configuration without logging in", "保存 CloudSSO 会话配置，不登录"), Run: func(ctx *cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.NewInvalidCommandError(args[0], ctx)
		}
		name, err := ssoSessionName(ctx)
		if err != nil {
			return err
		}
		_, err = configureSSOSession(ctx, name, true)
		return err
	}})
	cmd.AddSubCommand(&cli.Command{Name: "login", Usage: "login [--sso-session <name> | --profile <name>] [--cloud-sso-sign-in-url <url>]", Short: i18n.T("Log in to CloudSSO, configuring the session on first use", "登录 CloudSSO，首次使用时配置会话"), Run: func(ctx *cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.NewInvalidCommandError(args[0], ctx)
		}
		name, err := ssoSessionName(ctx)
		if err != nil {
			if f := SSOSessionFlag(ctx.Flags()); f != nil && f.IsAssigned() {
				return err
			}
			conf, loadErr := LoadConfigurationWithContext(ctx)
			if loadErr != nil {
				return loadErr
			}
			name = conf.GetCurrentProfile(ctx).SSOSession
			if name == "" {
				return err
			}
		}
		session, err := configureSSOSession(ctx, name, false)
		if err != nil {
			return err
		}
		token, err := cloudssoGetAccessToken(&cloudsso.SsoLogin{SignInUrl: session.SignInURL, HttpClient: utilNewHttpClient()})
		if err != nil {
			return fmt.Errorf("CloudSSO login failed; configuration saved, retry with `aliyun configure sso login --sso-session %s`: %w", name, err)
		}
		session.AccessToken, session.ExpiresAt = token.AccessToken, utilGetCurrentUnixTime()+int64(token.ExpiresIn)
		return saveSSOSession(ctx, session)
	}})
	cmd.AddSubCommand(&cli.Command{Name: "delete", Usage: "delete --sso-session <name>", Short: i18n.T("Delete an unreferenced CloudSSO session", "删除未被引用的 CloudSSO 会话"), Run: func(ctx *cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.NewInvalidCommandError(args[0], ctx)
		}
		name, err := ssoSessionName(ctx)
		if err != nil {
			return err
		}
		refs, err := ssoSessionReferences(ctx, name)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return fmt.Errorf("cannot delete SSO session %q: referenced by profiles: %s; update or delete these profiles first", name, strings.Join(refs, ", "))
		}
		if _, err := loadSSOSession(ctx, name); err != nil {
			return err
		}
		return os.Remove(ssoSessionPath(ctx, name))
	}})
	return cmd
}

func ssoSessionReferences(ctx *cli.Context, name string) ([]string, error) {
	if _, err := os.Stat(getConfigurePath(ctx)); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	conf, err := LoadConfigurationWithContext(ctx)
	if err != nil {
		return nil, err
	}
	var refs []string
	for _, p := range conf.Profiles {
		if p.SSOSession == name {
			refs = append(refs, p.Name)
		}
	}
	sort.Strings(refs)
	return refs, nil
}

func ssoLoginRequired(name string) error {
	return fmt.Errorf("SSO session %q is not logged in or has expired; run `aliyun configure sso login --sso-session %s`", name, name)
}
func clearCloudSSOSTS(p *Profile) {
	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.StsToken = ""
	p.StsExpiration = 0
}
func clearCloudSSOLogin(p *Profile) {
	p.CloudSSOSignInUrl = ""
	p.AccessToken = ""
	p.CloudSSOAccessTokenExpire = 0
}
func cloudSSOTarget(p *Profile) string {
	url := p.CloudSSOSignInUrl
	if p.SSOSession != "" {
		url = ""
	}
	return strings.Join([]string{p.SSOSession, url, p.CloudSSOAccountId, p.CloudSSOAccessConfig}, "\x00")
}

func configureSharedCloudSSO(ctx *cli.Context, w io.Writer, p *Profile) error {
	session, err := loadSSOSession(ctx, p.SSOSession)
	if err != nil {
		return err
	}
	if session.AccessToken == "" || session.ExpiresAt <= utilGetCurrentUnixTime() {
		return ssoLoginRequired(p.SSOSession)
	}
	p.CloudSSOSignInUrl, p.AccessToken, p.CloudSSOAccessTokenExpire = session.SignInURL, session.AccessToken, session.ExpiresAt
	defer clearCloudSSOLogin(p)
	return configureCloudSSOTarget(w, p, utilNewHttpClient())
}
