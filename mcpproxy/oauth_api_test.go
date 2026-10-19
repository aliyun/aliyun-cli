package mcpproxy

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/require"
)

type sdkHTTPHandler func(*http.Request) (*http.Response, error)

var sdkTestRoots sync.Once

func oauthSDKSubprocess(t *testing.T) bool {
	t.Helper()
	if os.Getenv("MCP_TLS_TEST_CHILD") == "1" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	args := []string{"-test.run=^" + regexp.QuoteMeta(t.Name()) + "$"}
	if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
		args = append(args, "-test.gocoverdir="+dir.Value.String())
	}
	command := exec.CommandContext(ctx, os.Args[0], args...)
	command.Env = append(os.Environ(), "MCP_TLS_TEST_CHILD=1", "GODEBUG="+os.Getenv("GODEBUG")+",x509usefallbackroots=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, ctx.Err(), "%s", output)
	return true
}

func installAPIHandler(t *testing.T, handler sdkHTTPHandler) {
	t.Helper()
	require.Equal(t, "1", os.Getenv("MCP_TLS_TEST_CHILD"))
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		response, err := handler(request)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"Code": "ServiceUnavailable", "Message": err.Error(), "RequestId": "local-request"})
			return
		}
		defer response.Body.Close()
		for key, values := range response.Header {
			w.Header()[key] = values
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	}))
	sdkTestRoots.Do(func() {
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		x509.SetFallbackRoots(roots)
	})
	previous := EndpointMap[RegionCN]
	endpoint := previous
	endpoint.IMS, endpoint.MCP = server.Listener.Addr().String(), server.Listener.Addr().String()
	EndpointMap[RegionCN] = endpoint
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("https_proxy", "")
	t.Cleanup(func() { EndpointMap[RegionCN] = previous; server.Close() })
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
	if oauthSDKSubprocess(t) {
		return
	}
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
	if oauthSDKSubprocess(t) {
		return
	}
	for _, failure := range []string{"credential", "server", "invalid JSON"} {
		t.Run(failure, func(t *testing.T) {
			ctx, profile := testCLIProfile(t)
			if failure == "credential" {
				profile.AccessKeyId, profile.AccessKeySecret = "", ""
			}
			installAPIHandler(t, func(*http.Request) (*http.Response, error) {
				if failure == "server" {
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

func TestListMCPServersPreservesCredentialsAndServerFailures(t *testing.T) {
	if oauthSDKSubprocess(t) {
		return
	}
	ctx, profile := testCLIProfile(t)
	installAPIHandler(t, func(*http.Request) (*http.Response, error) {
		return nil, errors.New("MCP catalog unavailable")
	})
	_, err := ListMCPServers(ctx, RegionCN)
	require.ErrorContains(t, err, "MCP catalog unavailable")
	profile.AccessKeyId, profile.AccessKeySecret = "", ""
	require.NoError(t, config.SaveConfiguration(&config.Configuration{CurrentProfile: "default", Profiles: []config.Profile{profile}}))
	_, err = ListMCPServers(ctx, RegionCN)
	require.Error(t, err)
}

func TestListMCPServersAndStartValidation(t *testing.T) {
	if oauthSDKSubprocess(t) {
		return
	}
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
	if oauthSDKSubprocess(t) {
		return
	}
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
	if oauthSDKSubprocess(t) {
		return
	}
	for _, tc := range []struct {
		name, wantError                                                      string
		missing, custom, invalidApp, emptyRefresh, oauthFailure, saveFailure bool
		listFailure, createFailure                                           bool
	}{
		{name: "existing application"},
		{name: "create default", missing: true},
		{name: "custom missing", custom: true, missing: true, wantError: "not found"},
		{name: "invalid application", invalidApp: true, wantError: "validation failed"},
		{name: "empty refresh token", emptyRefresh: true, wantError: "empty RefreshToken"},
		{name: "OAuth failure", oauthFailure: true, wantError: "OAuth login failed"},
		{name: "save failure", saveFailure: true, wantError: "failed to save mcp profile"},
		{name: "list failure", listFailure: true, wantError: "failed to find OAuth application"},
		{name: "create failure", missing: true, createFailure: true, wantError: "failed to create default OAuth application"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := testCLIProfile(t)
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
					if tc.listFailure {
						return nil, errors.New("application list unavailable")
					}
					if tc.missing {
						return sdkJSONResponse(`{"Applications":{"Application":[]}}`), nil
					}
					return sdkJSONResponse(`{"Applications":{"Application":[` + appJSON + `]}}`), nil
				case "CreateApplication":
					if tc.createFailure {
						return nil, errors.New("application creation denied")
					}
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
			if !tc.invalidApp && !(tc.custom && tc.missing) && !tc.listFailure && !tc.createFailure {
				callbackAuthorizationRequest(t, "authorization-code")
			}
			opts := ProxyConfig{Host: "127.0.0.1", Port: 0, RegionType: RegionCN, Scope: "/acs/mcp-server", AutoOpenBrowser: true}
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
	if oauthSDKSubprocess(t) {
		return
	}
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
	if oauthSDKSubprocess(t) {
		return
	}
	for _, failure := range []string{"network", "missing application", "invalid application", "save"} {
		t.Run(failure, func(t *testing.T) {
			ctx, configProfile := testCLIProfile(t)
			profile := &McpProfile{Name: "test", MCPOAuthSiteType: "CN", MCPOAuthAppName: MCPOAuthAppName, MCPOAuthAppId: "app-id", MCPOAuthRefreshToken: "refresh", MCPOAuthRefreshTokenExpire: 4102444800}
			require.NoError(t, saveMcpProfile(profile))
			installAPIHandler(t, func(req *http.Request) (*http.Response, error) {
				if failure == "network" {
					return nil, errors.New("offline")
				}
				if failure == "missing application" {
					return sdkJSONResponse(`{}`), nil
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
