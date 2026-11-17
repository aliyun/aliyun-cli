// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcpproxy

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/dara"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type credentialRoundTripper func(*http.Request) (*http.Response, error)

func (f credentialRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testCredentialModel() *credentialsv2.CredentialModel {
	return &credentialsv2.CredentialModel{
		AccessKeyId:     dara.String("test-ak-id"),
		AccessKeySecret: dara.String("test-ak-secret"),
	}
}

func credentialResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestCredentialAuthSignedExchange(t *testing.T) {
	for _, tc := range []struct {
		name, host, clientID, securityToken string
		region                              RegionType
	}{
		{"CN AK", "ramoauth.aliyuncs.com", "4071151845732613353", "", RegionCN},
		{"CN STS", "ramoauth.aliyuncs.com", "4071151845732613353", "sts-token+/=&", RegionCN},
		{"INTL AK", "ramoauth.alibabacloudcs.com", "4195410055503316452", "", RegionINTL},
		{"INTL STS", "ramoauth.alibabacloudcs.com", "4195410055503316452", "sts-token+/=&", RegionINTL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := testCredentialModel()
			model.SecurityToken = dara.String(tc.securityToken)
			calls := 0
			auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) {
				calls++
				return model, nil
			}, tc.region)
			startedAt := time.Unix(1700000000, 0)
			auth.now = func() time.Time { return startedAt }
			auth.client.Transport = credentialRoundTripper(func(request *http.Request) (*http.Response, error) {
				assert.Equal(t, "POST", request.Method)
				assert.Equal(t, "https", request.URL.Scheme)
				assert.Equal(t, tc.host, request.URL.Host)
				assert.Equal(t, "/", request.URL.Path)
				query := request.URL.Query()
				assert.Equal(t, "GenerateAccessToken", query.Get("Action"))
				assert.Equal(t, "2026-04-21", query.Get("Version"))
				assert.Equal(t, tc.clientID, query.Get("ClientId"))
				assert.Equal(t, "/internal/acs/openapi", query.Get("Scope"))
				assert.Equal(t, "test-ak-id", query.Get("AccessKeyId"))
				assert.Equal(t, tc.securityToken, query.Get("SecurityToken"))
				assert.Equal(t, "HMAC-SHA1", query.Get("SignatureMethod"))
				assert.Equal(t, "1.0", query.Get("SignatureVersion"))
				assert.NotEmpty(t, query.Get("SignatureNonce"))
				assert.NotEmpty(t, query.Get("Timestamp"))
				assert.NotContains(t, request.URL.String(), "test-ak-secret")
				if tc.securityToken == "" {
					assert.False(t, query.Has("SecurityToken"))
				}
				actualSignature := query.Get("Signature")
				query.Del("Signature")
				canonical := strings.ReplaceAll(query.Encode(), "+", "%20")
				stringToSign := "POST&%2F&" + strings.ReplaceAll(url.QueryEscape(canonical), "+", "%20")
				mac := hmac.New(sha1.New, []byte("test-ak-secret&"))
				_, _ = mac.Write([]byte(stringToSign))
				assert.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), actualSignature)
				return credentialResponse(200, `{"Data":{"AccessToken":"mcp-access-token","ExpiresIn":3600}}`), nil
			})

			token, err := auth.GetToken()
			require.NoError(t, err)
			assert.Equal(t, "mcp-access-token", token)
			assert.Equal(t, 1, calls)
			expiresAt, failed := auth.Snapshot()
			assert.Equal(t, startedAt.Add(time.Hour), expiresAt)
			assert.False(t, failed)
			assert.Equal(t, 30*time.Second, auth.client.Timeout)
			assert.ErrorIs(t, auth.client.CheckRedirect(nil, nil), http.ErrUseLastResponse)
		})
	}
}

func TestCredentialAuthRefreshesExpiringTokenAndCredentials(t *testing.T) {
	now := time.Unix(1700000000, 0)
	sourceCalls, exchangeCalls := 0, 0
	auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) {
		sourceCalls++
		model := testCredentialModel()
		model.SecurityToken = dara.String(fmt.Sprintf("sts-token-%d", sourceCalls))
		return model, nil
	}, RegionCN)
	auth.now = func() time.Time { return now }
	auth.client.Transport = credentialRoundTripper(func(request *http.Request) (*http.Response, error) {
		exchangeCalls++
		assert.Equal(t, fmt.Sprintf("sts-token-%d", exchangeCalls), request.URL.Query().Get("SecurityToken"))
		return credentialResponse(200, fmt.Sprintf(`{"Data":{"AccessToken":"token-%d","ExpiresIn":3600}}`, exchangeCalls)), nil
	})

	token, err := auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token)
	now = now.Add(3539 * time.Second)
	token, err = auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token)
	assert.Equal(t, 1, sourceCalls)
	now = now.Add(time.Second)
	token, err = auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-2", token)
	assert.Equal(t, 2, sourceCalls)
	assert.Equal(t, 2, exchangeCalls)

	now = now.Add(2 * time.Hour)
	token, err = auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-3", token)
}

