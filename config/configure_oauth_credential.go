package config

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// NewConfigureOAuthCredentialCommand is a credential-process boundary, not a
// configuration dump. OAuth secrets never leave the CLI-managed profile.
func NewConfigureOAuthCredentialCommand() *cli.Command {
	return &cli.Command{Name: "oauth-credential", Short: i18n.T("emit refreshed OAuth STS for credential processes (sensitive output)", "为凭证进程输出刷新的 OAuth STS（敏感输出）"), Run: func(ctx *cli.Context, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("oauth-credential takes no positional arguments")
		}
		// Pin the selected file/profile and discard refresh diagnostics so stdout is
		// exactly one versioned JSON object, even during token rotation.
		conf, err := LoadConfigurationWithContext(ctx)
		if err != nil {
			return fmt.Errorf("cannot load OAuth configuration")
		}
		profile := conf.GetCurrentProfile(ctx)
		if name, ok := ProfileFlag(ctx.Flags()).GetValue(); ok {
			var found bool
			profile, found = conf.GetProfile(name)
			if !found {
				return fmt.Errorf("OAuth profile not found")
			}
		}
		if profile.Mode != OAuth {
			return fmt.Errorf("oauth-credential requires an OAuth profile")
		}
		quiet := cli.NewCommandContext(io.Discard, io.Discard)
		AddFlags(quiet.Flags())
		ConfigurePathFlag(quiet.Flags()).SetValue(getConfigurePath(ctx))
		ConfigurePathFlag(quiet.Flags()).SetAssigned(true)
		if _, err := profile.GetCredential(quiet, nil); err != nil {
			return fmt.Errorf("OAuth credential refresh failed; reauthorize the selected profile if needed")
		}
		if !hasUsableOAuthSTS(&profile) {
			return fmt.Errorf("OAuth did not return usable STS credentials")
		}
		return json.NewEncoder(ctx.Stdout()).Encode(struct {
			Version         int
			Type            string
			AccessKeyId     string
			AccessKeySecret string
			SecurityToken   string
			Expiration      string
		}{1, "sts", profile.AccessKeyId, profile.AccessKeySecret, profile.StsToken, time.Unix(profile.StsExpiration, 0).UTC().Format(time.RFC3339)})
	}}
}
