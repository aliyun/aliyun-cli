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

package cloudsso

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func TryRefreshStsToken(signInUrl *string, accessToken *string, accessConfig *string, accountId *string, client *http.Client) (*CloudCredentialResponse, error) {
	// parse signInUrl get host and protocol
	if signInUrl == nil || *signInUrl == "" {
		return nil, errors.New("signInUrl is empty")
	}
	parsedUrl, err := ParseUrl(signInUrl)
	if err != nil {
		return nil, err
	}
	if len(parsedUrl) != 2 {
		return nil, errors.New("invalid signInUrl")
	}
	host := parsedUrl[1]
	protocol := parsedUrl[0]

	// 使用传入的HTTP客户端，如果为nil则使用默认客户端
	httpClient := client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	credential, err := CreateCloudCredential(protocol+"://"+host, *accessToken, CloudCredentialOptions{
		AccountId:             *accountId,
		AccessConfigurationId: *accessConfig,
	}, httpClient)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, errors.New("credential is nil")
	}
	return credential, nil
}

func ParseUrl(urlStr *string) ([]string, error) {
	if urlStr == nil || *urlStr == "" {
		return nil, errors.New("url is empty")
	}

	parsedUrl, err := url.Parse(*urlStr)
	if err != nil {
		return nil, err
	}

	host := parsedUrl.Host
	scheme := parsedUrl.Scheme

	if host == "" || scheme == "" {
		return nil, errors.New("invalid url: missing host or scheme")
	}

	return []string{scheme, host}, nil
}

type CloudCredentialOptions struct {
	AccountId             string `json:"AccountId"`
	AccessConfigurationId string `json:"AccessConfigurationId"`
}

type CloudCredentialResponse struct {
	AccessKeyId     string `json:"AccessKeyId"`
	AccessKeySecret string `json:"AccessKeySecret"`
	SecurityToken   string `json:"SecurityToken"`
	Expiration      string `json:"Expiration"`
	ExpirationInt64 int64  `json:"ExpirationInt64"`
}

type CloudCredentialResponseRaw struct {
	CloudCredential *CloudCredentialResponse `json:"CloudCredential"`
	RequestId       string                   `json:"RequestId"`
}

func CreateCloudCredential(prefix string, accessToken string, options CloudCredentialOptions, client *http.Client) (*CloudCredentialResponse, error) {
	urlFetch := fmt.Sprintf("%s/cloud-credentials", prefix)

	// Prepare request body
	data, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", urlFetch, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("user-agent", "aliyun/CLI-"+cli.Version)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response body: %w", err)
		}
		bodyString := string(bodyBytes)
		var errResp map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
			// 如果解析 JSON 失败，返回原始响应体作为错误信息
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyString)
		}
		return nil, fmt.Errorf("HTTP %d: %s: %s %s", resp.StatusCode, bodyString, errResp["ErrorCode"], errResp["ErrorMessage"])
	}

	// Parse successful response
	var result CloudCredentialResponseRaw
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.CloudCredential.Expiration != "" {
		// Parse expiration time
		expiration, err := time.Parse(time.RFC3339, result.CloudCredential.Expiration)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration time: %w", err)
		}
		result.CloudCredential.ExpirationInt64 = expiration.Unix()
	}

	return result.CloudCredential, nil
}
