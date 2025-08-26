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
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func TestNewConfigureCommand(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	excmd := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode <AuthenticateMode> --profile <profileName>",
	}
	configureGet := NewConfigureGetCommand()
	configureSet := NewConfigureSetCommand()
	configureList := NewConfigureListCommand()
	configureDelete := NewConfigureDeleteCommand()
	configureSwitch := NewConfigureSwitchCommand()
	excmd.AddSubCommand(configureGet)
	excmd.AddSubCommand(configureSet)
	excmd.AddSubCommand(configureList)
	excmd.AddSubCommand(configureDelete)
	excmd.AddSubCommand(configureSwitch)

	// testcase
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	// testcase
	err := configureGet.Run(ctx, []string{"get"})
	assert.Nil(t, err)
	assert.Equal(t, "\n", w.String())

	// testcase
	w.Reset()
	err = configureSet.Run(ctx, []string{"set"})
	assert.Nil(t, err)
	assert.Equal(t, "\x1b[1;31mfail to set configuration: region can't be empty\x1b[0m", w.String())

	// testcase
	w.Reset()
	err = configureList.Run(ctx, []string{"list"})
	assert.Nil(t, err)
	assert.Equal(t, "Profile   | Credential         | Valid   | Region           | Language\n--------- | ------------------ | ------- | ---------------- | --------\ndefault * | AK:***_id          | Invalid |                  | \naaa       | AK:******          | Invalid |                  | \n", w.String())

	// testcase
	w.Reset()
	stderr.Reset()
	err = configureDelete.Run(ctx, []string{"delete"})
	assert.Nil(t, err)
	assert.Equal(t, "\x1b[1;31mmissing --profile <profileName>\n\x1b[0m\x1b[1;33m\nusage:\n  aliyun configure delete --profile <profileName>\n\x1b[0m", stderr.String())

	w.Reset()
	stderr.Reset()
	err = configureSwitch.Run(ctx, []string{"switch"})
	assert.NotNil(t, err)
	assert.Equal(t, "the --profile <profileName> is required", err.Error())

	//testcase
	cmd := NewConfigureCommand()
	excmd.Run = cmd.Run
	assert.ObjectsAreEqualValues(excmd, cmd)

	//testcase
	w.Reset()
	stderr.Reset()
	err = cmd.Run(ctx, []string{"configure"})
	assert.Empty(t, w.String())
	assert.NotNil(t, err)
}

func TestDoConfigure(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:            "default",
						Mode:            AK,
						AccessKeyId:     "default_aliyun_access_key_id",
						AccessKeySecret: "default_aliyun_access_key_secret",
						OutputFormat:    "json",
					},
					{
						Name:            "aaa",
						Mode:            AK,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
					},
				},
			}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())
	err := doConfigure(ctx, "profile", "AK")
	assert.Nil(t, err)
	assert.Equal(t, "Configuring profile 'profile' in 'AK' authenticate mode...\n"+
		"Access Key Id []: Access Key Secret []: Default Region Id []: Default Output Format [json]: json (Only support json)\n"+
		"Default Language [zh|en] en: Saving profile[profile] ...Done.\n"+
		"-----------------------------------------------\n"+
		"!!! Configure Failed please configure again !!!\n"+
		"-----------------------------------------------\n"+
		"AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first\n"+
		"-----------------------------------------------\n"+
		"!!! Configure Failed please configure again !!!\n"+
		"-----------------------------------------------\n", w.String())
	w.Reset()

	err = doConfigure(ctx, "", "StsToken")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(w.String(), "Warning: You are changing the authentication type of profile 'default' from 'AK' to 'StsToken'\nConfiguring profile 'default' in 'StsToken' authenticate mode...\nAccess Key Id [*************************_id]: Access Key Secret [*****************************ret]: Sts Token []: Default Region Id []: Default Output Format [json]: json (Only support json)\nDefault Language [zh|en] : Saving profile[default] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n"))
	w.Reset()
}

