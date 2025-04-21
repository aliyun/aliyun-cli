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
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type ListUserParameter struct {
	BaseUrl     string       `json:"base_url"`
	AccessToken string       `json:"access_token"`
	HttpClient  *http.Client `json:"-"`
}

type AccountDetailResponse struct {
	AccountId   string `json:"AccountId"`
	DisplayName string `json:"DisplayName"`
}

// ListUsersResponse 保存列出用户的响应
type ListUsersResponse struct {
	Accounts    []AccountDetailResponse `json:"Accounts"`
	IsTruncated bool                    `json:"IsTruncated"`
	NextToken   string                  `json:"NextToken"`
}

// ErrorResponse 用于处理错误响应
type ErrorResponse struct {
	ErrorCode    string `json:"ErrorCode"`
	ErrorMessage string `json:"ErrorMessage"`
	RequestId    string `json:"RequestId"`
}

// ListUsers 获取账户列表，支持分页
func (p *ListUserParameter) ListUsers(nextToken string, maxResults int) (*ListUsersResponse, error) {
	apiUrl, err := url.Parse(fmt.Sprintf("%s/access-assignments/accounts", p.BaseUrl))
	if err != nil {
		return nil, err
	}

	query := apiUrl.Query()
	if nextToken != "" {
		query.Add("NextToken", nextToken)
	}
	if maxResults > 0 {
		query.Add("MaxResults", fmt.Sprintf("%d", maxResults))
	}
	apiUrl.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", apiUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", p.AccessToken))
	req.Header.Add("user-agent", "aliyun/CLI-"+cli.Version)

	p.HttpClient.Timeout = 10000 * time.Millisecond

	resp, err := p.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %s %s", errResp.ErrorCode, errResp.ErrorMessage, errResp.RequestId)
	}

	var result ListUsersResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAllUsers 获取所有账户列表
func (p *ListUserParameter) ListAllUsers() ([]AccountDetailResponse, error) {
	var accounts []AccountDetailResponse
	response, err := p.ListUsers("", 100)
	if err != nil {
		return nil, err
	}

	accounts = append(accounts, response.Accounts...)

	for response.IsTruncated {
		response, err = p.ListUsers(response.NextToken, 100)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, response.Accounts...)
	}

	return accounts, nil
}
