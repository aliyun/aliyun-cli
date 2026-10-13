package mcpproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/require"
)

func newMCPCredentialTestContext(t *testing.T) *cli.Context {
	t.Helper()
	setTestHome(t)
	for _, key := range []string{
		"ALIBABA_CLOUD_IGNORE_PROFILE", "ALIBABACLOUD_IGNORE_PROFILE",
		"ALIBABA_CLOUD_PROFILE", "ALIBABACLOUD_PROFILE", "ALICLOUD_PROFILE",
		"ALIBABA_CLOUD_ACCESS_KEY_ID", "ALIBABACLOUD_ACCESS_KEY_ID", "ALICLOUD_ACCESS_KEY_ID", "ACCESS_KEY_ID",
		"ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALIBABACLOUD_ACCESS_KEY_SECRET", "ALICLOUD_ACCESS_KEY_SECRET", "ACCESS_KEY_SECRET",
		"ALIBABA_CLOUD_SECURITY_TOKEN", "ALIBABACLOUD_SECURITY_TOKEN", "ALICLOUD_SECURITY_TOKEN", "SECURITY_TOKEN",
		"ALIBABA_CLOUD_CREDENTIALS_URI", "ALIBABA_CLOUD_STS_ENDPOINT", "ALIBABA_CLOUD_PROFILE_MODE",
		"ALIBABA_CLOUD_OIDC_PROVIDER_ARN", "ALIBABACLOUD_OIDC_PROVIDER_ARN",
		"ALIBABA_CLOUD_OIDC_TOKEN_FILE", "ALIBABACLOUD_OIDC_TOKEN_FILE",
		"ALIBABA_CLOUD_ROLE_ARN", "ALIBABACLOUD_ROLE_ARN", "ALIBABA_CLOUD_BEARER_TOKEN",
		"ALIBABA_CLOUD_ECS_METADATA_DISABLED", "HTTPS_PROXY", "https_proxy", config.EnvDisableExternalProcess,
	} {
		t.Setenv(key, "")
	}
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	ctx.SetInConfigureMode(true)
	return ctx
}

func writeMCPCredentialTestConfig(t *testing.T, ctx *cli.Context, profiles ...config.Profile) {
	t.Helper()
	require.NoError(t, config.SaveConfiguration(&config.Configuration{
		CurrentProfile: profiles[0].Name,
		Profiles:       profiles,
	}))
}

func mcpTestAKProfile(name, accessKey string) config.Profile {
	return config.Profile{
		Name: name, Mode: config.AK, RegionId: "cn-hangzhou",
		AccessKeyId: accessKey, AccessKeySecret: "test-secret",
	}
}

func TestCLICredentialSourceReloadsSelectedProfile(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	config.ProfileFlag(ctx.Flags()).SetAssigned(true)
	config.ProfileFlag(ctx.Flags()).SetValue("selected")
	t.Setenv("ALIBABA_CLOUD_PROFILE", "other")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "environment-key")
	source := newCLICredentialSource(ctx)
	for _, key := range []string{"first-key", "rotated-key"} {
		writeMCPCredentialTestConfig(t, ctx,
			mcpTestAKProfile("other", "other-key"), mcpTestAKProfile("selected", key))
		model, err := source()
		require.NoError(t, err)
		require.Equal(t, key, tea.StringValue(model.AccessKeyId))
	}
}

func TestCLICredentialSourceUsesSelectedProfileMode(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	profile := mcpTestAKProfile("selected", "selected-key")
	profile.Mode = config.StsToken
	profile.StsToken = "selected-token"
	writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "default-key"), profile)
	config.ProfileFlag(ctx.Flags()).SetAssigned(true)
	config.ProfileFlag(ctx.Flags()).SetValue("selected")
	model, err := newCLICredentialSource(ctx)()
	require.NoError(t, err)
	require.Equal(t, "sts", tea.StringValue(model.Type))
	require.Equal(t, "selected-key", tea.StringValue(model.AccessKeyId))
	require.Equal(t, "selected-token", tea.StringValue(model.SecurityToken))
}

