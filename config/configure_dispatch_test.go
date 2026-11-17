package config

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestInteractiveConfigureDispatchPreservesModeOnSaveFailure(t *testing.T) {
	for _, mode := range []AuthenticateMode{StsToken, RamRoleArn, EcsRamRole, RamRoleArnWithEcs, ChainableRamRoleArn, RsaKeyPair, External, CredentialsURI, OIDC} {
		t.Run(string(mode), func(t *testing.T) {
			previousLoad, previousSave, previousStdin := hookLoadOrCreateConfiguration, hookSaveConfigurationWithContext, stdin
			t.Cleanup(func() {
				hookLoadOrCreateConfiguration = previousLoad
				hookSaveConfigurationWithContext = previousSave
				stdin = previousStdin
			})
			stdin = strings.NewReader("")
			profile := NewProfile("selected")
			profile.Mode = AK
			profile.RegionId, profile.Language = "cn-hangzhou", "en"
			profile.RamRoleName, profile.RamRoleArn, profile.RoleSessionName = "role", "acs:ram::123:role/demo", "session"
			profile.ExpiredSeconds, profile.AccessKeyId, profile.AccessKeySecret = 900, "id", "secret"
			profile.ProcessCommand, profile.CredentialsURI, profile.SourceProfile = "credential-helper", "https://credentials.example", "source"
			hookLoadOrCreateConfiguration = func(func(string) (*Configuration, error)) func(string) (*Configuration, error) {
				return func(string) (*Configuration, error) {
					return &Configuration{CurrentProfile: "selected", Profiles: []Profile{profile}}, nil
				}
			}
			saveFailure := errors.New("disk full")
			hookSaveConfigurationWithContext = func(func(*cli.Context, *Configuration) error) func(*cli.Context, *Configuration) error {
				return func(_ *cli.Context, conf *Configuration) error {
					stored, ok := conf.GetProfile("selected")
					require.True(t, ok)
					require.Equal(t, mode, stored.Mode)
					require.Equal(t, "cn-hangzhou", stored.RegionId)
					require.Equal(t, "en", stored.Language)
					return saveFailure
				}
			}
			out := &bytes.Buffer{}
			ctx := cli.NewCommandContext(out, io.Discard)
			AddFlags(ctx.Flags())
			require.ErrorIs(t, doConfigure(ctx, "selected", string(mode)), saveFailure)
			require.Contains(t, out.String(), "Saving profile[selected]")
		})
	}
}
