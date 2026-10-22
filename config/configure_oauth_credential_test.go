package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func TestOAuthCredentialProcessContract(t *testing.T) {
	for _, mode := range []AuthenticateMode{OAuth, AK, StsToken} {
		t.Run(string(mode), func(t *testing.T) {
			p := Profile{Name: "selected", Mode: mode, OAuthSiteType: "CN", AccessKeyId: "ak", AccessKeySecret: "secret", StsToken: "sts", StsExpiration: time.Now().Unix() + 3600, OAuthRefreshToken: "must-never-export", OAuthAccessToken: "also-private"}
			path := filepath.Join(t.TempDir(), "config.json")
			data, err := json.Marshal(Configuration{CurrentProfile: "selected", Profiles: []Profile{p}})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, data, 0600))
			var out, stderr bytes.Buffer
			ctx := cli.NewCommandContext(&out, &stderr)
			AddFlags(ctx.Flags())
			ConfigurePathFlag(ctx.Flags()).SetValue(path)
			ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
			err = NewConfigureOAuthCredentialCommand().Run(ctx, nil)
			if mode != OAuth {
				require.Error(t, err)
				require.Empty(t, out.String())
				return
			}
			require.NoError(t, err)
			var got map[string]interface{}
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			require.Len(t, got, 6)
			require.Equal(t, float64(1), got["Version"])
			require.Equal(t, "sts", got["Type"])
			require.NotContains(t, out.String(), "must-never-export")
			require.NotContains(t, out.String(), "also-private")
			require.Equal(t, "secret", got["AccessKeySecret"])
		})
	}
}

func TestOAuthCredentialCheckpointFailureAndResume(t *testing.T) {
	previous := exchangeFromOAuthFunc
	t.Cleanup(func() { exchangeFromOAuthFunc = previous })
	path := filepath.Join(t.TempDir(), "config.json")
	initial := Profile{Name: "chosen", Mode: OAuth, OAuthSiteType: "CN", OAuthRefreshToken: "old-refresh"}
	data, err := json.Marshal(Configuration{CurrentProfile: "chosen", Profiles: []Profile{initial}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
	var out, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&out, &stderr)
	AddFlags(ctx.Flags())
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	exchangeFromOAuthFunc = func(w io.Writer, p *Profile, checkpoint func() error) error {
		fmt.Fprintln(w, "private-response-body")
		p.OAuthRefreshToken = "rotated-refresh"
		p.OAuthAccessToken = "rotated-access"
		p.OAuthAccessTokenExpire = time.Now().Unix() + 3600
		require.NoError(t, checkpoint())
		return errors.New("private-response-body")
	}
	err = NewConfigureOAuthCredentialCommand().Run(ctx, nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-response-body")
	require.Empty(t, out.String())
	require.Empty(t, stderr.String())
	saved, err := LoadConfigurationFromFile(path)
	require.NoError(t, err)
	p, ok := saved.GetProfile("chosen")
	require.True(t, ok)
	require.Equal(t, "rotated-refresh", p.OAuthRefreshToken)
	exchangeFromOAuthFunc = func(w io.Writer, p *Profile, checkpoint func() error) error {
		require.Equal(t, "rotated-refresh", p.OAuthRefreshToken)
		p.AccessKeyId = "new-key"
		p.AccessKeySecret = "new-secret"
		p.StsToken = "new-sts"
		p.StsExpiration = time.Now().Unix() + 3600
		return nil
	}
	require.NoError(t, NewConfigureOAuthCredentialCommand().Run(ctx, nil))
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	require.Equal(t, "new-sts", result["SecurityToken"])
	require.NotContains(t, out.String(), "rotated-refresh")
	require.NotContains(t, out.String(), "rotated-access")
}

func TestOAuthCredentialExplicitProfileAndOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	expiry := time.Now().Add(time.Hour).Unix()
	selected := Profile{Name: "selected", Mode: OAuth, OAuthSiteType: "CN", AccessKeyId: "selected-key", AccessKeySecret: "selected-secret", StsToken: "selected-token", StsExpiration: expiry}
	data, err := json.Marshal(Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: AK}, selected}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))
	var out, stderr bytes.Buffer
	ctx := cli.NewCommandContext(&out, &stderr)
	AddFlags(ctx.Flags())
	ConfigurePathFlag(ctx.Flags()).SetValue(path)
	ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	ProfileFlag(ctx.Flags()).SetValue("selected")
	ProfileFlag(ctx.Flags()).SetAssigned(true)
	require.NoError(t, NewConfigureOAuthCredentialCommand().Run(ctx, nil))
	decoder := json.NewDecoder(&out)
	var got map[string]interface{}
	require.NoError(t, decoder.Decode(&got))
	require.Equal(t, map[string]interface{}{
		"Version": float64(1), "Type": "sts", "AccessKeyId": "selected-key",
		"AccessKeySecret": "selected-secret", "SecurityToken": "selected-token",
		"Expiration": time.Unix(expiry, 0).UTC().Format(time.RFC3339),
	}, got)
	require.Equal(t, io.EOF, decoder.Decode(&got))
	require.Empty(t, stderr.String())
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, after, "cached export must not rewrite the configuration")
}

func TestOAuthCredentialInputFailuresDoNotExposeConfiguration(t *testing.T) {
	for _, kind := range []string{"missing-file", "malformed-file", "missing-profile", "positional"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if kind != "missing-file" {
				data := []byte(`{"current":"default","profiles":[]}`)
				if kind == "malformed-file" {
					data = []byte(`{"private-secret":`)
				}
				require.NoError(t, os.WriteFile(path, data, 0600))
			}
			var out, stderr bytes.Buffer
			ctx := cli.NewCommandContext(&out, &stderr)
			AddFlags(ctx.Flags())
			ConfigurePathFlag(ctx.Flags()).SetValue(path)
			ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
			ProfileFlag(ctx.Flags()).SetValue("missing-private-profile")
			ProfileFlag(ctx.Flags()).SetAssigned(true)
			var args []string
			if kind == "positional" {
				args = []string{"private-argument"}
			}
			err := NewConfigureOAuthCredentialCommand().Run(ctx, args)
			require.Error(t, err)
			for _, secret := range []string{"private-secret", "missing-private-profile", "private-argument", path} {
				require.NotContains(t, err.Error(), secret)
			}
			require.Empty(t, out.String())
			require.Empty(t, stderr.String())
		})
	}
}