func TestCLICredentialSourceUsesCLIProfileSelection(t *testing.T) {
	for _, tc := range []struct {
		name, environment, profile, selected string
		customPath                           bool
	}{
		{name: "current", selected: "current"},
		{name: "environment", environment: "ALIBABA_CLOUD_PROFILE", selected: "environment"},
		{name: "environment alias", environment: "ALIBABACLOUD_PROFILE", selected: "environment"},
		{name: "legacy environment alias", environment: "ALICLOUD_PROFILE", selected: "environment"},
		{name: "custom current", customPath: true, selected: "current"},
		{name: "custom environment", customPath: true, environment: "ALIBABA_CLOUD_PROFILE", selected: "environment"},
		{name: "custom explicit profile", customPath: true, environment: "ALIBABA_CLOUD_PROFILE", profile: "selected", selected: "selected"},
	} {
		for _, mode := range []config.AuthenticateMode{config.AK, config.StsToken} {
			t.Run(tc.name+"/"+string(mode), func(t *testing.T) {
				ctx := newMCPCredentialTestContext(t)
				if tc.environment != "" {
					t.Setenv(tc.environment, "environment")
				}
				if tc.profile != "" {
					setMCPCommandTestFlag(ctx, "profile", tc.profile)
				}
				prefix := "default-"
				if tc.customPath {
					setMCPCommandTestFlag(ctx, "config-path", filepath.Join(t.TempDir(), "config.json"))
					writeMCPCredentialTestConfig(t, ctx,
						mcpTestAKProfile("current", "default-current"),
						mcpTestAKProfile("environment", "default-environment"),
						mcpTestAKProfile("selected", "default-selected"))
					prefix = "custom-"
				}
				source := newCLICredentialSource(ctx)
				for _, generation := range []string{"first", "rotated"} {
					profiles := make([]config.Profile, 0, 3)
					for _, name := range []string{"current", "environment", "selected"} {
						profile := mcpTestAKProfile(name, prefix+generation+"-"+name)
						profile.Mode = mode
						if mode == config.StsToken {
							profile.StsToken = generation + "-token"
						}
						profiles = append(profiles, profile)
					}
					require.NoError(t, config.SaveConfigurationWithContext(ctx, &config.Configuration{
						CurrentProfile: "current", Profiles: profiles,
					}))

					model, err := source()

					require.NoError(t, err)
					require.Equal(t, prefix+generation+"-"+tc.selected, tea.StringValue(model.AccessKeyId))
					require.Equal(t, "test-secret", tea.StringValue(model.AccessKeySecret))
					if mode == config.StsToken {
						require.Equal(t, "sts", tea.StringValue(model.Type))
						require.Equal(t, generation+"-token", tea.StringValue(model.SecurityToken))
					} else {
						require.Equal(t, "access_key", tea.StringValue(model.Type))
						require.Empty(t, tea.StringValue(model.SecurityToken))
					}
				}
			})
		}
	}
}

func TestCLICredentialSourceUsesEnvironmentWhenIgnoringProfiles(t *testing.T) {
	for _, environment := range []string{"ALIBABA_CLOUD_IGNORE_PROFILE", "ALIBABACLOUD_IGNORE_PROFILE"} {
		for _, mode := range []config.AuthenticateMode{config.AK, config.StsToken} {
			t.Run(environment+"/"+string(mode), func(t *testing.T) {
				ctx := newMCPCredentialTestContext(t)
				writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("selected", "stored-key"))
				setMCPCommandTestFlag(ctx, "profile", "selected")
				t.Setenv(environment, "TRUE")
				source := newCLICredentialSource(ctx)
				for _, generation := range []string{"first", "rotated"} {
					t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", generation+"-key")
					t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", generation+"-secret")
					if mode == config.StsToken {
						t.Setenv("ALIBABA_CLOUD_SECURITY_TOKEN", generation+"-token")
					}
					model, err := source()
					require.NoError(t, err)
					require.Equal(t, generation+"-key", tea.StringValue(model.AccessKeyId))
					require.Equal(t, generation+"-secret", tea.StringValue(model.AccessKeySecret))
					if mode == config.StsToken {
						require.Equal(t, "sts", tea.StringValue(model.Type))
						require.Equal(t, generation+"-token", tea.StringValue(model.SecurityToken))
					} else {
						require.Equal(t, "access_key", tea.StringValue(model.Type))
						require.Empty(t, tea.StringValue(model.SecurityToken))
					}
				}
			})
		}
	}
}