func TestCredentialAuthConcurrentAcquisitionAndRejectedTokenRefresh(t *testing.T) {
	var sourceCalls, exchangeCalls atomic.Int32
	auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) {
		sourceCalls.Add(1)
		return testCredentialModel(), nil
	}, RegionCN)
	auth.client.Transport = credentialRoundTripper(func(*http.Request) (*http.Response, error) {
		call := exchangeCalls.Add(1)
		return credentialResponse(200, fmt.Sprintf(`{"Data":{"AccessToken":"token-%d","ExpiresIn":3600}}`, call)), nil
	})
	for _, force := range []bool{false, true} {
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				var token string
				var err error
				if force {
					token, err = auth.RefreshToken("token-1")
					assert.Equal(t, "token-2", token)
				} else {
					token, err = auth.GetToken()
					assert.Equal(t, "token-1", token)
				}
				assert.NoError(t, err)
			}()
		}
		close(start)
		wg.Wait()
	}
	assert.EqualValues(t, 2, sourceCalls.Load())
	assert.EqualValues(t, 2, exchangeCalls.Load())
}

func TestCredentialAuthRefreshFailureDiscardsRejectedToken(t *testing.T) {
	auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) {
		return testCredentialModel(), nil
	}, RegionCN)
	calls := 0
	auth.client.Transport = credentialRoundTripper(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("secret-token-in-signed-request-url")
		}
		return credentialResponse(200, fmt.Sprintf(`{"Data":{"AccessToken":"token-%d","ExpiresIn":3600}}`, calls)), nil
	})
	token, err := auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-1", token)
	token, err = auth.RefreshToken("token-1")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.NotContains(t, err.Error(), "secret-token")
	expiresAt, failed := auth.Snapshot()
	assert.True(t, expiresAt.IsZero())
	assert.True(t, failed)
	token, err = auth.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "token-3", token)
	_, failed = auth.Snapshot()
	assert.False(t, failed)
}

func TestCredentialAuthRejectsInvalidCredentials(t *testing.T) {
	for _, tc := range []struct {
		name   string
		model  *credentialsv2.CredentialModel
		err    error
		source bool
		region RegionType
	}{
		{name: "source missing", region: RegionCN},
		{name: "source failed", source: true, err: errors.New("secret credential payload"), region: RegionCN},
		{name: "nil credentials", source: true, region: RegionCN},
		{name: "empty credentials", source: true, model: &credentialsv2.CredentialModel{}, region: RegionCN},
		{name: "no ID", source: true, model: &credentialsv2.CredentialModel{AccessKeySecret: dara.String("secret")}, region: RegionCN},
		{name: "no secret", source: true, model: &credentialsv2.CredentialModel{AccessKeyId: dara.String("id")}, region: RegionCN},
		{name: "blank ID", source: true, model: &credentialsv2.CredentialModel{AccessKeyId: dara.String("  "), AccessKeySecret: dara.String("secret")}, region: RegionCN},
		{name: "bearer", source: true, model: &credentialsv2.CredentialModel{AccessKeyId: dara.String("id"), AccessKeySecret: dara.String("secret"), BearerToken: dara.String("secret token")}, region: RegionCN},
		{name: "invalid region", source: true, model: testCredentialModel(), region: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source mcpCredentialSource
			if tc.source {
				source = func() (*credentialsv2.CredentialModel, error) { return tc.model, tc.err }
			}
			auth := newCredentialAuth(source, tc.region)
			auth.client.Transport = credentialRoundTripper(func(*http.Request) (*http.Response, error) {
				t.Error("invalid credentials must not reach the token endpoint")
				return nil, errors.New("unexpected request")
			})
			token, err := auth.GetToken()
			require.Error(t, err)
			assert.Empty(t, token)
			assert.NotContains(t, err.Error(), "secret credential payload")
			assert.NotContains(t, err.Error(), "secret token")
			_, failed := auth.Snapshot()
			assert.True(t, failed)
		})
	}
}

