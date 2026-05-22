package maxc

import (
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

// loadProfileFunc is the test seam for injecting a synthetic profile or
// profile-load failure.
var loadProfileFunc = func(ctx *cli.Context) (config.Profile, error) {
	return config.LoadProfileWithContext(ctx)
}

// InjectAliyunCredentials reads the active aliyun profile and populates
// c.envMap with the ALIBABA_CLOUD_* credential triple (+ optional region).
// On profile-missing the child is left to resolve its own config — no error
// is surfaced.
//
// AK and StsToken modes read the profile fields directly; anything else
// (RamRoleArn, EcsRamRole, OIDC, CloudSSO, …) goes through Profile.GetCredential
// so STS exchange happens once, here, and the child sees only short-lived
// keys via environment.
func (c *Context) InjectAliyunCredentials(args []string) error {
	if c.envMap == nil {
		c.envMap = map[string]string{}
	}

	profile, err := loadProfileFunc(c.originCtx)
	if err != nil {
		return nil
	}

	id, secret, token, err := extractCredentials(c.originCtx, profile)
	if err != nil {
		return fmt.Errorf("resolve credentials for profile %q: %w", profile.Name, err)
	}

	if id != "" {
		c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"] = id
	}
	if secret != "" {
		c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"] = secret
	}
	if token != "" {
		c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"] = token
	}
	if profile.RegionId != "" {
		c.envMap["MAXCOMPUTE_REGION"] = profile.RegionId
	}
	return nil
}

func extractCredentials(ctx *cli.Context, p config.Profile) (string, string, string, error) {
	switch p.Mode {
	case config.AK:
		return p.AccessKeyId, p.AccessKeySecret, "", nil
	case config.StsToken:
		return p.AccessKeyId, p.AccessKeySecret, p.StsToken, nil
	default:
		cred, err := p.GetCredential(ctx, nil)
		if err != nil {
			return "", "", "", err
		}
		model, err := cred.GetCredential()
		if err != nil {
			return "", "", "", err
		}
		var id, secret, token string
		if model.AccessKeyId != nil {
			id = *model.AccessKeyId
		}
		if model.AccessKeySecret != nil {
			secret = *model.AccessKeySecret
		}
		if model.SecurityToken != nil {
			token = *model.SecurityToken
		}
		return id, secret, token, nil
	}
}