func TestCLICredentialSourceReloadsTemporaryProfiles(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.StsToken, config.CloudSSO, config.OAuth} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			profile := mcpTestAKProfile("default", "first-key")
			profile.Mode = mode
			profile.StsToken = "first-token"
			profile.StsExpiration = time.Now().Add(time.Hour).Unix()
			profile.CloudSSOSignInUrl = "https://cloudsso.invalid"
			profile.CloudSSOAccountId = "account"
			profile.CloudSSOAccessConfig = "access-config"
			profile.CloudSSOAccessTokenExpire = profile.StsExpiration
			profile.OAuthSiteType = "CN"
			source := newCLICredentialSource(ctx)
			for _, value := range []string{"first", "rotated"} {
				profile.AccessKeyId = value + "-key"
				profile.StsToken = value + "-token"
				writeMCPCredentialTestConfig(t, ctx, profile)
				model, err := source()
				require.NoError(t, err)
				require.Equal(t, value+"-key", tea.StringValue(model.AccessKeyId))
				require.Equal(t, value+"-token", tea.StringValue(model.SecurityToken))
			}
		})
	}
}

func TestCLICredentialSourceRefreshesURIAndCloudSSO(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.CredentialsURI, config.CloudSSO} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				response := map[string]any{
					"Code": "Success", "AccessKeyId": fmt.Sprintf("key-%d", calls),
					"AccessKeySecret": "secret", "SecurityToken": fmt.Sprintf("token-%d", calls),
					"Expiration": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
				}
				if mode == config.CloudSSO {
					require.Equal(t, "/cloud-credentials", r.URL.Path)
					response = map[string]any{"CloudCredential": response}
				}
				require.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer server.Close()
			profile := mcpTestAKProfile("default", "stale-key")
			profile.Mode = mode
			profile.CredentialsURI = server.URL
			profile.CloudSSOSignInUrl = server.URL
			profile.CloudSSOAccountId = "account"
			profile.CloudSSOAccessConfig = "access-config"
			profile.CloudSSOAccessTokenExpire = time.Now().Add(time.Hour).Unix()
			source := newCLICredentialSource(ctx)
			for call := 1; call <= 2; call++ {
				// Persist an expired STS profile to require another upstream credential refresh.
				writeMCPCredentialTestConfig(t, ctx, profile)
				model, err := source()
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("key-%d", call), tea.StringValue(model.AccessKeyId))
			}
			require.Equal(t, 2, calls)
		})
	}
}

func TestMCPCredentialProcessHelper(t *testing.T) {
	if os.Getenv("MCP_CREDENTIAL_PROCESS_HELPER") != "1" {
		return
	}
	data, err := os.ReadFile(os.Args[len(os.Args)-1])
	if err != nil {
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(data)
	os.Exit(0)
}

func TestCLICredentialSourceRerunsExternalProcess(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	t.Setenv("MCP_CREDENTIAL_PROCESS_HELPER", "1")
	fixture := filepath.Join(t.TempDir(), "credential.json")
	profile := mcpTestAKProfile("default", "unused")
	profile.Mode = config.External
	profile.ProcessCommand = strconv.Quote(os.Args[0]) + " -test.run=^TestMCPCredentialProcessHelper$ -- " + strconv.Quote(fixture)
	writeMCPCredentialTestConfig(t, ctx, profile)
	source := newCLICredentialSource(ctx)
	for _, key := range []string{"first", "rotated"} {
		data, err := json.Marshal(mcpTestAKProfile("default", key))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(fixture, data, 0600))
		model, err := source()
		require.NoError(t, err)
		require.Equal(t, key, tea.StringValue(model.AccessKeyId))
	}
	for _, data := range []string{
		`{"mode":"Anonymous"}`,
		`{"mode":"BearerToken","bearer_token":"test-bearer"}`,
	} {
		require.NoError(t, os.WriteFile(fixture, []byte(data), 0600))
		model, err := source()
		require.Error(t, err)
		require.Nil(t, model)
		require.NotContains(t, err.Error(), "test-bearer")
	}
}

