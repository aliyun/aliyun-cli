package cloudsso

import (
	"github.com/aliyun/aliyun-cli/v3/util"
	"testing"
)

func TestSsoLogin_JudgeLogin(t *testing.T) {
	// 保存原始GetCurrentUnixTime函数，测试结束后恢复
	originalGetCurrentUnixTime := util.GetCurrentUnixTime

	// 过期时间：当前时间前后5分钟
	expiredTime := originalGetCurrentUnixTime() - 300 // 5分钟前(已过期)
	validTime := originalGetCurrentUnixTime() + 300   // 5分钟后(有效)

	tests := []struct {
		name       string
		login      SsoLogin
		wantLogin  bool
		wantErr    bool
		errMessage string
	}{
		{
			name: "SignInUrl为空应返回错误",
			login: SsoLogin{
				SignInUrl:    "",
				AccessToken:  "token",
				AccountId:    "account",
				AccessConfig: "config",
				ExpireTime:   validTime,
			},
			wantLogin:  false,
			wantErr:    true,
			errMessage: "signInUrl is required",
		},
		{
			name: "AccessToken为空时需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "",
				AccountId:    "account",
				AccessConfig: "config",
				ExpireTime:   validTime,
			},
			wantLogin: true,
			wantErr:   false,
		},
		{
			name: "AccountId为空时需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "token",
				AccountId:    "",
				AccessConfig: "config",
				ExpireTime:   validTime,
			},
			wantLogin: true,
			wantErr:   false,
		},
		{
			name: "AccessConfig为空时需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "token",
				AccountId:    "account",
				AccessConfig: "",
				ExpireTime:   validTime,
			},
			wantLogin: true,
			wantErr:   false,
		},
		{
			name: "ExpireTime为0时需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "token",
				AccountId:    "account",
				AccessConfig: "config",
				ExpireTime:   0,
			},
			wantLogin: true,
			wantErr:   false,
		},
		{
			name: "ExpireTime已过期时需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "token",
				AccountId:    "account",
				AccessConfig: "config",
				ExpireTime:   expiredTime,
			},
			wantLogin: true,
			wantErr:   false,
		},
		{
			name: "所有条件都满足时不需要登录",
			login: SsoLogin{
				SignInUrl:    "https://example.com",
				AccessToken:  "token",
				AccountId:    "account",
				AccessConfig: "config",
				ExpireTime:   validTime,
			},
			wantLogin: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login, err := tt.login.JudgeLogin()

			// 检查是否应该返回错误
			if (err != nil) != tt.wantErr {
				t.Errorf("JudgeLogin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 如果期望有错误，检查错误消息
			if tt.wantErr && err.Error() != tt.errMessage {
				t.Errorf("JudgeLogin() error message = %v, want %v", err.Error(), tt.errMessage)
				return
			}

			// 检查返回值是否符合预期
			if login != tt.wantLogin {
				t.Errorf("JudgeLogin() = %v, want %v", login, tt.wantLogin)
			}
		})
	}
}

func TestSsoLogin_GetAccessToken_Manual(t *testing.T) {
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
}
