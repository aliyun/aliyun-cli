package cloudsso

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"io"
	"net/http"
	"strconv"
)

// AccessConfigurationsParameter is a struct that holds the parameters for accessing configurations.
type AccessConfigurationsParameter struct {
	UrlPrefix   string       `json:"urlPrefix"`
	AccessToken string       `json:"accessToken"`
	AccountId   string       `json:"accountId"`
	HttpClient  *http.Client `json:"-"`
}

// AccessConfigurationsRequest 表示获取访问配置的请求参数
type AccessConfigurationsRequest struct {
	AccountId  string
	NextToken  string
	MaxResults int
}

// AccessConfigurationsResponse 表示访问配置的响应
type AccessConfigurationsResponse struct {
	AccessConfigurationsForAccount []AccessConfiguration `json:"AccessConfigurationsForAccount"`
	NextToken                      string                `json:"NextToken"`
	IsTruncated                    bool                  `json:"IsTruncated"`
}

// AccessConfiguration 表示单个访问配置
type AccessConfiguration struct {
	AccessConfigurationId          string `json:"AccessConfigurationId"`
	AccessConfigurationName        string `json:"AccessConfigurationName"`
	AccessConfigurationDescription string `json:"AccessConfigurationDescription"`
}

// ListAccessConfigurationsForAccount 获取单次访问配置列表
func (p *AccessConfigurationsParameter) ListAccessConfigurationsForAccount(req AccessConfigurationsRequest) (*AccessConfigurationsResponse, error) {
	// 构建URL
	url := fmt.Sprintf("%s/access-assignments/access-configurations", p.UrlPrefix)

	// 添加查询参数
	query := url + "?AccountId=" + req.AccountId
	if req.NextToken != "" {
		query += "&NextToken=" + req.NextToken
	}
	if req.MaxResults > 0 {
		query += "&MaxResults=" + strconv.Itoa(req.MaxResults)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.AccessToken)
	httpReq.Header.Set("User-Agent", "aliyun/CLI-"+cli.Version)

	// 发送请求
	client := p.HttpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 检查错误
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		var errResp struct {
			ErrorCode    string `json:"ErrorCode"`
			ErrorMessage string `json:"ErrorMessage"`
			RequestId    string `json:"RequestId"`
		}

		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("%s: %s %s", errResp.ErrorCode, errResp.ErrorMessage, errResp.RequestId)
	}

	// 解析响应
	var result AccessConfigurationsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListAllAccessConfigurations 获取所有访问配置列表
func (p *AccessConfigurationsParameter) ListAllAccessConfigurations(req AccessConfigurationsRequest) ([]AccessConfiguration, error) {
	var configurations []AccessConfiguration

	// 获取第一页数据
	response, err := p.ListAccessConfigurationsForAccount(req)
	if err != nil {
		return nil, err
	}

	configurations = append(configurations, response.AccessConfigurationsForAccount...)

	// 如果有更多页，继续请求
	for response.IsTruncated {
		req.NextToken = response.NextToken

		response, err = p.ListAccessConfigurationsForAccount(req)
		if err != nil {
			return nil, err
		}

		configurations = append(configurations, response.AccessConfigurationsForAccount...)
	}

	return configurations, nil
}