func TestCLICredentialSourceResolvesRoleAndOIDC(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.RamRoleArn, config.ChainableRamRoleArn, config.OIDC} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.NoError(t, r.ParseForm())
				if mode == config.OIDC {
					require.Equal(t, fmt.Sprintf("oidc-%d", calls), r.Form.Get("OIDCToken"))
				} else {
					require.Equal(t, fmt.Sprintf("source-%d", calls), r.Form.Get("AccessKeyId"))
				}
				fmt.Fprintf(w, `{"Credentials":{"AccessKeyId":"role-%d","AccessKeySecret":"secret","SecurityToken":"token","Expiration":%q}}`,
					calls, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
			}))
			defer server.Close()
			previousTransport := http.DefaultTransport
			http.DefaultTransport = server.Client().Transport
			t.Cleanup(func() { http.DefaultTransport = previousTransport })
			profile := mcpTestAKProfile("default", "source-1")
			profile.Mode = mode
			profile.RamRoleArn = "acs:ram::123:role/test"
			profile.RoleSessionName = "test-session"
			profile.StsEndpoint = strings.TrimPrefix(server.URL, "https://")
			profile.SourceProfile = "source"
			profile.OIDCProviderARN = "acs:ram::123:oidc-provider/test"
			profile.OIDCTokenFile = filepath.Join(t.TempDir(), "oidc-token")
			source := newCLICredentialSource(ctx)
			for call := 1; call <= 2; call++ {
				profile.AccessKeyId = fmt.Sprintf("source-%d", call)
				require.NoError(t, os.WriteFile(profile.OIDCTokenFile, []byte(fmt.Sprintf("oidc-%d", call)), 0600))
				writeMCPCredentialTestConfig(t, ctx, profile, mcpTestAKProfile("source", profile.AccessKeyId))
				model, err := source()
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("role-%d", call), tea.StringValue(model.AccessKeyId))
				if mode != config.ChainableRamRoleArn {
					cached, err := source()
					require.NoError(t, err)
					require.Equal(t, model, cached)
					require.Equal(t, call, calls)
				}
			}
			require.Equal(t, 2, calls)
		})
	}
}

func TestCLICredentialSourceUsesSDKRoleRefresh(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.RamRoleArn, config.OIDC} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.NoError(t, r.ParseForm())
				if mode == config.OIDC {
					require.Equal(t, fmt.Sprintf("oidc-%d", calls), r.Form.Get("OIDCToken"))
				}
				validity := time.Hour
				if calls == 1 {
					validity = time.Minute
				}
				fmt.Fprintf(w, `{"Credentials":{"AccessKeyId":"role-%d","AccessKeySecret":"secret","SecurityToken":"token","Expiration":%q}}`,
					calls, time.Now().Add(validity).UTC().Format(time.RFC3339))
			}))
			defer server.Close()
			previousTransport := http.DefaultTransport
			http.DefaultTransport = server.Client().Transport
			t.Cleanup(func() { http.DefaultTransport = previousTransport })
			profile := mcpTestAKProfile("default", "source")
			profile.Mode = mode
			profile.RamRoleArn = "acs:ram::123:role/test"
			profile.RoleSessionName = "test-session"
			profile.StsEndpoint = strings.TrimPrefix(server.URL, "https://")
			profile.OIDCProviderARN = "acs:ram::123:oidc-provider/test"
			profile.OIDCTokenFile = filepath.Join(t.TempDir(), "oidc-token")
			writeMCPCredentialTestConfig(t, ctx, profile)
			source := newCLICredentialSource(ctx)
			for _, expected := range []int{1, 2, 2} {
				require.NoError(t, os.WriteFile(profile.OIDCTokenFile, []byte(fmt.Sprintf("oidc-%d", expected)), 0600))
				model, err := source()
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("role-%d", expected), tea.StringValue(model.AccessKeyId))
				require.Equal(t, expected, calls)
			}
		})
	}
}

