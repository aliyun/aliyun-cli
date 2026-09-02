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
	if accessToken == nil {
		return nil, errors.New("accessToken is nil")
	}
	if accessConfig == nil {
		return nil, errors.New("accessConfig is nil")
	}
	if accountId == nil {
		return nil, errors.New("accountId is nil")
	}
	host := parsedUrl[1]
	protocol := parsedUrl[0]

	// 使用传入的 HTTP 客户端，未配置超时时补充有限的默认值。
	httpClient := cloudSSOHTTPClient(client)

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
	client = cloudSSOHTTPClient(client)
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
	if resp == nil {
		return nil, errors.New("cloud credential response is nil")
	}
	if resp.Body == nil {
		return nil, errors.New("cloud credential response body is nil")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle HTTP errors
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		bodyString := string(body)
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err != nil {
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

	credential := result.CloudCredential
	if credential == nil {
		// Older endpoints and test doubles may return the credential object at
		// the top level. Continue accepting that shape for compatibility.
		var direct CloudCredentialResponse
		if err := json.Unmarshal(body, &direct); err != nil {
			return nil, fmt.Errorf("failed to parse cloud credential: %w", err)
		}
		if direct.AccessKeyId != "" || direct.AccessKeySecret != "" || direct.SecurityToken != "" || direct.Expiration != "" {
			credential = &direct
		}
	}
	if credential == nil {
		return nil, errors.New("cloud credential is missing from response")
	}

	if credential.Expiration != "" {
		// Parse expiration time
		expiration, err := time.Parse(time.RFC3339, credential.Expiration)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expiration time: %w", err)
		}
		credential.ExpirationInt64 = expiration.Unix()
	}

	return credential, nil
}
