package mcpproxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

type sdkHTTPHandler func(*http.Request) (*http.Response, error)

func (f sdkHTTPHandler) Call(req *http.Request, _ *http.Transport) (*http.Response, error) {
	return f(req)
}

func installAPIHandler(t *testing.T, handler sdkHTTPHandler) {
	t.Helper()
	original := newOpenAPIClient
	newOpenAPIClient = func(ctx *cli.Context, profile config.Profile, endpoint string) (*openapiClient.Client, error) {
		client, err := original(ctx, profile, endpoint)
		if err == nil {
			client.HttpClient = handler
		}
		return client, err
	}
	t.Cleanup(func() { newOpenAPIClient = original })
}

func testCLIProfile(t *testing.T) (*cli.Context, config.Profile) {
	t.Helper()
	setTestHome(t)
	t.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
	t.Setenv("ALIBABACLOUD_IGNORE_PROFILE", "")
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	profile := config.NewProfile("default")
	profile.Mode, profile.RegionId = config.AK, "cn-hangzhou"
	profile.AccessKeyId, profile.AccessKeySecret = "test-key", "test-secret"
	require.NoError(t, os.MkdirAll(config.GetConfigPath(), 0700))
	require.NoError(t, config.SaveConfiguration(&config.Configuration{CurrentProfile: "default", Profiles: []config.Profile{profile}}))
	return ctx, profile
}

func sdkJSONResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func sdkAction(req *http.Request) string {
	for name, values := range req.Header {
		if strings.EqualFold(name, "x-acs-action") && len(values) > 0 {
			return values[0]
		}
	}
	return req.URL.Query().Get("Action")
}

const applicationJSON = `{"AppId":"app-id","AppName":"aliyun-cli-mcp-proxy","DisplayName":"MCP test","AppType":"NativeApp","AccessTokenValidity":1000,"RefreshTokenValidity":9000,"RedirectUris":{"RedirectUri":["http://127.0.0.1:8088/callback"]},"DelegatedScope":{"PredefinedScopes":{"PredefinedScope":[{"Name":"/acs/mcp-server"},{"Name":"other"}]}}}`

func TestOAuthApplicationAPIsMapRequestsAndResponses(t *testing.T) {
	ctx, profile := testCLIProfile(t)
	installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
		require.NotEmpty(t, req.Header.Get("Authorization"))
		action := sdkAction(req)
		switch action {
		case "GetApplication":
			require.Equal(t, "app-id", req.URL.Query().Get("AppId"))
			return sdkJSONResponse(`{"Application":` + applicationJSON + `}`), nil
		case "ListApplications":
			return sdkJSONResponse(`{"Applications":{"Application":[{"AppName":"other"},` + applicationJSON + `]}}`), nil
		case "CreateApplication":
			require.Equal(t, "NativeApp", req.URL.Query().Get("AppType"))
			require.Equal(t, "/acs/mcp-server", req.URL.Query().Get("PredefinedScopes"))
			require.Equal(t, "http://127.0.0.1:8088/callback", req.URL.Query().Get("RedirectUris"))
			return sdkJSONResponse(`{"Application":` + applicationJSON + `}`), nil
		default:
			t.Fatalf("unexpected action %q", action)
			return nil, nil
		}
	})
	byID, err := findOAuthApplicationById(ctx, profile, "app-id", RegionCN)
	require.NoError(t, err)
	require.Equal(t, []string{"/acs/mcp-server", "other"}, byID.Scopes)
	require.Equal(t, []string{"http://127.0.0.1:8088/callback"}, byID.RedirectUris)
	require.Equal(t, 1000, byID.AccessTokenValidity)
	byName, err := findOAuthApplicationByName(ctx, profile, RegionCN, MCPOAuthAppName)
	require.NoError(t, err)
	require.Equal(t, byID, byName)
	missing, err := findOAuthApplicationByName(ctx, profile, RegionCN, "missing")
	require.NoError(t, err)
	require.Nil(t, missing)
	created, err := createDefaultMCPOauthApplication(ctx, profile, RegionCN, "127.0.0.1", 8088, "/acs/mcp-server")
	require.NoError(t, err)
	require.Equal(t, byID, created)
}