func TestCLICredentialSourceRefreshesECSRoles(t *testing.T) {
	for _, mode := range []config.AuthenticateMode{config.EcsRamRole, config.RamRoleArnWithEcs} {
		t.Run(string(mode), func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			metadataCalls, stsCalls := 0, 0
			metadata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/latest/api/token" {
					fmt.Fprint(w, "metadata-session-token")
					return
				}
				require.Equal(t, "/latest/meta-data/ram/security-credentials/test-role", r.URL.Path)
				require.Equal(t, "metadata-session-token", r.Header.Get("x-aliyun-ecs-metadata-token"))
				metadataCalls++
				validity := time.Hour
				if mode == config.EcsRamRole && metadataCalls == 1 {
					validity = time.Minute
				}
				fmt.Fprintf(w, `{"Code":"Success","AccessKeyId":"ecs-%d","AccessKeySecret":"secret","SecurityToken":"ecs-token-%d","Expiration":%q}`,
					metadataCalls, metadataCalls, time.Now().Add(validity).UTC().Format(time.RFC3339))
			}))
			defer metadata.Close()
			sts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				stsCalls++
				require.NoError(t, r.ParseForm())
				require.Equal(t, fmt.Sprintf("ecs-%d", stsCalls), r.Form.Get("AccessKeyId"))
				require.Equal(t, fmt.Sprintf("ecs-token-%d", stsCalls), r.Form.Get("SecurityToken"))
				fmt.Fprintf(w, `{"Credentials":{"AccessKeyId":"final-%d","AccessKeySecret":"secret","SecurityToken":"token","Expiration":%q}}`,
					stsCalls, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
			}))
			defer sts.Close()
			metadataURL, err := url.Parse(metadata.URL)
			require.NoError(t, err)
			stsHost := strings.TrimPrefix(sts.URL, "https://")
			transport := sts.Client().Transport.(*http.Transport).Clone()
			transport.Proxy = func(r *http.Request) (*url.URL, error) {
				if r.URL.Host == "100.100.100.200" {
					return metadataURL, nil
				}
				if r.URL.Host == stsHost {
					return nil, nil
				}
				return nil, fmt.Errorf("unexpected credential request target")
			}
			previousTransport := http.DefaultTransport
			http.DefaultTransport = transport
			t.Cleanup(func() {
				transport.CloseIdleConnections()
				http.DefaultTransport = previousTransport
			})
			profile := mcpTestAKProfile("default", "unused")
			profile.Mode = mode
			profile.RamRoleName = "test-role"
			profile.RamRoleArn = "acs:ram::123:role/test"
			profile.RoleSessionName = "test-session"
			profile.StsEndpoint = stsHost
			writeMCPCredentialTestConfig(t, ctx, profile)
			source := newCLICredentialSource(ctx)
			for call := 1; call <= 2; call++ {
				model, err := source()
				require.NoError(t, err)
				prefix := "ecs"
				if mode == config.RamRoleArnWithEcs {
					prefix = "final"
				}
				require.Equal(t, fmt.Sprintf("%s-%d", prefix, call), tea.StringValue(model.AccessKeyId))
			}
			if mode == config.EcsRamRole {
				cached, err := source()
				require.NoError(t, err)
				require.Equal(t, "ecs-2", tea.StringValue(cached.AccessKeyId))
			}
			require.Equal(t, 2, metadataCalls)
			if mode == config.RamRoleArnWithEcs {
				require.Equal(t, 2, stsCalls)
			}
		})
	}
}

func TestCLICredentialSourceRSAHelper(t *testing.T) {
	if os.Getenv("MCP_CREDENTIAL_RSA_HELPER") != "1" {
		return
	}
	certificate, err := os.ReadFile(os.Getenv("MCP_CREDENTIAL_RSA_CERT"))
	require.NoError(t, err)
	roots := x509.NewCertPool()
	require.True(t, roots.AppendCertsFromPEM(certificate))
	x509.SetFallbackRoots(roots)
	ctx := cli.NewCommandContext(io.Discard, io.Discard)
	config.AddFlags(ctx.Flags())
	source := newCLICredentialSource(ctx)
	for _, call := range []int{1, 2, 2} {
		model, err := source()
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("rsa-%d", call), tea.StringValue(model.AccessKeyId))
		require.Equal(t, "rsa_key_pair", tea.StringValue(model.Type))
		require.Empty(t, tea.StringValue(model.SecurityToken))
	}
}

func TestCLICredentialSourceRefreshesRSAKeyPair(t *testing.T) {
	ctx := newMCPCredentialTestContext(t)
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "GenerateSessionAccessKey", r.URL.Query().Get("Action"))
		require.NotEmpty(t, r.URL.Query().Get("Signature"))
		validity := time.Hour
		if calls == 1 {
			validity = time.Second
		}
		fmt.Fprintf(w, `{"SessionAccessKey":{"SessionAccessKeyId":"rsa-%d","SessionAccessKeySecret":"secret","Expiration":%q}}`,
			calls, time.Now().Add(validity).UTC().Format(time.RFC3339))
	}))
	defer server.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privateKey, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "private-key.pem")
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey}), 0600))
	certificatePath := filepath.Join(t.TempDir(), "certificate.pem")
	require.NoError(t, os.WriteFile(certificatePath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600))
	profile := mcpTestAKProfile("default", "unused")
	profile.Mode = config.RsaKeyPair
	profile.PrivateKey = keyPath
	profile.KeyPairName = "test-key-pair"
	profile.StsEndpoint = strings.TrimPrefix(server.URL, "https://")
	writeMCPCredentialTestConfig(t, ctx, profile)
	// The legacy RSA provider creates its own transport; keep its test trust roots in a subprocess.
	command := exec.Command(os.Args[0], "-test.run=^TestCLICredentialSourceRSAHelper$")
	command.Env = append(os.Environ(), "MCP_CREDENTIAL_RSA_HELPER=1",
		"MCP_CREDENTIAL_RSA_CERT="+certificatePath,
		"GODEBUG="+os.Getenv("GODEBUG")+",x509usefallbackroots=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Equal(t, 2, calls)
}