func TestConfigureAK(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureAK(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Nil(t, err)
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: ", w.String())
}

func TestConfigureStsToken(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureStsToken(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", StsToken: "ststoken", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: Sts Token [ststoken]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRamRoleArn(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRamRoleArn(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RamRoleArn: "RamRoleArn", RoleSessionName: "RoleSessionName", ExternalId: "ExternalId", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: Sts Region []: Ram Role Arn [RamRoleArn]: Role Session Name [RoleSessionName]: External ID [ExternalId]: Expired Seconds [900]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureEcsRamRole(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureEcsRamRole(w, &Profile{Name: "default", Mode: AK, RamRoleName: "RamRoleName", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Ecs Ram Role [RamRoleName]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRamRoleArnWithEcs(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRamRoleArnWithEcs(w, &Profile{
		Name:            "default",
		Mode:            RamRoleArnWithEcs,
		RamRoleName:     "RamRoleName",
		RamRoleArn:      "rra",
		StsRegion:       "cn-hangzhou",
		RoleSessionName: "rsn",
		RegionId:        "cn-hangzhou",
		ExpiredSeconds:  3600,
		OutputFormat:    "json",
	})
	assert.Equal(t, "Ecs Ram Role [RamRoleName]: Sts Region [cn-hangzhou]: Ram Role Arn [rra]: Role Session Name [rsn]: Expired Seconds [3600]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRamRoleArnWithEcsWhenZeroExpiredSeconds(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRamRoleArnWithEcs(w, &Profile{
		Name:            "default",
		Mode:            RamRoleArnWithEcs,
		RamRoleName:     "RamRoleName",
		RamRoleArn:      "rra",
		StsRegion:       "cn-hangzhou",
		RoleSessionName: "rsn",
		RegionId:        "cn-hangzhou",
		OutputFormat:    "json",
	})
	assert.Equal(t, "Ecs Ram Role [RamRoleName]: Sts Region [cn-hangzhou]: Ram Role Arn [rra]: Role Session Name [rsn]: Expired Seconds [900]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureChainableRamRoleArn(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureChainableRamRoleArn(w, &Profile{
		Name:            "default",
		Mode:            ChainableRamRoleArn,
		SourceProfile:   "source",
		RamRoleArn:      "rra",
		StsRegion:       "cn-hangzhou",
		RoleSessionName: "rsn",
		ExternalId:      "eid",
		RegionId:        "cn-hangzhou",
		ExpiredSeconds:  3600,
		OutputFormat:    "json",
	})
	assert.Equal(t, "Source Profile [source]: Sts Region [cn-hangzhou]: Ram Role Arn [rra]: Role Session Name [rsn]: External ID [eid]: Expired Seconds [3600]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureChainableRamRoleArnWhenZeroExpiredSeconds(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureChainableRamRoleArn(w, &Profile{
		Name:            "default",
		Mode:            ChainableRamRoleArn,
		SourceProfile:   "source",
		RamRoleArn:      "rra",
		StsRegion:       "cn-hangzhou",
		RoleSessionName: "rsn",
		RegionId:        "cn-hangzhou",
		OutputFormat:    "json",
		ExternalId:      "eid",
	})
	assert.Equal(t, "Source Profile [source]: Sts Region [cn-hangzhou]: Ram Role Arn [rra]: Role Session Name [rsn]: External ID [eid]: Expired Seconds [900]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRsaKeyPair(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRsaKeyPair(w, &Profile{Name: "default", Mode: AK, RamRoleName: "RamRoleName", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Rsa Private Key File: ", w.String())
	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "read key file  failed open : The system cannot find the file specified.")
	} else {
		assert.EqualError(t, err, "read key file  failed open : no such file or directory")
	}
}

