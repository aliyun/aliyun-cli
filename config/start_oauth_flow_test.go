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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock variables for testing startOauthFlow
var (
	mockBrowserOpened  bool
	mockBrowserURL     string
	mockHttpClientFunc func() *http.Client
	mockCurrentTime    int64
	mockMutex          sync.RWMutex // 添加读写锁来保护并发访问
)

// Mock util.OpenBrowser
func mockOpenBrowser(url string) error {
	mockMutex.Lock()
	defer mockMutex.Unlock()
	mockBrowserOpened = true
	mockBrowserURL = url
	return nil
}

// 安全地读取 mockBrowserOpened
func getMockBrowserOpened() bool {
	mockMutex.RLock()
	defer mockMutex.RUnlock()
	return mockBrowserOpened
}

// 安全地读取 mockBrowserURL
func getMockBrowserURL() string {
	mockMutex.RLock()
	defer mockMutex.RUnlock()
	return mockBrowserURL
}

// 安全地重置 mock 变量
func resetMockVariables() {
	mockMutex.Lock()
	defer mockMutex.Unlock()
	mockBrowserOpened = false
	mockBrowserURL = ""
	mockCurrentTime = 0
}

// Mock util.NewHttpClient
func mockNewHttpClient() *http.Client {
	if mockHttpClientFunc != nil {
		return mockHttpClientFunc()
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// Mock util.GetCurrentUnixTime
func mockGetCurrentUnixTime() int64 {
	mockMutex.RLock()
	defer mockMutex.RUnlock()
	if mockCurrentTime > 0 {
		return mockCurrentTime
	}
	return 1640995200 // Default: 2022-01-01 00:00:00 UTC
}

// 安全地设置 mockCurrentTime
func setMockCurrentTime(time int64) {
	mockMutex.Lock()
	defer mockMutex.Unlock()
	mockCurrentTime = time
}

func TestStartOauthFlow_Success_CN_SiteType(t *testing.T) {
	// Setup mock functions
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient
	originalUtilGetCurrentUnixTime := utilGetCurrentUnixTime

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		utilGetCurrentUnixTime = originalUtilGetCurrentUnixTime
		resetMockVariables()
	}()

	// Set up mocks
	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient
	utilGetCurrentUnixTime = mockGetCurrentUnixTime
	setMockCurrentTime(1640995200)

	// Create a test server to mock OAuth token endpoint
	tokenServer := createMockOAuthTokenServer(t, http.StatusOK, map[string]interface{}{
		"access_token":  "test_access_token",
		"refresh_token": "test_refresh_token",
		"expires_in":    3600,
		"token_type":    "Bearer",
	})
	defer tokenServer.Close()

	// Override OAuth base URL for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap["CN"]
	oauthBaseUrlMap["CN"] = tokenServer.URL
	defer func() {
		oauthBaseUrlMap["CN"] = originalOauthBaseUrlMap
	}()

	// Create test profile
	cp := &Profile{
		OAuthSiteType: "CN",
	}

	var output bytes.Buffer

	// Start OAuth flow in a goroutine
	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Simulate OAuth callback with valid code
	go func() {
		time.Sleep(200 * time.Millisecond)
		// We need to extract the state from the URL that was "opened" in browser
		parsedURL, err := url.Parse(getMockBrowserURL())
		if err != nil {
			t.Errorf("Failed to parse browser URL: %v", err)
			return
		}
		state := parsedURL.Query().Get("state")

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:12345/cli/callback?code=test_auth_code&state=%s", state))
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Wait for result
	select {
	case err := <-resultChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}

	// Verify browser was called
	assert.True(t, getMockBrowserOpened(), "Browser should have been called")
	assert.Contains(t, getMockBrowserURL(), "https://signin.aliyun.com/oauth2/v1/auth")
	assert.Contains(t, getMockBrowserURL(), "client_id=4038181954557748008")
	assert.Contains(t, getMockBrowserURL(), "code_challenge_method=S256")

	// Verify profile was updated with tokens
	assert.Equal(t, "test_access_token", cp.OAuthAccessToken)
	assert.Equal(t, "test_refresh_token", cp.OAuthRefreshToken)
	assert.Equal(t, int64(1640998800), cp.OAuthAccessTokenExpire) // current time + 3600

	// Verify output contains expected messages
	outputStr := output.String()
	assert.Contains(t, outputStr, "Please open the following URL in your browser")
	assert.Contains(t, outputStr, "https://signin.aliyun.com/oauth2/v1/auth")
}

func TestStartOauthFlow_Success_INTL_SiteType(t *testing.T) {
	// Setup mock functions
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient
	originalUtilGetCurrentUnixTime := utilGetCurrentUnixTime

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		utilGetCurrentUnixTime = originalUtilGetCurrentUnixTime
		resetMockVariables()
	}()

	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient
	utilGetCurrentUnixTime = mockGetCurrentUnixTime
	setMockCurrentTime(1640995200)

	// Create a test server to mock OAuth token endpoint
	tokenServer := createMockOAuthTokenServer(t, http.StatusOK, map[string]interface{}{
		"access_token":  "test_access_token_intl",
		"refresh_token": "test_refresh_token_intl",
		"expires_in":    7200,
		"token_type":    "Bearer",
	})
	defer tokenServer.Close()

	// Override OAuth base URL for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap["INTL"]
	oauthBaseUrlMap["INTL"] = tokenServer.URL
	defer func() {
		oauthBaseUrlMap["INTL"] = originalOauthBaseUrlMap
	}()

	// Create test profile
	cp := &Profile{
		OAuthSiteType: "INTL",
	}

	var output bytes.Buffer

	// Start OAuth flow in a goroutine
	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate OAuth callback
	go func() {
		time.Sleep(200 * time.Millisecond)
		parsedURL, err := url.Parse(getMockBrowserURL())
		if err != nil {
			t.Errorf("Failed to parse browser URL: %v", err)
			return
		}
		state := parsedURL.Query().Get("state")

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:12345/cli/callback?code=test_auth_code_intl&state=%s", state))
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case err := <-resultChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}

	assert.True(t, getMockBrowserOpened())
	assert.Contains(t, getMockBrowserURL(), "https://signin.alibabacloud.com/oauth2/v1/auth")
	assert.Contains(t, getMockBrowserURL(), "client_id=4103531455503354461")
	assert.Equal(t, "test_access_token_intl", cp.OAuthAccessToken)
	assert.Equal(t, "test_refresh_token_intl", cp.OAuthRefreshToken)
	assert.Equal(t, int64(1641002400), cp.OAuthAccessTokenExpire) // current time + 7200
}

func TestStartOauthFlow_InvalidSiteType_Error(t *testing.T) {
	cp := &Profile{
		OAuthSiteType: "INVALID",
	}

	var output bytes.Buffer
	err := startOauthFlow(&output, cp)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid OAuth site type: INVALID, only support CN or INTL")
}

func TestStartOauthFlow_CallbackError_InvalidState(t *testing.T) {
	// Setup mocks
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		resetMockVariables()
	}()

	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient

	cp := &Profile{
		OAuthSiteType: "CN",
	}

	var output bytes.Buffer

	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate callback with invalid state
	go func() {
		time.Sleep(200 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:12345/cli/callback?code=test_code&state=invalid_state")
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case err := <-resultChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestStartOauthFlow_CallbackError_NoCode(t *testing.T) {
	// Setup mocks
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		resetMockVariables()
	}()

	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient

	cp := &Profile{
		OAuthSiteType: "CN",
	}

	var output bytes.Buffer

	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate callback without code
	go func() {
		time.Sleep(200 * time.Millisecond)
		parsedURL, err := url.Parse(getMockBrowserURL())
		if err != nil {
			t.Errorf("Failed to parse browser URL: %v", err)
			return
		}
		state := parsedURL.Query().Get("state")

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:12345/cli/callback?state=%s", state))
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case err := <-resultChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "code not found")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestStartOauthFlow_TokenExchangeError(t *testing.T) {
	// Setup mocks
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient
	originalUtilGetCurrentUnixTime := utilGetCurrentUnixTime

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		utilGetCurrentUnixTime = originalUtilGetCurrentUnixTime
		resetMockVariables()
	}()

	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient
	utilGetCurrentUnixTime = mockGetCurrentUnixTime
	setMockCurrentTime(1640995200)

	// Create a test server that returns error
	tokenServer := createMockOAuthTokenServer(t, http.StatusBadRequest, map[string]interface{}{
		"error":             "invalid_grant",
		"error_description": "The authorization code is invalid",
		"requestId":         "test-request-id",
	})
	defer tokenServer.Close()

	// Override OAuth base URL for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap["CN"]
	oauthBaseUrlMap["CN"] = tokenServer.URL
	defer func() {
		oauthBaseUrlMap["CN"] = originalOauthBaseUrlMap
	}()

	cp := &Profile{
		OAuthSiteType: "CN",
	}

	var output bytes.Buffer

	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate OAuth callback
	go func() {
		time.Sleep(200 * time.Millisecond)
		parsedURL, err := url.Parse(getMockBrowserURL())
		if err != nil {
			t.Errorf("Failed to parse browser URL: %v", err)
			return
		}
		state := parsedURL.Query().Get("state")

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:12345/cli/callback?code=invalid_code&state=%s", state))
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case err := <-resultChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get token")
		assert.Contains(t, err.Error(), "400")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}
}

func TestStartOauthFlow_TokenExchangeSuccess_NoAccessToken(t *testing.T) {
	// Setup mocks
	originalUtilOpenBrowser := utilOpenBrowser
	originalUtilNewHttpClient := utilNewHttpClient
	originalUtilGetCurrentUnixTime := utilGetCurrentUnixTime

	defer func() {
		utilOpenBrowser = originalUtilOpenBrowser
		utilNewHttpClient = originalUtilNewHttpClient
		utilGetCurrentUnixTime = originalUtilGetCurrentUnixTime
		resetMockVariables()
	}()

	utilOpenBrowser = mockOpenBrowser
	utilNewHttpClient = mockNewHttpClient
	utilGetCurrentUnixTime = mockGetCurrentUnixTime
	setMockCurrentTime(1640995200)

	// Create a test server that returns success but no access token
	tokenServer := createMockOAuthTokenServer(t, http.StatusOK, map[string]interface{}{
		"refresh_token": "test_refresh_token",
		"expires_in":    3600,
		"token_type":    "Bearer",
		// access_token is missing
	})
	defer tokenServer.Close()

	// Override OAuth base URL for testing
	originalOauthBaseUrlMap := oauthBaseUrlMap["CN"]
	oauthBaseUrlMap["CN"] = tokenServer.URL
	defer func() {
		oauthBaseUrlMap["CN"] = originalOauthBaseUrlMap
	}()

	cp := &Profile{
		OAuthSiteType: "CN",
	}

	var output bytes.Buffer

	resultChan := make(chan error, 1)
	go func() {
		err := startOauthFlow(&output, cp)
		resultChan <- err
	}()

	time.Sleep(100 * time.Millisecond)

	// Simulate OAuth callback
	go func() {
		time.Sleep(200 * time.Millisecond)
		parsedURL, err := url.Parse(getMockBrowserURL())
		if err != nil {
			t.Errorf("Failed to parse browser URL: %v", err)
			return
		}
		state := parsedURL.Query().Get("state")

		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:12345/cli/callback?code=test_code&state=%s", state))
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case err := <-resultChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access token not found in response")
	case <-time.After(5 * time.Second):
		t.Fatal("Test timeout")
	}
}

// Helper function to create mock token server
func createMockOAuthTokenServer(t *testing.T, statusCode int, response map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/token", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err)

		// Verify required parameters
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.NotEmpty(t, r.FormValue("code"))
		assert.NotEmpty(t, r.FormValue("client_id"))
		assert.NotEmpty(t, r.FormValue("redirect_uri"))
		assert.NotEmpty(t, r.FormValue("code_verifier"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		responseBytes, err := json.Marshal(response)
		require.NoError(t, err)

		_, err = w.Write(responseBytes)
		require.NoError(t, err)
	}))
}