func TestCLICredentialSourceErrors(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		_, err := newCLICredentialSource(nil)()
		require.ErrorContains(t, err, "CLI context is required")
	})
	t.Run("missing selected profile", func(t *testing.T) {
		ctx := newMCPCredentialTestContext(t)
		writeMCPCredentialTestConfig(t, ctx, mcpTestAKProfile("default", "key"))
		config.ProfileFlag(ctx.Flags()).SetAssigned(true)
		config.ProfileFlag(ctx.Flags()).SetValue("missing")
		_, err := newCLICredentialSource(ctx)()
		require.ErrorContains(t, err, "failed to load MCP credential profile")
	})
	for _, mode := range []config.AuthenticateMode{config.BearerToken, config.Anonymous, "FutureMode"} {
		t.Run(string(mode), func(t *testing.T) {
			profile := mcpTestAKProfile("default", "key")
			profile.Mode = mode
			_, err := resolveMCPProfileCredential(nil, profile)
			require.ErrorContains(t, err, "not supported")
		})
	}
	for _, mode := range []config.AuthenticateMode{config.EcsRamRole, config.RamRoleArnWithEcs} {
		t.Run(string(mode)+" disabled metadata", func(t *testing.T) {
			ctx := newMCPCredentialTestContext(t)
			t.Setenv("ALIBABA_CLOUD_ECS_METADATA_DISABLED", "true")
			profile := mcpTestAKProfile("default", "key")
			profile.Mode = mode
			_, err := resolveMCPProfileCredential(ctx, profile)
			require.ErrorContains(t, err, "IMDS credentials is disabled")
		})
	}
	t.Run("RSA key source failure", func(t *testing.T) {
		ctx := newMCPCredentialTestContext(t)
		profile := mcpTestAKProfile("default", "key")
		profile.Mode = config.RsaKeyPair
		profile.PrivateKey = filepath.Join(t.TempDir(), "missing-private-key")
		profile.KeyPairName = "key-pair"
		_, err := resolveMCPProfileCredential(ctx, profile)
		require.ErrorContains(t, err, "failed to resolve MCP credential")
		require.ErrorContains(t, err, "PrivateKeyFile")
	})
	t.Run("incomplete STS profile", func(t *testing.T) {
		profile := mcpTestAKProfile("default", "key")
		profile.Mode = config.StsToken
		_, err := resolveMCPProfileCredential(nil, profile)
		require.ErrorContains(t, err, "invalid MCP credential profile")
	})
}

func TestValidateMCPCredential(t *testing.T) {
	for _, credentialType := range []string{"access_key", "rsa_key_pair", "sts", "ram_role_arn", "ecs_ram_role", "oidc_role_arn"} {
		t.Run(credentialType, func(t *testing.T) {
			model := (&credentialsv2.CredentialModel{}).SetType(credentialType).SetAccessKeyId("key").SetAccessKeySecret("secret")
			if credentialType != "access_key" && credentialType != "rsa_key_pair" {
				require.ErrorContains(t, validateMCPCredential(model), "security token")
				model.SetSecurityToken("token")
			}
			require.NoError(t, validateMCPCredential(model))
			model.AccessKeyId = nil
			require.ErrorContains(t, validateMCPCredential(model), "access key ID or secret")
			model.SetAccessKeyId("key").SetAccessKeySecret("")
			require.ErrorContains(t, validateMCPCredential(model), "access key ID or secret")
		})
	}
	require.ErrorContains(t, validateMCPCredential(nil), "no credential")
	for _, credentialType := range []string{"", "bearer", "Anonymous", "unknown"} {
		require.ErrorContains(t, validateMCPCredential((&credentialsv2.CredentialModel{}).SetType(credentialType)), "not supported")
	}
}
