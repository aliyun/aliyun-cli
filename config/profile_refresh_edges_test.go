package config

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/stretchr/testify/require"
)

func TestCredentialRefreshPropagatesConfigurationFailures(t *testing.T) {
	oldLoad, oldSave, oldExchange, oldRefresh := hookLoadOrCreateConfiguration, saveConfigurationFunc, exchangeFromOAuthFunc, tryRefreshStsTokenFunc
	t.Cleanup(func() {
		hookLoadOrCreateConfiguration, saveConfigurationFunc, exchangeFromOAuthFunc, tryRefreshStsTokenFunc = oldLoad, oldSave, oldExchange, oldRefresh
	})
	cause := errors.New("configuration unavailable")
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	AddFlags(ctx.Flags())
	for _, mode := range []AuthenticateMode{OAuth, CloudSSO} {
		for _, stage := range []string{"load", "save", "persist matching profile"} {
			t.Run(string(mode)+"/"+stage, func(t *testing.T) {
				profile := &Profile{Name: "selected", Mode: mode, CloudSSOSignInUrl: "https://sso.example.invalid", CloudSSOAccountId: "account", CloudSSOAccessConfig: "config", CloudSSOAccessTokenExpire: 9999999999}
				exchangeFromOAuthFunc = func(_ io.Writer, p *Profile) error {
					p.AccessKeyId, p.AccessKeySecret, p.StsToken = "id", "secret", "token"
					p.StsExpiration = 9999999999
					return nil
				}
				tryRefreshStsTokenFunc = func(*string, *string, *string, *string, *http.Client) (*cloudsso.CloudCredentialResponse, error) {
					return &cloudsso.CloudCredentialResponse{AccessKeyId: "id", AccessKeySecret: "secret", SecurityToken: "token", ExpirationInt64: 9999999999}, nil
				}
				hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
					return func(string) (*Configuration, error) {
						if stage == "load" {
							return nil, cause
						}
						stored := *profile
						stored.Endpoint = "stored-endpoint"
						stored.AccessKeyId, stored.AccessKeySecret, stored.StsToken, stored.StsExpiration = "", "", "", 0
						return &Configuration{Profiles: []Profile{{Name: "unrelated"}, stored}}, nil
					}
				}
				saves := 0
				saveConfigurationFunc = func(saveCtx *cli.Context, conf *Configuration) error {
					require.Same(t, ctx, saveCtx)
					saves++
					if stage == "save" {
						return cause
					}
					stored, ok := conf.GetProfile("selected")
					require.True(t, ok)
					require.Equal(t, "id", stored.AccessKeyId)
					require.Equal(t, "stored-endpoint", stored.Endpoint)
					require.Empty(t, conf.Profiles[0].AccessKeyId)
					return nil
				}
				credential, err := profile.GetCredential(ctx, nil)
				if stage == "persist matching profile" {
					require.NoError(t, err)
					require.NotNil(t, credential)
					require.Equal(t, 1, saves)
				} else {
					require.ErrorIs(t, err, cause)
					require.Nil(t, credential)
				}
			})
		}
	}
}

func TestCredentialConstructorsAndChainSourceFailures(t *testing.T) {
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	for _, profile := range []*Profile{
		{Name: "ecs", Mode: EcsRamRole, RamRoleName: "role"},
		{Name: "role", Mode: RamRoleArn, AccessKeyId: "id", AccessKeySecret: "secret", StsToken: "token", RamRoleArn: "acs:ram::123:role/test", RoleSessionName: "session"},
		{Name: "rsa", Mode: RsaKeyPair, PrivateKey: "missing-key-file", KeyPairName: "key", ExpiredSeconds: 900},
	} {
		t.Run(string(profile.Mode), func(t *testing.T) {
			credential, err := profile.GetCredential(ctx, nil)
			if profile.Mode == RsaKeyPair {
				require.ErrorContains(t, err, "Can not open PrivateKeyFile")
				require.Nil(t, credential)
			} else {
				require.NoError(t, err)
				require.NotNil(t, credential)
			}
		})
	}
	chain := &Profile{Name: "chain", Mode: ChainableRamRoleArn, SourceProfile: "source", parent: &Configuration{}}
	_, err := chain.GetCredential(ctx, nil)
	require.ErrorContains(t, err, "can not load the source profile")
	chain.parent.Profiles = []Profile{{Name: "source", Mode: AK}}
	_, err = chain.GetCredential(ctx, nil)
	require.ErrorContains(t, err, "AccessKeyId/AccessKeySecret is empty")
	profile := &Profile{Name: "infer", CloudSSOSignInUrl: "https://sso.example.invalid"}
	AutoModeRecognition(profile)
	require.Equal(t, CloudSSO, profile.Mode)
	profile = &Profile{Mode: BearerToken, RegionId: "cn-hangzhou", BearerTokenValue: "token", BearerTokenHeaderKey: "X-凭证"}
	require.ErrorContains(t, profile.Validate(), "printable ASCII")
}