func TestConfigureExternal(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureExternal(w, &Profile{Name: "default", Mode: External, ProcessCommand: "process command", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Process Command [process command]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureCredentialsURI(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureCredentialsURI(w, &Profile{
		Name:           "default",
		Mode:           CredentialsURI,
		CredentialsURI: "http://credentials.uri/",
		RegionId:       "cn-hangzhou",
		OutputFormat:   "json",
	})
	assert.Equal(t, "Credentials URI [http://credentials.uri/]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureOIDC(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureOIDC(w, &Profile{
		Name:            "default",
		Mode:            OIDC,
		OIDCProviderARN: "oidcproviderarn",
		OIDCTokenFile:   "/path/to/oidc/token/file",
		RamRoleArn:      "rra",
		RoleSessionName: "rsn",
		RegionId:        "cn-hangzhou",
		OutputFormat:    "json",
	})
	assert.Equal(t, "OIDC Provider ARN [oidcproviderarn]: OIDC Token File [/path/to/oidc/token/file]: RAM Role ARN [rra]: Role Session Name [rsn]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureCloudSSO(t *testing.T) {
	// 保存原始 stdin 以便后续恢复
	originalStdin := stdin
	defer func() {
		stdin = originalStdin
	}()

	w := new(bytes.Buffer)

	// Case 1: 测试未输入 CloudSSOSignInUrl 的情况
	stdin = strings.NewReader("")
	profile := &Profile{
		Name:         "default",
		Mode:         CloudSSO,
		RegionId:     "cn-hangzhou",
		OutputFormat: "json",
	}
	err := configureCloudSSO(w, profile)
	assert.EqualError(t, err, "CloudSSOSignInUrl is required")
	assert.Equal(t, "CloudSSO Sign In Url []: ", w.String())
}

func TestConfigureCloudSSOWithMock(t *testing.T) {
	// 保存原始 stdin 和函数以便后续恢复
	originalStdin := stdin
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken

	defer func() {
		stdin = originalStdin
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
	}()

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\n1\n1\n")
	w := new(bytes.Buffer)

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	// 测试用例
	profile := &Profile{
		Name:         "default",
		Mode:         CloudSSO,
		RegionId:     "cn-hangzhou",
		OutputFormat: "json",
	}

	err := configureCloudSSO(w, profile)
	assert.Nil(t, err)

	// 验证结果
	assert.Equal(t, "https://sso.example.com", profile.CloudSSOSignInUrl)
	assert.Equal(t, "mock-access-token", profile.AccessToken)
	assert.Equal(t, "account123", profile.CloudSSOAccountId)
	assert.Equal(t, "config123", profile.CloudSSOAccessConfig)
	assert.Equal(t, "mock-ak-id", profile.AccessKeyId)
	assert.Equal(t, "mock-ak-secret", profile.AccessKeySecret)
	assert.Equal(t, "mock-security-token", profile.StsToken)
}

func TestDoConfigureWithCloudSSO(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\ncn-beijing\njson\nen\n")

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	// 测试 doConfigure 对 CloudSSO 模式的处理
	err := doConfigure(ctx, "cloud-sso-profile", "CloudSSO")
	assert.Nil(t, err)

	// 验证输出包含配置 CloudSSO 的信息
	output := w.String()
	assert.Contains(t, output, "Configuring profile 'cloud-sso-profile' in 'CloudSSO' authenticate mode...")
	assert.Contains(t, output, "Saving profile[cloud-sso-profile]")
	assert.Contains(t, output, "Done.")
}

func TestDoConfigureWithCloudSSOWhenSpecifyAccountNotExist(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\ncn-beijing\njson\nen\n")

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	// 加一个参数，指定 CloudSSOAccountId
	var accountIdFlag = NewCloudSSOAccountIdFlag()
	accountIdFlag.SetValue("account2")
	accountIdFlag.SetAssigned(true)
	ctx.Flags().Add(accountIdFlag)

	// 测试 doConfigure 对 CloudSSO 模式的处理
	err := doConfigure(ctx, "cloud-sso-profile", "CloudSSO")
	assert.Nil(t, err)

	// 验证输出包含配置 CloudSSO 的信息
	output := w.String()
	assert.Contains(t, output, "Account account2 not found, please choose again")
	assert.Contains(t, output, "Saving profile[cloud-sso-profile]")
	assert.Contains(t, output, "Done.")
}

// 返回多个账户的测试用例，需要主动输入一个数字选择
func TestDoConfigureWithCloudSSOReturnMultiAccount(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\ncn-beijing\njson\nen\n")

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
			{
				AccountId:   "account456",
				DisplayName: "Test Account 2",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	// 测试 doConfigure 对 CloudSSO 模式���处理
	err := doConfigure(ctx, "cloud-sso-profile", "CloudSSO")
	assert.Nil(t, err)

	// 验证输出包含配置 CloudSSO 的信息
	output := w.String()
	assert.Contains(t, output, "Please input the account number")
	assert.Contains(t, output, "Saving profile[cloud-sso-profile]")
	assert.Contains(t, output, "Done.")
}

// 通过 flag 指定accessConfigId，不存在的情况
func TestDoConfigureWithCloudSSOWhenSpecifyAccessConfigNotExist(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\ncn-beijing\njson\nen\n")

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
			{
				AccountId:   "account456",
				DisplayName: "Test Account 2",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	var f = NewCloudSSOAccessConfigFlag()
	f.SetAssigned(true)
	f.SetValue("xx")
	ctx.Flags().Add(f)

	// 测试 doConfigure 对 CloudSSO 模式的处理
	err := doConfigure(ctx, "cloud-sso-profile", "CloudSSO")
	assert.Nil(t, err)

	// 验证输出包含配置 CloudSSO 的信息
	output := w.String()
	assert.Contains(t, output, "Access Configuration xx not found, please choose again")
}

// 返回多个 access config的测试用例，需要主动输入一个数字选择
func TestDoConfigureWithCloudSSOWithMultiAccessConfig(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	// Mock stdin 输入
	stdin = strings.NewReader("https://sso.example.com\ncn-beijing\njson\nen\n")

	// Mock 所有 cloudsso 函数
	cloudssoGetAccessToken = func(ssoLogin *cloudsso.SsoLogin) (*cloudsso.AccessTokenResponse, error) {
		return &cloudsso.AccessTokenResponse{
			AccessToken: "mock-access-token",
			ExpiresIn:   3600,
		}, nil
	}

	cloudssoListAllUsers = func(userParam *cloudsso.ListUserParameter) ([]cloudsso.AccountDetailResponse, error) {
		return []cloudsso.AccountDetailResponse{
			{
				AccountId:   "account123",
				DisplayName: "Test Account",
			},
			{
				AccountId:   "account456",
				DisplayName: "Test Account 2",
			},
		}, nil
	}

	cloudssoListAllAccessConfigurations = func(accessParam *cloudsso.AccessConfigurationsParameter, req cloudsso.AccessConfigurationsRequest) ([]cloudsso.AccessConfiguration, error) {
		return []cloudsso.AccessConfiguration{
			{
				AccessConfigurationId:   "config123",
				AccessConfigurationName: "Test Config",
			},
			{
				AccessConfigurationId:   "config345",
				AccessConfigurationName: "Test Config2",
			},
		}, nil
	}

	cloudssoTryRefreshStsToken = func(signInUrl, accessToken, accessConfig, accountId *string, httpClient *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			Expiration:      "2099-01-01T00:00:00Z",
		}, nil
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)

	// 测试 doConfigure 对 CloudSSO 模式的处理
	err := doConfigure(ctx, "cloud-sso-profile", "CloudSSO")
	assert.Nil(t, err)

	// 验证输出包含配置 CloudSSO 的信息
	output := w.String()
	assert.Contains(t, output, "Please input the access configuration number")
}

// 原始的CloudSSOSignInUrl不为空，新输入为空
func TestDoConfigureWithCloudSSOWhenCloudSSOSignInUrlNotEmpty(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	originalGetAccessToken := cloudssoGetAccessToken
	originalListAllUsers := cloudssoListAllUsers
	originalListAllAccessConfigurations := cloudssoListAllAccessConfigurations
	originalTryRefreshStsToken := cloudssoTryRefreshStsToken
	originalStdin := stdin

	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
		cloudssoGetAccessToken = originalGetAccessToken
		cloudssoListAllUsers = originalListAllUsers
		cloudssoListAllAccessConfigurations = originalListAllAccessConfigurations
		cloudssoTryRefreshStsToken = originalTryRefreshStsToken
		stdin = originalStdin
	}()

	// Mock 配置加载和保存
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:         "default",
						Mode:         AK,
						AccessKeyId:  "default_ak",
						OutputFormat: "json",
					},
				},
			}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	var f = NewCloudSSOSignInUrlFlag()
	f.SetAssigned(true)
	f.SetValue("xxx")
	ctx.Flags().Add(f)

	profile := &Profile{
		Name:         "default",
		Mode:         CloudSSO,
		OutputFormat: "json",
	}

	stdin = strings.NewReader("yyy")
	err := doConfigure(ctx, profile.Name, string(AuthenticateMode("CloudSSO")))
	// stdin write empty
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "missing protocol")
}

