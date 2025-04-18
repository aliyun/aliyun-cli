package cloudsso

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// MockHTTPClient 是一个模拟的HTTP客户端
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("no mock function provided")
}

// Do 实现http.Client接口
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// 创建一个模拟的HTTP响应
func mockHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestListAllAccessConfigurations(t *testing.T) {
	// 测试用例1：正常情况，一次请求获取所有数据（无分页）
	t.Run("Success with single page", func(t *testing.T) {
		// 模拟HTTP客户端
		mockTransport := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				// 验证请求URL
				if !strings.Contains(req.URL.String(), "AccountId=123456") {
					t.Errorf("Expected URL to contain 'AccountId=123456', got %s", req.URL.String())
				}

				// 返回模拟响应
				responseBody := `{
					"AccessConfigurationsForAccount": [
						{
							"AccessConfigurationId": "ac-1",
							"AccessConfigurationName": "Config 1",
							"AccessConfigurationDescription": "Description 1"
						},
						{
							"AccessConfigurationId": "ac-2",
							"AccessConfigurationName": "Config 2",
							"AccessConfigurationDescription": "Description 2"
						}
					],
					"IsTruncated": false
				}`
				return mockHTTPResponse(200, responseBody), nil
			},
		}

		// 创建一个使用模拟Transport的HTTP客户端
		mockClient := &http.Client{
			Transport: mockTransport,
		}

		// 创建参数对象
		params := &AccessConfigurationsParameter{
			UrlPrefix:  "https://example.com",
			AccountId:  "123456",
			HttpClient: mockClient,
		}

		// 调用被测试的函数
		configs, err := params.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: "123456",
		})

		// 验证结果
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(configs) != 2 {
			t.Errorf("Expected 2 configurations, got %d", len(configs))
		}
		if configs[0].AccessConfigurationId != "ac-1" {
			t.Errorf("Expected first config ID to be 'ac-1', got %s", configs[0].AccessConfigurationId)
		}
		if configs[1].AccessConfigurationName != "Config 2" {
			t.Errorf("Expected second config name to be 'Config 2', got %s", configs[1].AccessConfigurationName)
		}
	})

	// 测试用例2：正常情况，多次请求获取所有数据（有分页）
	t.Run("Success with pagination", func(t *testing.T) {
		callCount := 0
		// 模拟HTTP客户端
		mockTransport := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				callCount++

				// 第一页响应
				if callCount == 1 {
					responseBody := `{
						"AccessConfigurationsForAccount": [
							{
								"AccessConfigurationId": "ac-1",
								"AccessConfigurationName": "Config 1",
								"AccessConfigurationDescription": "Description 1"
							}
						],
						"NextToken": "page2",
						"IsTruncated": true
					}`
					return mockHTTPResponse(200, responseBody), nil
				}

				// 第二页响应
				responseBody := `{
					"AccessConfigurationsForAccount": [
						{
							"AccessConfigurationId": "ac-2",
							"AccessConfigurationName": "Config 2",
							"AccessConfigurationDescription": "Description 2"
						}
					],
					"IsTruncated": false
				}`
				return mockHTTPResponse(200, responseBody), nil
			},
		}

		mockClient := &http.Client{
			Transport: mockTransport,
		}

		// 创建参数对象
		params := &AccessConfigurationsParameter{
			UrlPrefix:  "https://example.com",
			AccountId:  "123456",
			HttpClient: mockClient,
		}

		// 调用被测试的函数
		configs, err := params.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: "123456",
		})

		// 验证结果
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(configs) != 2 {
			t.Errorf("Expected 2 configurations, got %d", len(configs))
		}
		if callCount != 2 {
			t.Errorf("Expected 2 HTTP calls, got %d", callCount)
		}
	})

	// 测试用例3：HTTP请求失败
	t.Run("HTTP request error", func(t *testing.T) {
		// 模拟HTTP客户端
		mockTransport := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			},
		}

		mockClient := &http.Client{
			Transport: mockTransport,
		}

		// 创建参数对象
		params := &AccessConfigurationsParameter{
			UrlPrefix:  "https://example.com",
			AccountId:  "123456",
			HttpClient: mockClient,
		}

		// 调用被测试的函数
		_, err := params.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: "123456",
		})

		// 验证结果
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "network error") {
			t.Errorf("Expected error containing 'network error', got %v", err)
		}
	})

	// 测试用例4：响应状态码表示错误（4xx）
	t.Run("Error response status code", func(t *testing.T) {
		// 模拟HTTP客户端
		mockTransport := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				responseBody := `{
					"ErrorCode": "AccessDenied",
					"ErrorMessage": "You do not have permission",
					"RequestId": "ABC123"
				}`
				return mockHTTPResponse(403, responseBody), nil
			},
		}

		mockClient := &http.Client{
			Transport: mockTransport,
		}

		// 创建参数对象
		params := &AccessConfigurationsParameter{
			UrlPrefix:  "https://example.com",
			AccountId:  "123456",
			HttpClient: mockClient,
		}

		// 调用被测试的函数
		_, err := params.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: "123456",
		})

		// 验证结果
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !strings.Contains(err.Error(), "AccessDenied") {
			t.Errorf("Expected error containing 'AccessDenied', got %v", err)
		}
	})

	// 测试用例5：解析响应体失败
	t.Run("Invalid response body", func(t *testing.T) {
		// 模拟HTTP客户端
		mockTransport := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				// 返回无效的JSON
				return mockHTTPResponse(200, "invalid json"), nil
			},
		}

		mockClient := &http.Client{
			Transport: mockTransport,
		}

		// 创建参数对象
		params := &AccessConfigurationsParameter{
			UrlPrefix:  "https://example.com",
			AccountId:  "123456",
			HttpClient: mockClient,
		}

		// 调用被测试的函数
		_, err := params.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: "123456",
		})

		// 验证结果
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

// actual test code
func TestSsoLogin_GetAccessConfigurations_Manual(t *testing.T) {
	t.Skip("默认跳过，需要手工移除此行执行真实测试")

	sso := SsoLogin{
		SignInUrl: "https://signin-cn-shanghai.alibabacloudsso.com/start/login",
	}
	token, err := sso.GetAccessToken()
	if err != nil {
		t.Errorf("GetAccessToken error: %v", err)
	} else {
		t.Logf("GetAccessToken: %s", token.AccessToken)
	}
	// start get user list
	param := &ListUserParameter{
		BaseUrl:     "https://signin-cn-shanghai.alibabacloudsso.com",
		AccessToken: token.AccessToken,
		HttpClient:  &http.Client{},
	}
	users, err := param.ListAllUsers()
	if err != nil {
		t.Errorf("ListAllUsers error: %v", err)
	} else {
		t.Logf("ListAllUsers: %v", users)
	}
	// 确保数量大于 0
	if len(users) == 0 {
		t.Error("ListAllUsers returned no users")
	}
	// 获取第一个用户的 AccessConfigurations
	if len(users) > 0 {
		user := users[0]
		param2 := &AccessConfigurationsParameter{
			UrlPrefix:   "https://signin-cn-shanghai.alibabacloudsso.com",
			AccessToken: token.AccessToken,
			AccountId:   user.AccountId,
			HttpClient:  &http.Client{},
		}
		configs, err := param2.ListAllAccessConfigurations(AccessConfigurationsRequest{
			AccountId: user.AccountId,
		})
		if err != nil {
			t.Errorf("ListAllAccessConfigurations error: %v", err)
		} else {
			t.Logf("ListAllAccessConfigurations: %v", configs)
		}
	} else {
		t.Error("No users found")
	}
}
