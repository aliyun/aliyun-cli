package cloudsso

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
)

func installLoginTransport(t *testing.T, do func(*http.Request) (*http.Response, error)) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = &MockHTTPClient{DoFunc: do}
	t.Cleanup(func() { http.DefaultTransport = previous })
}

func TestClientIDEnvironmentPrecedence(t *testing.T) {
	t.Setenv("ALIBABACLOUD_SSO_CLIENT_ID", "")
	t.Setenv("ALIBABA_CLOUD_SSO_CLIENT_ID", "")
	if got := getClientId(); got != ClientId {
		t.Fatalf("default client ID = %q", got)
	}
	t.Setenv("ALIBABA_CLOUD_SSO_CLIENT_ID", "legacy-client")
	if got := getClientId(); got != "legacy-client" {
		t.Fatalf("legacy client ID = %q", got)
	}
	t.Setenv("ALIBABACLOUD_SSO_CLIENT_ID", "preferred-client")
	if got := getClientId(); got != "preferred-client" {
		t.Fatalf("preferred client ID = %q", got)
	}
}

func TestDeviceAuthorizationAndTokenRequests(t *testing.T) {
	t.Setenv("ALIBABACLOUD_SSO_CLIENT_ID", "custom-client")
	installLoginTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.Header.Get("Accept") != "application/json" || req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s, headers %v", req.Method, req.URL, req.Header)
		}
		var body map[string]string
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["ClientId"] != "custom-client" {
			t.Errorf("client ID = %q", body["ClientId"])
		}
		switch req.URL.Path {
		case "/device-authorization":
			if body["PortalUrl"] != "https://sso.example/start/login" || body["CodeChallenge"] != transformCodeVerifier(codeVerifier) || body["CodeChallengeMethod"] != "S256" {
				t.Errorf("device request = %v", body)
			}
			return mockHTTPResponse(200, `{"DeviceCode":"device","UserCode":"user","VerificationUri":"https://sso.example/verify","VerificationUriComplete":"https://sso.example/verify?code=user","Interval":2}`), nil
		case "/token":
			if body["DeviceCode"] != "device" || body["CodeVerifier"] != codeVerifier || body["GrantType"] != "urn:ietf:params:oauth:grant-type:device_code" {
				t.Errorf("token request = %v", body)
			}
			return mockHTTPResponse(200, `{"AccessToken":"access","ExpiresIn":3600}`), nil
		default:
			t.Fatalf("unexpected URL %s", req.URL)
			return nil, nil
		}
	})
	portal, _ := url.Parse("https://sso.example/start/login")
	device, err := startDeviceAuthorization(portal)
	if err != nil || device.DeviceCode != "device" || device.Interval != 2 || device.UserCode != "user" {
		t.Fatalf("device response = %#v, %v", device, err)
	}
	token, err := createAccessToken(portal.String(), device.DeviceCode)
	if err != nil || token.AccessToken != "access" || token.ExpiresIn != 3600 {
		t.Fatalf("token response = %#v, %v", token, err)
	}
	if got := transformCodeVerifier("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("RFC 7636 code challenge = %q", got)
	}
}

func TestLoginHTTPErrors(t *testing.T) {
	for _, operation := range []string{"device", "token"} {
		for _, tc := range []struct {
			name, body, want string
			status           int
			err              error
			readError        bool
		}{
			{name: "transport", err: errors.New("connection failed"), want: "connection failed"},
			{name: "body read", readError: true, want: "body read failed"},
			{name: "non JSON failure", status: 503, body: "service unavailable", want: "service unavailable"},
			{name: "API failure", status: 400, body: `{"ErrorCode":"AuthorizationPending","ErrorMessage":"wait","RequestId":"request"}`, want: "AuthorizationPending"},
			{name: "invalid success JSON", status: 200, body: "{", want: "unexpected end of JSON input"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				installLoginTransport(t, func(*http.Request) (*http.Response, error) {
					if tc.err != nil {
						return nil, tc.err
					}
					if tc.readError {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(iotest.ErrReader(errors.New("body read failed")))}, nil
					}
					return mockHTTPResponse(tc.status, tc.body), nil
				})
				var err error
				if operation == "device" {
					portal, _ := url.Parse("https://sso.example/start")
					_, err = startDeviceAuthorization(portal)
				} else {
					_, err = createAccessToken("https://sso.example/start", "device")
				}
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want containing %q", err, tc.want)
				}
			})
		}
	}
	if _, err := createAccessToken(":bad", "device"); err == nil {
		t.Fatal("invalid sign-in URL accepted")
	}
	if _, err := createAccessToken("https://sso.example\n", "device"); err == nil {
		t.Fatal("invalid host accepted")
	}
	if _, err := startDeviceAuthorization(&url.URL{Scheme: "http", Host: "bad\nhost"}); err == nil {
		t.Fatal("invalid device authorization host accepted")
	}
}

func TestGetAccessTokenPollsUntilAuthorized(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	polls := 0
	installLoginTransport(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/device-authorization" {
			return mockHTTPResponse(200, `{"DeviceCode":"device","UserCode":"user","VerificationUri":"https://sso.example/verify","Interval":0}`), nil
		}
		polls++
		if polls == 1 {
			return mockHTTPResponse(400, `{"ErrorCode":"AuthorizationPending"}`), nil
		}
		if polls == 2 {
			return mockHTTPResponse(503, `{"ErrorCode":"TransientError"}`), nil
		}
		return mockHTTPResponse(200, `{"AccessToken":"authorized","ExpiresIn":100}`), nil
	})
	token, err := (SsoLogin{SignInUrl: "https://sso.example/start"}).GetAccessToken()
	if err != nil || token.AccessToken != "authorized" || polls != 3 {
		t.Fatalf("GetAccessToken() = %#v, %v, polls %d", token, err, polls)
	}
}

func TestGetAccessTokenRejectsInvalidURLAndAuthorizationFailure(t *testing.T) {
	if _, err := (SsoLogin{SignInUrl: ":bad"}).GetAccessToken(); err == nil {
		t.Fatal("invalid sign-in URL accepted")
	}
	installLoginTransport(t, func(*http.Request) (*http.Response, error) {
		return mockHTTPResponse(403, `{"ErrorCode":"AccessDenied","ErrorMessage":"forbidden","RequestId":"request"}`), nil
	})
	if _, err := (SsoLogin{SignInUrl: "https://sso.example/start"}).GetAccessToken(); err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Fatalf("authorization failure = %v", err)
	}
}
