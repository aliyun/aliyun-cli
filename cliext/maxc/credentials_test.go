package maxc

import (
	"bytes"
	"errors"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func newOriginCtx() *cli.Context {
	return cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
}

func withLoadProfileStub(t *testing.T, fn func(*cli.Context) (config.Profile, error)) {
	t.Helper()
	orig := loadProfileFunc
	t.Cleanup(func() { loadProfileFunc = orig })
	loadProfileFunc = fn
}

func TestInjectCreds_AK(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "default",
			Mode:            config.AK,
			AccessKeyId:     "ak-id",
			AccessKeySecret: "ak-secret",
			RegionId:        "cn-hangzhou",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"]; got != "ak-id" {
		t.Errorf("ACCESS_KEY_ID = %q", got)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_SECRET"]; got != "ak-secret" {
		t.Errorf("ACCESS_KEY_SECRET = %q", got)
	}
	if _, ok := c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"]; ok {
		t.Errorf("SECURITY_TOKEN must be absent for plain AK")
	}
	if got := c.envMap["MAXCOMPUTE_REGION"]; got != "cn-hangzhou" {
		t.Errorf("MAXCOMPUTE_REGION = %q", got)
	}
}

func TestInjectCreds_StsToken(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "sts-profile",
			Mode:            config.StsToken,
			AccessKeyId:     "sts-id",
			AccessKeySecret: "sts-secret",
			StsToken:        "sts-token-xyz",
			RegionId:        "cn-shanghai",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if got := c.envMap["ALIBABA_CLOUD_SECURITY_TOKEN"]; got != "sts-token-xyz" {
		t.Errorf("SECURITY_TOKEN = %q, want sts-token-xyz", got)
	}
	if got := c.envMap["ALIBABA_CLOUD_ACCESS_KEY_ID"]; got != "sts-id" {
		t.Errorf("ACCESS_KEY_ID = %q", got)
	}
}

func TestInjectCreds_NoProfile_Silent(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{}, errors.New("profile not found")
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Errorf("expected silent fallback (nil), got %v", err)
	}
	if len(c.envMap) != 0 {
		t.Errorf("envMap should be empty on profile miss, got %v", c.envMap)
	}
}

func TestInjectCreds_GetCredentialFails(t *testing.T) {
	// Unknown mode forces extractCredentials into the default branch where
	// Profile.GetCredential() is invoked — which fails fast for an unknown
	// authentication mode without any network access.
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name: "broken",
			Mode: config.AuthenticateMode("DefinitelyNotAValidMode"),
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	err := c.InjectAliyunCredentials(nil)
	if err == nil {
		t.Fatal("expected credential resolution error, got nil")
	}
}

func TestInjectCreds_NoRegion_DoesNotSetMAXCOMPUTE_REGION(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name:            "no-region",
			Mode:            config.AK,
			AccessKeyId:     "x",
			AccessKeySecret: "y",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx()}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if _, ok := c.envMap["MAXCOMPUTE_REGION"]; ok {
		t.Errorf("MAXCOMPUTE_REGION must not be set when profile.RegionId is empty")
	}
}

func TestInjectCreds_PreservesExistingEnvMap(t *testing.T) {
	withLoadProfileStub(t, func(*cli.Context) (config.Profile, error) {
		return config.Profile{
			Name: "ak", Mode: config.AK, AccessKeyId: "i", AccessKeySecret: "s",
		}, nil
	})

	c := &Context{originCtx: newOriginCtx(), envMap: map[string]string{"PREEXISTING": "v"}}
	if err := c.InjectAliyunCredentials(nil); err != nil {
		t.Fatalf("InjectAliyunCredentials: %v", err)
	}
	if c.envMap["PREEXISTING"] != "v" {
		t.Errorf("pre-existing env entries were dropped")
	}
}