func TestReadInput(t *testing.T) {
	defer func() {
		stdin = os.Stdin
	}()
	// read empty string, return default value
	stdin = strings.NewReader("")
	assert.Equal(t, "default", ReadInput("default"))
	// read input, return input
	stdin = strings.NewReader("input from stdion\n")
	assert.Equal(t, "input from stdion", ReadInput("default"))

	// read input with spaces
	stdin = strings.NewReader("input from stdion  \n")
	assert.Equal(t, "input from stdion", ReadInput("default"))
}

func TestMosaicString(t *testing.T) {
	assert.Equal(t, "****rX", MosaicString("IamMrX", 2))
	assert.Equal(t, "******", MosaicString("IamMrX", 10))
}

func TestGetLastChars(t *testing.T) {
	assert.Equal(t, "rX", GetLastChars("IamMrX", 2))
	assert.Equal(t, "******", GetLastChars("IamMrX", 10))
}

func TestNewConfigureCommandRun(t *testing.T) {

	testCases := []struct {
		name            string
		args            []string
		flags           map[string]string
		configuration   *Configuration
		loadConfigErr   error
		expectedProfile string
		expectedMode    string
		expectError     bool
	}{
		{
			// 场景1: mode为空，使用default profile的模式
			name:  "空mode时，使用default profile的模式",
			args:  []string{},
			flags: map[string]string{},
			configuration: &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name: "default",
						Mode: AK,
					},
				},
			},
			loadConfigErr:   nil,
			expectedProfile: "default",
			expectedMode:    "AK",
			expectError:     false,
		},
		{
			// 场景2: mode为空，指定存在的profile
			name:  "空mode时，指定存在的profile",
			args:  []string{},
			flags: map[string]string{"profile": "test-profile"},
			configuration: &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name: "default",
						Mode: AK,
					},
					{
						Name: "test-profile",
						Mode: StsToken,
					},
				},
			},
			loadConfigErr:   nil,
			expectedProfile: "test-profile",
			expectedMode:    "StsToken",
			expectError:     false,
		},
		{
			// 场景3: mode不为空，覆盖profile的模式
			name:  "非空mode时，覆盖profile的模式",
			args:  []string{},
			flags: map[string]string{"profile": "test-profile", "mode": "RamRoleArn"},
			configuration: &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name: "test-profile",
						Mode: StsToken,
					},
				},
			},
			loadConfigErr:   nil,
			expectedProfile: "test-profile",
			expectedMode:    "RamRoleArn",
			expectError:     false,
		},
		{
			// 场景4: 配置加载失败时，mode保持为空
			name:            "配置加载失败时，mode为空",
			args:            []string{},
			flags:           map[string]string{"profile": "test-profile"},
			configuration:   nil,
			loadConfigErr:   fmt.Errorf("load configuration failed"),
			expectedProfile: "test-profile",
			expectedMode:    "",
			expectError:     false,
		},
		{
			// 场景5: profile为空且CurrentProfile为空，mode为空
			name:  "profile和CurrentProfile都为空，mode为空",
			args:  []string{},
			flags: map[string]string{},
			configuration: &Configuration{
				CurrentProfile: "",
				Profiles:       []Profile{},
			},
			loadConfigErr:   nil,
			expectedProfile: "",
			expectedMode:    "",
			expectError:     false,
		},
		{
			// 场景6: ��额外参数时应当返回错误
			name:            "有额外参数时返回错误",
			args:            []string{"extra-arg"},
			flags:           map[string]string{},
			configuration:   nil,
			loadConfigErr:   nil,
			expectedProfile: "",
			expectedMode:    "",
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 重置捕获的参数
			var capturedProfileName, capturedMode string
			var doConfigureCalled bool

			// Mock loadConfiguration 函数
			hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
				return func(path string) (*Configuration, error) {
					return tc.configuration, tc.loadConfigErr
				}
			}

			// Mock doConfigure 函数
			doConfigureProxy = func(ctx *cli.Context, profileName string, mode string) error {
				doConfigureCalled = true
				capturedProfileName = profileName
				capturedMode = mode
				return nil
			}

			// 准备CLI上下文
			buffer := new(bytes.Buffer)
			buffer2 := new(bytes.Buffer)
			ctx := cli.NewCommandContext(buffer, buffer2)
			AddFlags(ctx.Flags())
			for k, v := range tc.flags {
				ctx.Flags().Get(k).SetAssigned(true)
				ctx.Flags().Get(k).SetValue(v)
			}

			// 执行测试
			cmd := NewConfigureCommand()
			err := cmd.Run(ctx, tc.args)

			// 验证结果
			if tc.expectError {
				if err == nil {
					t.Errorf("期望返回错误，但得到nil")
				}
				return
			}

			if err != nil {
				t.Errorf("期望无错误，但得到: %v", err)
			}

			if !doConfigureCalled {
				if !tc.expectError {
					t.Errorf("期望调用doConfigure，但未调用")
				}
				return
			}

			if capturedProfileName != tc.expectedProfile {
				t.Errorf("期望profile名称为 %s，但得到 %s", tc.expectedProfile, capturedProfileName)
			}

			if capturedMode != tc.expectedMode {
				t.Errorf("期望mode为 %s，但得到 %s", tc.expectedMode, capturedMode)
			}
		})
	}
}

