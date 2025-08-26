// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureOAuth_Success(t *testing.T) {
	// Mock stdin input
	originalStdin := stdin
	defer func() { stdin = originalStdin }()
	stdin = strings.NewReader("0\n") // Choose CN site type

	// Mock OAuth functions
	originalOauthStartOauthFlow := oauthStartOauthFlow
	originalOauthExchangeFromOAuth := oauthExchangeFromOAuth
	defer func() {
		oauthStartOauthFlow = originalOauthStartOauthFlow
		oauthExchangeFromOAuth = originalOauthExchangeFromOAuth
	}()

	oauthStartOauthFlow = func(w io.Writer, cp *Profile) error {
		// Mock successful OAuth flow
		cp.OAuthAccessToken = "mock_access_token"
		cp.OAuthRefreshToken = "mock_refresh_token"
		cp.OAuthAccessTokenExpire = 9999999999
		return nil
	}

	oauthExchangeFromOAuth = func(w io.Writer, cp *Profile) error {
		// Mock successful exchange
		cp.AccessKeyId = "mock_access_key_id"
		cp.AccessKeySecret = "mock_access_key_secret"
		cp.StsToken = "mock_sts_token"
		return nil
	}

	w := new(bytes.Buffer)
	cp := &Profile{
		Name: "test",
		Mode: OAuth,
	}

	err := configureOAuth(w, cp)
	assert.NoError(t, err)
	assert.Equal(t, "CN", cp.OAuthSiteType)
	assert.Equal(t, "mock_access_key_id", cp.AccessKeyId)
	assert.Equal(t, "mock_access_key_secret", cp.AccessKeySecret)
	assert.Equal(t, "mock_sts_token", cp.StsToken)
	assert.Contains(t, w.String(), "OAuth Site Type")
}

func TestConfigureOAuth_InvalidSiteType(t *testing.T) {
	originalStdin := stdin
	defer func() { stdin = originalStdin }()
	stdin = strings.NewReader("invalid\n")

	w := new(bytes.Buffer)
	cp := &Profile{
		Name:          "test",
		Mode:          OAuth,
		OAuthSiteType: "INVALID",
	}

	err := configureOAuth(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OAuth site type")
}

func TestConfigureOAuth_StartFlowError(t *testing.T) {
	originalStdin := stdin
	defer func() { stdin = originalStdin }()
	stdin = strings.NewReader("0\n")

	originalOauthStartOauthFlow := oauthStartOauthFlow
	defer func() {
		oauthStartOauthFlow = originalOauthStartOauthFlow
	}()

	oauthStartOauthFlow = func(w io.Writer, cp *Profile) error {
		return assert.AnError
	}

	w := new(bytes.Buffer)
	cp := &Profile{
		Name: "test",
		Mode: OAuth,
	}

	err := configureOAuth(w, cp)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestConfigureOAuth_ExchangeError(t *testing.T) {
	originalStdin := stdin
	defer func() { stdin = originalStdin }()
	stdin = strings.NewReader("0\n")

	originalOauthStartOauthFlow := oauthStartOauthFlow
	originalOauthExchangeFromOAuth := oauthExchangeFromOAuth
	defer func() {
		oauthStartOauthFlow = originalOauthStartOauthFlow
		oauthExchangeFromOAuth = originalOauthExchangeFromOAuth
	}()

	oauthStartOauthFlow = func(w io.Writer, cp *Profile) error {
		return nil
	}

	oauthExchangeFromOAuth = func(w io.Writer, cp *Profile) error {
		return assert.AnError
	}

	w := new(bytes.Buffer)
	cp := &Profile{
		Name: "test",
		Mode: OAuth,
	}

	err := configureOAuth(w, cp)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestStartOauthFlow_InvalidSiteType(t *testing.T) {
	w := new(bytes.Buffer)
	cp := &Profile{
		Name:          "test",
		Mode:          OAuth,
		OAuthSiteType: "INVALID",
	}

	err := startOauthFlow(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OAuth site type")
}

func TestExchangeFromOAuth_Success(t *testing.T) {
	// Create a mock HTTP server for exchange endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/exchange" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer mock_access_token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			exchangeResp := map[string]interface{}{
				"requestId":       "mock_request_id",
				"accessKeyId":     "mock_access_key_id",
				"accessKeySecret": "mock_access_key_secret",
				"expiration":      "2023-12-31T23:59:59Z",
				"securityToken":   "mock_security_token",
			}
			json.NewEncoder(w).Encode(exchangeResp)
		}
	}))
	defer server.Close()

	// Override OAuth URLs for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap
	defer func() {
		oauthBaseUrlMap = originalOauthBaseUrlMap
	}()
	oauthBaseUrlMap["CN"] = server.URL

	w := new(bytes.Buffer)
	cp := &Profile{
		Name:                   "test",
		Mode:                   OAuth,
		OAuthSiteType:          "CN",
		OAuthAccessToken:       "mock_access_token",
		OAuthAccessTokenExpire: 9999999999, // Not expired (far future)
		OAuthRefreshToken:      "mock_refresh_token",
	}

	err := exchangeFromOAuth(w, cp)
	assert.NoError(t, err)
	assert.Equal(t, "mock_access_key_id", cp.AccessKeyId)
	assert.Equal(t, "mock_access_key_secret", cp.AccessKeySecret)
	assert.Equal(t, "mock_security_token", cp.StsToken)
}