func TestOAuthApplicationAPIFailures(t *testing.T) {
	for _, failure := range []string{"credential", "network", "invalid JSON"} {
		t.Run(failure, func(t *testing.T) {
			ctx, profile := testCLIProfile(t)
			if failure == "credential" {
				profile.AccessKeyId, profile.AccessKeySecret = "", ""
			}
			installAPIHandler(t, func(*http.Request) (*http.Response, error) {
				if failure == "network" {
					return nil, errors.New("network unavailable")
				}
				return sdkJSONResponse(`{"Application":[],"Applications":[]}`), nil
			})
			_, err := findOAuthApplicationById(ctx, profile, "app", RegionCN)
			require.Error(t, err)
			_, err = findOAuthApplicationByName(ctx, profile, RegionCN, "app")
			require.Error(t, err)
			_, err = createDefaultMCPOauthApplication(ctx, profile, RegionCN, "localhost", 8088, "scope")
			require.Error(t, err)
		})
	}
}

func TestListMCPServersAndStartValidation(t *testing.T) {
	ctx, _ := testCLIProfile(t)
	body := `{"apiMcpServers":[{"id":"id","name":"name","sourceType":"api","product":"ecs","urls":{"mcp":"https://upstream/mcp/test","sse":"https://upstream/sse/test"}}]}`
	installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "ListApiMcpServers", sdkAction(req))
		require.Equal(t, "/apimcpservers", req.URL.Path)
		return sdkJSONResponse(body), nil
	})
	servers, err := ListMCPServers(ctx, RegionCN)
	require.NoError(t, err)
	require.Len(t, servers, 1)
	require.Equal(t, "ecs", servers[0].Product)
	body = `{"apiMcpServers":[]}`
	require.ErrorContains(t, startMCPProxy(ctx, ProxyConfig{RegionType: RegionCN}), "no MCP servers found")
	body = `{"apiMcpServers":"bad"}`
	_, err = ListMCPServers(ctx, RegionCN)
	require.Error(t, err)
	require.ErrorContains(t, startMCPProxy(ctx, ProxyConfig{RegionType: RegionCN}), "failed to list MCP servers")
}

func TestGetOrCreateProfileReusesValidatedApplication(t *testing.T) {
	ctx, _ := testCLIProfile(t)
	profile := &McpProfile{Name: "test", MCPOAuthSiteType: "CN", MCPOAuthAppName: MCPOAuthAppName, MCPOAuthAppId: "app-id", MCPOAuthAccessToken: "access", MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: 4102444800}
	require.NoError(t, saveMcpProfile(profile))
	installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "GetApplication", sdkAction(req))
		return sdkJSONResponse(`{"Application":` + applicationJSON + `}`), nil
	})
	result, err := getOrCreateMCPProfile(ctx, ProxyConfig{Host: "127.0.0.1", Port: 8088, RegionType: RegionCN, Scope: "/acs/mcp-server"})
	require.NoError(t, err)
	require.Equal(t, "access", result.MCPOAuthAccessToken)
	require.Equal(t, 1000, result.MCPOAuthAccessTokenValidity)
	require.Equal(t, 9000, result.MCPOAuthRefreshTokenValidity)
}

