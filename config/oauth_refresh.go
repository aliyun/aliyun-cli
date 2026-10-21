package config

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/util"
	"os"
	"path/filepath"
)

// Refresh before expiry so credentials do not expire during the next signed request.
const oauthRefreshSkewSeconds int64 = 60

func hasUsableOAuthSTS(p *Profile) bool {
	return p.StsExpiration > util.GetCurrentUnixTime()+oauthRefreshSkewSeconds && p.AccessKeyId != "" && p.AccessKeySecret != "" && p.StsToken != ""
}

func (cp *Profile) refreshOAuthCredential(ctx *cli.Context) error {
	if cp.OAuthSiteType != "CN" && cp.OAuthSiteType != "INTL" {
		return fmt.Errorf("invalid OAuth site type: %s, only support CN or INTL", cp.OAuthSiteType)
	}
	// Resolve file symlinks as well as directory aliases so all refreshers lock
	// and replace the same file, without replacing a user's configuration symlink.
	path := getConfigurePath(ctx)
	resolved, resolveErr := filepath.EvalSymlinks(path)
	if resolveErr != nil && !os.IsNotExist(resolveErr) {
		return resolveErr
	}
	if resolveErr == nil && resolved != path {
		pinned := cli.NewCommandContext(ctx.Stdout(), ctx.Stderr())
		AddFlags(pinned.Flags())
		ConfigurePathFlag(pinned.Flags()).SetValue(resolved)
		ConfigurePathFlag(pinned.Flags()).SetAssigned(true)
		ctx = pinned
	}
	unlock, err := lockOAuthConfiguration(getConfigurePath(ctx))
	if err != nil {
		return fmt.Errorf("lock OAuth configuration: %w", err)
	}
	defer unlock()
	conf, err := hookLoadOrCreateConfiguration(LoadOrCreateConfiguration)(getConfigurePath(ctx))
	if err != nil {
		return err
	}
	disk, ok := conf.GetProfile(cp.Name)
	if !ok {
		return fmt.Errorf("OAuth profile %q no longer exists", cp.Name)
	}
	if disk.Mode != OAuth || disk.OAuthSiteType != cp.OAuthSiteType {
		return fmt.Errorf("OAuth profile %q changed during refresh; retry the command", cp.Name)
	}
	// Retain request-only routing overrides, but never refresh the stale token
	// read before waiting for another process.
	refreshed := mergeProfileAfterCredentialRefresh(*cp, &disk)
	if !hasUsableOAuthSTS(&refreshed) {
		checkpoint := func() error {
			if refreshed.OAuthRefreshToken == disk.OAuthRefreshToken && refreshed.OAuthAccessToken == disk.OAuthAccessToken && refreshed.OAuthAccessTokenExpire == disk.OAuthAccessTokenExpire {
				return nil
			}
			conf.PutProfile(mergeProfileAfterCredentialRefresh(disk, &refreshed))
			if err := saveConfigurationFunc(ctx, conf); err != nil {
				return fmt.Errorf("persist rotated OAuth credentials before STS exchange: %w", err)
			}
			return nil
		}
		exchangeErr := exchangeFromOAuthFunc(ctx.Stdout(), &refreshed, checkpoint)
		// A successful OAuth rotation must survive even if the subsequent STS
		// exchange fails. Never retry using an already-consumed refresh token.
		if exchangeErr == nil || refreshed.OAuthRefreshToken != disk.OAuthRefreshToken || refreshed.OAuthAccessToken != disk.OAuthAccessToken {
			conf.PutProfile(mergeProfileAfterCredentialRefresh(disk, &refreshed))
			if err := saveConfigurationFunc(ctx, conf); err != nil {
				return fmt.Errorf("persist refreshed OAuth credentials: %w", err)
			}
		}
		if exchangeErr != nil {
			return exchangeErr
		}
	}
	*cp = refreshed
	return nil
}