func TestExchangeFromOAuth_InvalidSiteType(t *testing.T) {
	w := new(bytes.Buffer)
	cp := &Profile{
		Name:          "test",
		Mode:          OAuth,
		OAuthSiteType: "INVALID",
	}

	err := exchangeFromOAuth(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OAuth site type")
}

func TestExchangeFromOAuth_TokenExpired_RefreshSuccess(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/token" {
			// Mock refresh token response
			tokenResp := map[string]interface{}{
				"access_token":  "new_access_token",
				"refresh_token": "new_refresh_token",
				"expires_in":    3600,
				"token_type":    "Bearer",
			}
			json.NewEncoder(w).Encode(tokenResp)
		} else if r.URL.Path == "/v1/exchange" {
			// Mock exchange response
			exchangeResp := map[string]interface{}{
				"requestId":       "mock_request_id",
				"accessKeyId":     "mock_access_key_id",
				"accessKeySecret": "mock_access_key_secret",
				"expiration":      "2023-12-31T23:59:59Z",
				"securityToken":   "mock_security_token",
			}
			json.NewEncoder(w).Encode(exchangeResp)
		}
	}))
	defer server.Close()

	// Override OAuth URLs for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap
	defer func() {
		oauthBaseUrlMap = originalOauthBaseUrlMap
	}()
	oauthBaseUrlMap["CN"] = server.URL

	w := new(bytes.Buffer)
	cp := &Profile{
		Name:                   "test",
		Mode:                   OAuth,
		OAuthSiteType:          "CN",
		OAuthAccessToken:       "old_access_token",
		OAuthAccessTokenExpire: 0, // Expired
		OAuthRefreshToken:      "mock_refresh_token",
	}

	err := exchangeFromOAuth(w, cp)
	assert.NoError(t, err)
	assert.Equal(t, "new_access_token", cp.OAuthAccessToken)
	assert.Equal(t, "new_refresh_token", cp.OAuthRefreshToken)
	assert.Equal(t, "mock_access_key_id", cp.AccessKeyId)
}

func TestExchangeFromOAuth_NoRefreshToken(t *testing.T) {
	w := new(bytes.Buffer)
	cp := &Profile{
		Name:                   "test",
		Mode:                   OAuth,
		OAuthSiteType:          "CN",
		OAuthAccessToken:       "old_access_token",
		OAuthAccessTokenExpire: 0,  // Expired
		OAuthRefreshToken:      "", // No refresh token
	}

	err := exchangeFromOAuth(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "both access token and refresh token are empty")
}

