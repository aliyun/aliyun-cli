package mcpproxy

import (
	"fmt"
	"reflect"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

type mcpCredentialSource func() (*credentialsv2.CredentialModel, error)

func newCLICredentialSource(ctx *cli.Context) mcpCredentialSource {
	var cachedProfile config.Profile
	var cachedCredential credentialsv2.Credential
	return func() (*credentialsv2.CredentialModel, error) {
		if ctx == nil {
			return nil, fmt.Errorf("CLI context is required for MCP credential authentication")
		}
		profile, err := config.LoadProfileWithContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load MCP credential profile: %w", err)
		}
		var reusable bool
		switch profile.Mode {
		case config.AK, config.StsToken, config.RamRoleArn, config.EcsRamRole, config.RsaKeyPair, config.OIDC:
			reusable = true
		}
		credential := cachedCredential
		// Native providers own their refresh cache; CLI snapshot resolvers must be evaluated again.
		if !reusable || credential == nil || !reflect.DeepEqual(profile, cachedProfile) {
			credential, err = resolveMCPProfileCredential(ctx, profile)
			if err != nil {
				return nil, err
			}
			cachedProfile, cachedCredential = profile, nil
			if reusable {
				cachedCredential = credential
			}
		}
		model, err := credential.GetCredential()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve MCP credential: %w", err)
		}
		if err := validateMCPCredential(model); err != nil {
			return nil, err
		}
		return model, nil
	}
}

func validateMCPProfileMode(mode config.AuthenticateMode) error {
	switch mode {
	case config.AK, config.StsToken, config.RamRoleArn, config.EcsRamRole,
		config.RsaKeyPair, config.RamRoleArnWithEcs, config.ChainableRamRoleArn,
		config.External, config.CredentialsURI, config.OIDC, config.CloudSSO, config.OAuth:
		return nil
	default:
		return fmt.Errorf("profile mode %q is not supported for MCP credential authentication", mode)
	}
}

func resolveMCPProfileCredential(ctx *cli.Context, profile config.Profile) (credentialsv2.Credential, error) {
	if err := validateMCPProfileMode(profile.Mode); err != nil {
		return nil, err
	}
	if err := profile.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MCP credential profile: %w", err)
	}
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve MCP credential: %w", err)
	}
	if credential == nil {
		return nil, fmt.Errorf("MCP credential profile did not return an AK or STS credential")
	}
	return credential, nil
}

func validateMCPCredential(model *credentialsv2.CredentialModel) error {
	if model == nil {
		return fmt.Errorf("MCP credential provider returned no credential")
	}
	switch tea.StringValue(model.Type) {
	case "access_key", "rsa_key_pair":
	case "sts", "ram_role_arn", "ecs_ram_role", "oidc_role_arn":
		if tea.StringValue(model.SecurityToken) == "" {
			return fmt.Errorf("MCP STS credential is missing its security token")
		}
	default:
		return fmt.Errorf("credential type %q is not supported for MCP credential authentication", tea.StringValue(model.Type))
	}
	if tea.StringValue(model.AccessKeyId) == "" || tea.StringValue(model.AccessKeySecret) == "" {
		return fmt.Errorf("MCP credential is missing its access key ID or secret")
	}
	return nil
}
