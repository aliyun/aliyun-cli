package cloudsso

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAllUsers(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   interface{}
		mockStatusCode int
		expectError    bool
	}{
		{
			name: "Success with single page",
			mockResponse: ListUsersResponse{
				Accounts: []AccountDetailResponse{
					{AccountId: "123", DisplayName: "Account1"},
					{AccountId: "456", DisplayName: "Account2"},
				},
				IsTruncated: false,
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Success with multiple pages",
			mockResponse: []ListUsersResponse{
				{
					Accounts: []AccountDetailResponse{
						{AccountId: "123", DisplayName: "Account1"},
					},
					IsTruncated: true,
					NextToken:   "token1",
				},
				{
					Accounts: []AccountDetailResponse{
						{AccountId: "456", DisplayName: "Account2"},
					},
					IsTruncated: false,
				},
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name: "Error response from server",
			mockResponse: ErrorResponse{
				ErrorCode:    "InvalidToken",
				ErrorMessage: "The provided token is invalid.",
				RequestId:    "req-123",
			},
			mockStatusCode: http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode == http.StatusOK {
					if responses, ok := tt.mockResponse.([]ListUsersResponse); ok {
						// Handle paginated responses
						if r.URL.Query().Get("NextToken") == "token1" {
							json.NewEncoder(w).Encode(responses[1])
						} else {
							json.NewEncoder(w).Encode(responses[0])
						}
					} else {
						json.NewEncoder(w).Encode(tt.mockResponse)
					}
				} else {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Prepare ListUserParameter
			client := &http.Client{}
			param := &ListUserParameter{
				BaseUrl:     server.URL,
				AccessToken: "mock-token",
				HttpClient:  client,
			}

			// Call ListAllUsers
			accounts, err := param.ListAllUsers()

			// Validate results
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect error but got: %v", err)
				}
				if tt.name == "Success with multiple pages" {
					expectedAccounts := 0
					for _, resp := range tt.mockResponse.([]ListUsersResponse) {
						expectedAccounts += len(resp.Accounts)
					}
					if len(accounts) != expectedAccounts {
						t.Errorf("expected %d accounts, got %d", expectedAccounts, len(accounts))
					}
				} else {
					if len(accounts) != len(tt.mockResponse.(ListUsersResponse).Accounts) {
						t.Errorf("expected %d accounts, got %d", len(tt.mockResponse.(ListUsersResponse).Accounts), len(accounts))
					}
				}
			}
		})
	}
}

func TestSsoLogin_GetUserList_Manual(t *testing.T) {
	t.Skip("默认跳过，需要手工移除此行执行真实测试")

	sso := SsoLogin{
		SignInUrl: "https://signin-cn-shanghai.alibabacloudsso.com/start/login",
	}
	token, err := sso.GetAccessToken()
	if err != nil {
		t.Errorf("GetAccessToken error: %v", err)
	} else {
		t.Logf("GetAccessToken: %s", token)
	}
	// start get user list
	param := &ListUserParameter{
		BaseUrl:     "https://signin-cn-shanghai.alibabacloudsso.com",
		AccessToken: token,
		HttpClient:  &http.Client{},
	}
	users, err := param.ListAllUsers()
	if err != nil {
		t.Errorf("ListAllUsers error: %v", err)
	} else {
		t.Logf("ListAllUsers: %v", users)
	}
}