func TestExchangeFromOAuth_ExchangeFailed(t *testing.T) {
	// Create a mock HTTP server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/exchange" {
			w.WriteHeader(http.StatusBadRequest)
			errorResp := map[string]interface{}{
				"error":             "invalid_token",
				"error_description": "The access token is invalid",
				"requestId":         "mock_request_id",
			}
			json.NewEncoder(w).Encode(errorResp)
		}
	}))
	defer server.Close()

	originalOauthBaseUrlMap := oauthBaseUrlMap
	defer func() {
		oauthBaseUrlMap = originalOauthBaseUrlMap
	}()
	oauthBaseUrlMap["CN"] = server.URL

	w := new(bytes.Buffer)
	cp := &Profile{
		Name:                   "test",
		Mode:                   OAuth,
		OAuthSiteType:          "CN",
		OAuthAccessToken:       "invalid_token",
		OAuthAccessTokenExpire: 9999999999, // Not expired
		OAuthRefreshToken:      "mock_refresh_token",
	}

	err := exchangeFromOAuth(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exchange failed")
}

func TestTryRefreshOauthToken_Success(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/token" {
			tokenResp := map[string]interface{}{
				"access_token":  "new_access_token",
				"refresh_token": "new_refresh_token",
				"expires_in":    3600,
				"token_type":    "Bearer",
			}
			json.NewEncoder(w).Encode(tokenResp)
		}
	}))
	defer server.Close()

	originalOauthBaseUrlMap := oauthBaseUrlMap
	defer func() {
		oauthBaseUrlMap = originalOauthBaseUrlMap
	}()
	oauthBaseUrlMap["CN"] = server.URL

	w := new(bytes.Buffer)
	cp := &Profile{
		Name:              "test",
		Mode:              OAuth,
		OAuthSiteType:     "CN",
		OAuthRefreshToken: "mock_refresh_token",
	}

	err := tryRefreshOauthToken(w, cp)
	assert.NoError(t, err)
	assert.Equal(t, "new_access_token", cp.OAuthAccessToken)
	assert.Equal(t, "new_refresh_token", cp.OAuthRefreshToken)
}

func TestTryRefreshOauthToken_NoRefreshToken(t *testing.T) {
	w := new(bytes.Buffer)
	cp := &Profile{
		Name:              "test",
		Mode:              OAuth,
		OAuthSiteType:     "CN",
		OAuthRefreshToken: "",
	}

	err := tryRefreshOauthToken(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh token is empty")
}

func TestTryRefreshOauthToken_InvalidSiteType(t *testing.T) {
	w := new(bytes.Buffer)
	cp := &Profile{
		Name:              "test",
		Mode:              OAuth,
		OAuthSiteType:     "INVALID",
		OAuthRefreshToken: "mock_refresh_token",
	}

	err := tryRefreshOauthToken(w, cp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OAuth site type")
}

func TestBuildReLoginOauthCommand(t *testing.T) {
	cp := &Profile{
		Name:          "test-profile",
		OAuthSiteType: "CN",
	}

	result := buildReLoginOauthCommand(cp)
	expected := "aliyun configure --mode OAuth --oauth-site-type CN --profile test-profile"
	assert.Equal(t, expected, result)

	// Test with default profile
	cp.Name = "default"
	result = buildReLoginOauthCommand(cp)
	expected = "aliyun configure --mode OAuth --oauth-site-type CN"
	assert.Equal(t, expected, result)

	// Test with empty name
	cp.Name = ""
	result = buildReLoginOauthCommand(cp)
	expected = "aliyun configure --mode OAuth --oauth-site-type CN"
	assert.Equal(t, expected, result)
}

func TestGenerateCodeVerifier(t *testing.T) {
	result := generateCodeVerifier()
	// Should return a string of 128 characters
	assert.IsType(t, "", result)
	assert.NotEmpty(t, result)
}

func TestGenerateCodeChallenge(t *testing.T) {
	codeVerifier := "test_code_verifier"
	result := generateCodeChallenge(codeVerifier)
	// The result should be a base64url encoded SHA256 hash
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "=") // base64url should not have padding
}

func TestOAuthDetectPortUse(t *testing.T) {
	// Test normal case - should find an available port
	port, err := detectPortUse(12345, 12349)
	assert.NoError(t, err)
	assert.True(t, port >= 12345 && port <= 12349)
}