func TestCredentialAuthExchangeErrorsAreSafeAndNotRetried(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		err        error
	}{
		{"network", "", 0, errors.New("https://ramoauth.aliyuncs.com/?SecurityToken=secret-value")},
		{"forbidden", `{"Code":"Forbidden","Message":"secret-value","AccessToken":"secret-value"}`, 403, nil},
		{"server error", `{"Code":"InternalError","Message":"secret-value"}`, 500, nil},
		{"redirect", `{"Data":{"AccessToken":"secret-value","ExpiresIn":3600}}`, 302, nil},
		{"malformed JSON", `secret-value`, 200, nil},
		{"missing token", `{"Data":{"ExpiresIn":3600,"Message":"secret-value"}}`, 200, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) { return testCredentialModel(), nil }, RegionCN)
			calls := 0
			auth.client.Transport = credentialRoundTripper(func(*http.Request) (*http.Response, error) {
				calls++
				if tc.err != nil {
					return nil, tc.err
				}
				response := credentialResponse(tc.status, tc.body)
				if tc.status == 302 {
					response.Header.Set("Location", "https://unexpected.example/?secret=secret-value")
				}
				return response, nil
			})
			token, err := auth.GetToken()
			require.Error(t, err)
			assert.Empty(t, token)
			assert.NotContains(t, err.Error(), "secret-value")
			assert.Equal(t, 1, calls)
		})
	}
}

func TestParseCredentialToken(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"nested", `{"Data":{"AccessToken":"token","ExpiresIn":3600}}`, true},
		{"root", `{"AccessToken":"token","ExpiresIn":3600}`, true},
		{"snake case", `{"data":{"access_token":"token","expires_in":3600}}`, true},
		{"string expiry", `{"AccessToken":"token","ExpiresIn":"3600"}`, true},
		{"trim token", `{"AccessToken":" token ","ExpiresIn":3600}`, true},
		{"nil body", `null`, false},
		{"array body", `[]`, false},
		{"empty token", `{"AccessToken":"","ExpiresIn":3600}`, false},
		{"blank token", `{"AccessToken":"  ","ExpiresIn":3600}`, false},
		{"non string token", `{"AccessToken":12,"ExpiresIn":3600}`, false},
		{"token line break", `{"AccessToken":"token\r\ninject","ExpiresIn":3600}`, false},
		{"token whitespace", `{"AccessToken":"token\tinject","ExpiresIn":3600}`, false},
		{"missing expiry", `{"AccessToken":"token"}`, false},
		{"zero expiry", `{"AccessToken":"token","ExpiresIn":0}`, false},
		{"negative expiry", `{"AccessToken":"token","ExpiresIn":-1}`, false},
		{"fractional expiry", `{"AccessToken":"token","ExpiresIn":1.5}`, false},
		{"nonnumeric expiry", `{"AccessToken":"token","ExpiresIn":"soon"}`, false},
		{"object expiry", `{"AccessToken":"token","ExpiresIn":{}}`, false},
		{"overflow expiry", `{"AccessToken":"token","ExpiresIn":"9223372036854775807"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body interface{}
			require.NoError(t, json.Unmarshal([]byte(tc.body), &body))
			now := time.Unix(1700000000, 0)
			token, expiresAt, err := parseCredentialToken(map[string]interface{}{"body": body}, now)
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, "token", token)
				assert.Equal(t, now.Add(time.Hour), expiresAt)
			} else {
				require.Error(t, err)
				assert.Empty(t, token)
				assert.True(t, expiresAt.IsZero())
			}
		})
	}
}

func TestCredentialAuthExpiryIncludesAcquisitionTime(t *testing.T) {
	now := time.Unix(1700000000, 0)
	auth := newCredentialAuth(func() (*credentialsv2.CredentialModel, error) { return testCredentialModel(), nil }, RegionCN)
	auth.now = func() time.Time { return now }
	auth.client.Transport = credentialRoundTripper(func(*http.Request) (*http.Response, error) {
		now = now.Add(2 * time.Second)
		return credentialResponse(200, `{"Data":{"AccessToken":"token","ExpiresIn":1}}`), nil
	})
	expiresAt, failed := auth.Snapshot()
	assert.True(t, expiresAt.IsZero())
	assert.False(t, failed)
	token, err := auth.GetToken()
	require.ErrorContains(t, err, "expired during acquisition")
	assert.Empty(t, token)
	expiresAt, failed = auth.Snapshot()
	assert.True(t, expiresAt.IsZero())
	assert.True(t, failed)
}