func TestGetOrCreateMCPProfileAuthorization(t *testing.T) {
	for _, tc := range []struct {
		name, wantError                                                      string
		missing, custom, invalidApp, emptyRefresh, oauthFailure, saveFailure bool
	}{
		{name: "existing application"},
		{name: "create default", missing: true},
		{name: "custom missing", custom: true, missing: true, wantError: "not found"},
		{name: "invalid application", invalidApp: true, wantError: "validation failed"},
		{name: "empty refresh token", emptyRefresh: true, wantError: "empty RefreshToken"},
		{name: "OAuth failure", oauthFailure: true, wantError: "OAuth login failed"},
		{name: "save failure", saveFailure: true, wantError: "failed to save mcp profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := testCLIProfile(t)
			authorizationInput(t, "authorization-code\n", true)
			previousMux := http.DefaultServeMux
			http.DefaultServeMux = http.NewServeMux()
			t.Cleanup(func() { http.DefaultServeMux = previousMux })
			appJSON := strings.ReplaceAll(applicationJSON, ":8088/callback", ":0/callback")
			if tc.invalidApp {
				appJSON = strings.ReplaceAll(appJSON, "NativeApp", "WebApp")
			}
			created := false
			installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
				switch sdkAction(req) {
				case "ListApplications":
					if tc.missing {
						return sdkJSONResponse(`{"Applications":{"Application":[]}}`), nil
					}
					return sdkJSONResponse(`{"Applications":{"Application":[` + appJSON + `]}}`), nil
				case "CreateApplication":
					created = true
					return sdkJSONResponse(`{"Application":` + appJSON + `}`), nil
				default:
					t.Fatalf("unexpected API %s", sdkAction(req))
					return nil, nil
				}
			})
			oauthServer(t, func(w http.ResponseWriter, _ *http.Request) {
				if tc.oauthFailure {
					w.WriteHeader(400)
					_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
					return
				}
				refresh := "refresh"
				if tc.emptyRefresh {
					refresh = ""
				}
				_, _ = io.WriteString(w, `{"access_token":"authorized","refresh_token":"`+refresh+`","expires_in":1000}`)
			})
			if tc.saveFailure {
				require.NoError(t, os.MkdirAll(getMCPConfigPath()+".tmp", 0700))
			}
			opts := ProxyConfig{Host: "127.0.0.1", Port: 0, RegionType: RegionCN, Scope: "/acs/mcp-server"}
			if tc.custom {
				opts.OAuthAppName = "custom-app"
			}
			result, err := getOrCreateMCPProfile(ctx, opts)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.missing, created)
			require.Equal(t, "authorized", result.MCPOAuthAccessToken)
			require.Equal(t, "refresh", result.MCPOAuthRefreshToken)
			require.Equal(t, 1000, result.MCPOAuthAccessTokenValidity)
			require.Equal(t, 9000, result.MCPOAuthRefreshTokenValidity)
			data, err := os.ReadFile(getMCPConfigPath())
			require.NoError(t, err)
			stored, err := NewMcpProfileFromBytes(data)
			require.NoError(t, err)
			require.Equal(t, result, stored)
		})
	}
}

func TestMCPCommandParsesAccessListsAndReusesProfile(t *testing.T) {
	ctx, _ := testCLIProfile(t)
	profile := &McpProfile{Name: "test", MCPOAuthSiteType: "CN", MCPOAuthAppName: MCPOAuthAppName, MCPOAuthAppId: "app-id", MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: 4102444800}
	require.NoError(t, saveMcpProfile(profile))
	cmd := NewMCPProxyCommand()
	for _, flag := range cmd.Flags().Flags() {
		ctx.Flags().Add(flag)
	}
	for name, value := range map[string]string{"allowed-servers": " first , , second ", "blocked-servers": " blocked , , /hidden ", "no-browser": "true"} {
		flag := ctx.Flags().Get(name)
		flag.SetValue(value)
		flag.SetAssigned(true)
	}
	installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
		if sdkAction(req) == "GetApplication" {
			return sdkJSONResponse(`{"Application":` + applicationJSON + `}`), nil
		}
		require.Equal(t, "ListApiMcpServers", sdkAction(req))
		return sdkJSONResponse(`{"apiMcpServers":[]}`), nil
	})
	require.ErrorContains(t, cmd.Run(ctx, nil), "no MCP servers found")
}

func TestSavedProfileRemoteValidationFailures(t *testing.T) {
	for _, failure := range []string{"network", "invalid application", "save"} {
		t.Run(failure, func(t *testing.T) {
			ctx, configProfile := testCLIProfile(t)
			profile := &McpProfile{Name: "test", MCPOAuthSiteType: "CN", MCPOAuthAppName: MCPOAuthAppName, MCPOAuthAppId: "app-id", MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: 4102444800}
			require.NoError(t, saveMcpProfile(profile))
			installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
				if failure == "network" {
					return nil, errors.New("offline")
				}
				body := applicationJSON
				if failure == "invalid application" {
					body = strings.ReplaceAll(body, "NativeApp", "WebApp")
				}
				return sdkJSONResponse(`{"Application":` + body + `}`), nil
			})
			opts := ProxyConfig{Host: "127.0.0.1", Port: 8088, RegionType: RegionCN, Scope: "/acs/mcp-server"}
			if failure == "save" {
				require.NoError(t, os.MkdirAll(getMCPConfigPath()+".tmp", 0700))
				_, err := getOrCreateMCPProfile(ctx, opts)
				require.ErrorContains(t, err, "failed to save mcp profile")
			} else {
				require.Nil(t, loadExistingMCPProfile(ctx, configProfile, opts, MCPOAuthAppName))
			}
		})
	}
}