func TestDetectPortUse(t *testing.T) {
	// Test case 1: 正常情况 - 应该能找到可用端口
	t.Run("normal case - find available port", func(t *testing.T) {
		// 使用一个较大的端口范围来确保能找到可用端口
		port, err := detectPortUse(15000, 15010)
		assert.NoError(t, err)
		assert.True(t, port >= 15000 && port <= 15010, "返回的端口应该在指定范围内")
		assert.Greater(t, port, 0, "端口号应该大于0")
	})

	// Test case 2: 单个端口范围
	t.Run("single port range", func(t *testing.T) {
		// 使用单个端口进行测试
		port, err := detectPortUse(15020, 15020)
		if err != nil {
			// 如果端口被占用，这是正常的
			assert.Contains(t, err.Error(), "no available port found")
		} else {
			assert.Equal(t, 15020, port)
		}
	})

	// Test case 3: 无效范围 - start > end
	t.Run("invalid range - start greater than end", func(t *testing.T) {
		port, err := detectPortUse(15030, 15025)
		assert.Error(t, err)
		assert.Equal(t, 0, port)
		assert.Contains(t, err.Error(), "no available port found in range 15030-15025")
	})

	// Test case 4: 占用端口后测试
	t.Run("port already in use", func(t *testing.T) {
		// 先占用一个端口
		testPort := 15040
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", testPort))
		if err != nil {
			// 如果端口已被占用，跳过此测试
			t.Skipf("Port %d already in use, skipping test", testPort)
			return
		}
		defer ln.Close()

		// 现在测试在只包含被占用端口的范围内查找
		port, err := detectPortUse(testPort, testPort)
		assert.Error(t, err)
		assert.Equal(t, 0, port)
		assert.Contains(t, err.Error(), fmt.Sprintf("no available port found in range %d-%d", testPort, testPort))
	})

	// Test case 5: 边界值测试
	t.Run("boundary values", func(t *testing.T) {
		// 测试端口号1（通常需要root权限）
		port, err := detectPortUse(1, 1)
		if err != nil {
			// 预期的错误，因为通常需要特殊权限
			assert.Contains(t, err.Error(), "no available port found")
		}
		assert.True(t, port == 0 || port == 1)

		// 测试一个较大的端口号
		port, err = detectPortUse(65534, 65534)
		if err != nil {
			assert.Contains(t, err.Error(), "no available port found")
		}
		assert.True(t, port == 0 || port == 65534)
	})

	// Test case 6: 中等范围测试
	t.Run("medium range", func(t *testing.T) {
		// 测试一个中等大小的范围
		port, err := detectPortUse(15050, 15060)
		assert.NoError(t, err)
		assert.True(t, port >= 15050 && port <= 15060)
	})

	// Test case 7: 端口范围内部分端口被占用
	t.Run("some ports in range are occupied", func(t *testing.T) {
		startPort := 15070
		endPort := 15075

		// 占用范围内的前几个端口
		var listeners []net.Listener
		for i := 0; i < 3; i++ {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", startPort+i))
			if err == nil {
				listeners = append(listeners, ln)
			}
		}

		// 确保在测试结束后关闭所有监听器
		defer func() {
			for _, ln := range listeners {
				ln.Close()
			}
		}()

		// 应该能找到一个可用的端口（范围内未被占用的端口）
		port, err := detectPortUse(startPort, endPort)
		if err != nil {
			// 如果所有端口都被占用，这也是可能的
			assert.Contains(t, err.Error(), "no available port found")
		} else {
			assert.True(t, port >= startPort && port <= endPort)
			// 验证返回的端口确实可用
			testLn, testErr := net.Listen("tcp", fmt.Sprintf(":%d", port))
			assert.NoError(t, testErr)
			if testLn != nil {
				testLn.Close()
			}
		}
	})

	// Test case 8: 零值测试
	t.Run("zero values", func(t *testing.T) {
		port, err := detectPortUse(0, 0)
		// 当请求端口0时，函数会尝试绑定端口0
		// 如果成功，函数返回0（请求的端口号），但系统实际分配了一个随机端口
		// 如果失败，则返回错误
		if err != nil {
			assert.Equal(t, 0, port)
			assert.Contains(t, err.Error(), "no available port found in range 0-0")
		} else {
			// 函数成功绑定了端口0，返回请求的端口号0
			assert.Equal(t, 0, port, "函数应该返回请求的端口号0")
		}
	})

	// Test case 9: 负值测试
	t.Run("negative values", func(t *testing.T) {
		port, err := detectPortUse(-1, -1)
		assert.Error(t, err)
		assert.Equal(t, 0, port)
		assert.Contains(t, err.Error(), "no available port found")
	})

	// Test case 10: 函数返回值验证
	t.Run("return value validation", func(t *testing.T) {
		port, err := detectPortUse(15080, 15090)
		if err == nil {
			// 如果成功找到端口，验证端口确实可用
			ln, testErr := net.Listen("tcp", fmt.Sprintf(":%d", port))
			assert.NoError(t, testErr, "返回的端口应该是可用的")
			if ln != nil {
				ln.Close()
			}
		} else {
			// 如果没有找到可用端口，确保返回值为0
			assert.Equal(t, 0, port, "当没有找到可用端口时，应该返回0")
		}
	})
}
