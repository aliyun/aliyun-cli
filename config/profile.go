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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/util"
	credentialsv2 "github.com/aliyun/credentials-go/credentials"
)

type AuthenticateMode string

const (
	AK                  = AuthenticateMode("AK")
	StsToken            = AuthenticateMode("StsToken")
	RamRoleArn          = AuthenticateMode("RamRoleArn")
	EcsRamRole          = AuthenticateMode("EcsRamRole")
	RsaKeyPair          = AuthenticateMode("RsaKeyPair")
	RamRoleArnWithEcs   = AuthenticateMode("RamRoleArnWithRoleName")
	ChainableRamRoleArn = AuthenticateMode("ChainableRamRoleArn")
	External            = AuthenticateMode("External")
	CredentialsURI      = AuthenticateMode("CredentialsURI")
	OIDC                = AuthenticateMode("OIDC")
)

type Profile struct {
	Name            string           `json:"name"`
	Mode            AuthenticateMode `json:"mode"`
	AccessKeyId     string           `json:"access_key_id"`
	AccessKeySecret string           `json:"access_key_secret"`
	StsToken        string           `json:"sts_token"`
	StsRegion       string           `json:"sts_region"`
	RamRoleName     string           `json:"ram_role_name"`
	RamRoleArn      string           `json:"ram_role_arn"`
	RoleSessionName string           `json:"ram_session_name"`
	SourceProfile   string           `json:"source_profile"`
	PrivateKey      string           `json:"private_key"`
	KeyPairName     string           `json:"key_pair_name"`
	ExpiredSeconds  int              `json:"expired_seconds"`
	Verified        string           `json:"verified"`
	RegionId        string           `json:"region_id"`
	OutputFormat    string           `json:"output_format"`
	Language        string           `json:"language"`
	Site            string           `json:"site"`
	ReadTimeout     int              `json:"retry_timeout"`
	ConnectTimeout  int              `json:"connect_timeout"`
	RetryCount      int              `json:"retry_count"`
	ProcessCommand  string           `json:"process_command"`
	CredentialsURI  string           `json:"credentials_uri"`
	OIDCProviderARN string           `json:"oidc_provider_arn"`
	OIDCTokenFile   string           `json:"oidc_token_file"`
	parent          *Configuration   //`json:"-"`
}

func NewProfile(name string) Profile {
	return Profile{
		Name:         name,
		Mode:         "",
		OutputFormat: "json",
		Language:     i18n.GetLanguage(),
	}
}

func (cp *Profile) Validate() error {
	if cp.RegionId == "" {
		return fmt.Errorf("region can't be empty")
	}

	if !IsRegion(cp.RegionId) {
		return fmt.Errorf("invalid region %s", cp.RegionId)
	}

	if cp.Mode == "" {
		return fmt.Errorf("profile %s is not configure yet, run `aliyun configure --profile %s` first", cp.Name, cp.Name)
	}

	switch cp.Mode {
	case AK:
		return cp.ValidateAK()
	case StsToken:
		err := cp.ValidateAK()
		if err != nil {
			return err
		}
		if cp.StsToken == "" {
			return fmt.Errorf("invalid sts_token")
		}
	case RamRoleArn:
		err := cp.ValidateAK()
		if err != nil {
			return err
		}
		if cp.RamRoleArn == "" {
			return fmt.Errorf("invalid ram_role_arn")
		}
		if cp.RoleSessionName == "" {
			return fmt.Errorf("invalid role_session_name")
		}
	case EcsRamRole, RamRoleArnWithEcs:
	case RsaKeyPair:
		if cp.PrivateKey == "" {
			return fmt.Errorf("invalid private_key")
		}
		if cp.KeyPairName == "" {
			return fmt.Errorf("invalid key_pair_name")
		}
	case External:
		if cp.ProcessCommand == "" {
			return fmt.Errorf("invalid process_command")
		}
	case CredentialsURI:
		if cp.CredentialsURI == "" {
			return fmt.Errorf("invalid credentials_uri")
		}
	case OIDC:
		if cp.OIDCProviderARN == "" {
			return fmt.Errorf("invalid oidc_provider_arn")
		}
		if cp.OIDCTokenFile == "" {
			return fmt.Errorf("invalid oidc_token_file")
		}
		if cp.RamRoleArn == "" {
			return fmt.Errorf("invalid ram_role_arn")
		}
		if cp.RoleSessionName == "" {
			return fmt.Errorf("invalid role_session_name")
		}
	case ChainableRamRoleArn:
		if cp.SourceProfile == "" {
			return fmt.Errorf("invalid source_profile")
		}
		if cp.RamRoleArn == "" {
			return fmt.Errorf("invalid ram_role_arn")
		}
		if cp.RoleSessionName == "" {
			return fmt.Errorf("invalid role_session_name")
		}
	default:
		return fmt.Errorf("invalid mode: %s", cp.Mode)
	}
	return nil
}

func (cp *Profile) GetParent() *Configuration {
	return cp.parent
}

func (cp *Profile) OverwriteWithFlags(ctx *cli.Context) {
	cp.Mode = AuthenticateMode(ModeFlag(ctx.Flags()).GetStringOrDefault(string(cp.Mode)))
	cp.AccessKeyId = AccessKeyIdFlag(ctx.Flags()).GetStringOrDefault(cp.AccessKeyId)
	cp.AccessKeySecret = AccessKeySecretFlag(ctx.Flags()).GetStringOrDefault(cp.AccessKeySecret)
	cp.StsToken = StsTokenFlag(ctx.Flags()).GetStringOrDefault(cp.StsToken)
	cp.StsRegion = StsRegionFlag(ctx.Flags()).GetStringOrDefault(cp.StsRegion)
	cp.RamRoleName = RamRoleNameFlag(ctx.Flags()).GetStringOrDefault(cp.RamRoleName)
	cp.RamRoleArn = RamRoleArnFlag(ctx.Flags()).GetStringOrDefault(cp.RamRoleArn)
	cp.RoleSessionName = RoleSessionNameFlag(ctx.Flags()).GetStringOrDefault(cp.RoleSessionName)
	cp.KeyPairName = KeyPairNameFlag(ctx.Flags()).GetStringOrDefault(cp.KeyPairName)
	cp.PrivateKey = PrivateKeyFlag(ctx.Flags()).GetStringOrDefault(cp.PrivateKey)
	cp.RegionId = RegionFlag(ctx.Flags()).GetStringOrDefault(cp.RegionId)
	cp.Language = LanguageFlag(ctx.Flags()).GetStringOrDefault(cp.Language)
	cp.ReadTimeout = ReadTimeoutFlag(ctx.Flags()).GetIntegerOrDefault(cp.ReadTimeout)
	cp.ConnectTimeout = ConnectTimeoutFlag(ctx.Flags()).GetIntegerOrDefault(cp.ConnectTimeout)
	cp.RetryCount = RetryCountFlag(ctx.Flags()).GetIntegerOrDefault(cp.RetryCount)
	cp.ExpiredSeconds = ExpiredSecondsFlag(ctx.Flags()).GetIntegerOrDefault(cp.ExpiredSeconds)
	cp.ProcessCommand = ProcessCommandFlag(ctx.Flags()).GetStringOrDefault(cp.ProcessCommand)

	if cp.AccessKeyId == "" {
		cp.AccessKeyId = util.GetFromEnv("ALIBABACLOUD_ACCESS_KEY_ID", "ALICLOUD_ACCESS_KEY_ID", "ACCESS_KEY_ID")
	}

	if cp.AccessKeySecret == "" {
		cp.AccessKeySecret = util.GetFromEnv("ALIBABACLOUD_ACCESS_KEY_SECRET", "ALICLOUD_ACCESS_KEY_SECRET", "ACCESS_KEY_SECRET")
	}

	if cp.StsToken == "" {
		cp.StsToken = util.GetFromEnv("ALIBABACLOUD_SECURITY_TOKEN", "ALICLOUD_SECURITY_TOKEN", "SECURITY_TOKEN")
	}

	if cp.RegionId == "" {
		cp.RegionId = util.GetFromEnv("ALIBABACLOUD_REGION_ID", "ALICLOUD_REGION_ID", "REGION_ID", "REGION")
	}

	if cp.CredentialsURI == "" {
		cp.CredentialsURI = os.Getenv("ALIBABA_CLOUD_CREDENTIALS_URI")
	}

	if cp.OIDCProviderARN == "" {
		cp.OIDCProviderARN = util.GetFromEnv("ALIBABACLOUD_OIDC_PROVIDER_ARN", "ALIBABA_CLOUD_OIDC_PROVIDER_ARN")
	}

	if cp.OIDCTokenFile == "" {
		cp.OIDCTokenFile = util.GetFromEnv("ALIBABACLOUD_OIDC_TOKEN_FILE", "ALIBABA_CLOUD_OIDC_TOKEN_FILE")
	}

	if cp.RamRoleArn == "" {
		cp.RamRoleArn = util.GetFromEnv("ALIBABACLOUD_ROLE_ARN", "ALIBABA_CLOUD_ROLE_ARN")
	}

	AutoModeRecognition(cp)
}

func AutoModeRecognition(cp *Profile) {
	if cp.Mode != AuthenticateMode("") {
		return
	}
	if cp.AccessKeyId != "" && cp.AccessKeySecret != "" {
		cp.Mode = AK
		if cp.StsToken != "" {
			cp.Mode = StsToken
		} else if cp.RamRoleArn != "" {
			cp.Mode = RamRoleArn
		}
	} else if cp.PrivateKey != "" && cp.KeyPairName != "" {
		cp.Mode = RsaKeyPair
	} else if cp.RamRoleName != "" {
		cp.Mode = EcsRamRole
	} else if cp.ProcessCommand != "" {
		cp.Mode = External
	} else if cp.OIDCProviderARN != "" && cp.OIDCTokenFile != "" && cp.RamRoleArn != "" {
		cp.Mode = OIDC
	}
}

func (cp *Profile) ValidateAK() error {
	if len(cp.AccessKeyId) == 0 {
		return fmt.Errorf("invalid access_key_id: %s", cp.AccessKeyId)
	}
	if len(cp.AccessKeySecret) == 0 {
		return fmt.Errorf("invaild access_key_secret: %s", cp.AccessKeySecret)
	}
	return nil
}

func getSTSEndpoint(regionId string) string {
	if regionId != "" {
		return fmt.Sprintf("sts.%s.aliyuncs.com", regionId)
	}
	return "sts.aliyuncs.com"
}

func (cp *Profile) GetCredential(ctx *cli.Context, proxyHost *string) (cred credentialsv2.Credential, err error) {
	config := new(credentialsv2.Config)
	// The AK, StsToken are direct credential
	// Others are indirect credential
	cp.Validate()
	switch cp.Mode {
	case AK:
		if cp.AccessKeyId == "" || cp.AccessKeySecret == "" {
			err = fmt.Errorf("AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")
			return
		}

		if cp.RegionId == "" {
			err = fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
			return
		}

		config.SetType("access_key").
			SetAccessKeyId(cp.AccessKeyId).
			SetAccessKeySecret(cp.AccessKeySecret)

	case StsToken:
		config.SetType("sts").
			SetAccessKeyId(cp.AccessKeyId).
			SetAccessKeySecret(cp.AccessKeySecret).
			SetSecurityToken(cp.StsToken)

	case RamRoleArn:
		config.SetType("ram_role_arn").
			SetAccessKeyId(cp.AccessKeyId).
			SetAccessKeySecret(cp.AccessKeySecret).
			SetRoleArn(cp.RamRoleArn).
			SetRoleSessionName(cp.RoleSessionName).
			SetRoleSessionExpiration(cp.ExpiredSeconds).
			SetSTSEndpoint(getSTSEndpoint(cp.StsRegion))

		if cp.StsToken != "" {
			config.SetSecurityToken(cp.StsToken)
		}

	case EcsRamRole:
		config.SetType("ecs_ram_role").
			SetRoleName(cp.RamRoleName)

	case RsaKeyPair:
		config.SetType("rsa_key_pair").
			SetPrivateKeyFile(cp.PrivateKey).
			SetPublicKeyId(cp.KeyPairName).
			SetSessionExpiration(cp.ExpiredSeconds).
			SetSTSEndpoint(getSTSEndpoint(cp.StsRegion))

	case RamRoleArnWithEcs:
		config.SetType("ecs_ram_role").
			SetRoleName(cp.RamRoleName)
		client, err := credentialsv2.NewCredential(config)
		if err != nil {
			return nil, err
		}
		// 从 ECS RAM Role 获取中间 STS
		model, err := client.GetCredential()
		if err != nil {
			return nil, err
		}

		// 扮演最终角色
		config.SetType("ram_role_arn").
			SetAccessKeyId(*model.AccessKeyId).
			SetAccessKeySecret(*model.AccessKeySecret).
			SetSecurityToken(*model.SecurityToken).
			SetRoleArn(cp.RamRoleArn).
			SetRoleSessionName(cp.RoleSessionName).
			SetRoleSessionExpiration(cp.ExpiredSeconds).
			SetSTSEndpoint(getSTSEndpoint(cp.StsRegion))

	case ChainableRamRoleArn:
		profileName := cp.SourceProfile

		// 从 configuration 中重新获取 source profile
		source, loaded := cp.parent.GetProfile(profileName)
		if !loaded {
			err = fmt.Errorf("can not load the source profile: " + profileName)
			return
		}

		middle, err2 := source.GetCredential(ctx, proxyHost)
		if err2 != nil {
			err = err2
			return
		}

		// 从上游处获得中间 AK/STS
		model, err3 := middle.GetCredential()

		if err3 != nil {
			err = err3
			return
		}

		// 扮演最终角色
		config.SetType("ram_role_arn").
			SetAccessKeyId(*model.AccessKeyId).
			SetAccessKeySecret(*model.AccessKeySecret).
			SetRoleArn(cp.RamRoleArn).
			SetRoleSessionName(cp.RoleSessionName).
			SetRoleSessionExpiration(cp.ExpiredSeconds).
			SetSTSEndpoint(getSTSEndpoint(cp.StsRegion))

		if model.SecurityToken != nil {
			config.SetSecurityToken(*model.SecurityToken)
		}

	case External:
		args := strings.Fields(cp.ProcessCommand)
		cmd := exec.Command(args[0], args[1:]...)
		buf, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		// 解析得到新的 profile 配置
		err = json.Unmarshal(buf, cp)
		if err != nil {
			fmt.Println(cp.ProcessCommand)
			fmt.Println(string(buf))
			return nil, err
		}
		return cp.GetCredential(ctx, proxyHost)

	case CredentialsURI:
		uri := cp.CredentialsURI

		if uri == "" {
			uri = os.Getenv("ALIBABA_CLOUD_CREDENTIALS_URI")
		}

		if uri == "" {
			return nil, fmt.Errorf("invalid credentials uri")
		}

		res, err := http.Get(uri)
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 200 {
			return nil, fmt.Errorf("get credentials from %s failed, status code %d", uri, res.StatusCode)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, err
		}

		type Response struct {
			Code            string
			AccessKeyId     string
			AccessKeySecret string
			SecurityToken   string
			Expiration      string
		}
		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, fmt.Errorf("unmarshal credentials failed, the body %s", string(body))
		}

		if response.Code != "Success" {
			return nil, fmt.Errorf("get sts token err, Code is not Success")
		}

		config.SetType("sts").
			SetAccessKeyId(response.AccessKeyId).
			SetAccessKeySecret(response.AccessKeySecret).
			SetSecurityToken(response.SecurityToken)

	case OIDC:
		config.SetType("oidc_role_arn").
			SetOIDCProviderArn(cp.OIDCProviderARN).
			SetOIDCTokenFilePath(cp.OIDCTokenFile).
			SetRoleArn(cp.RamRoleArn).
			SetRoleSessionName(cp.RoleSessionName).
			SetSTSEndpoint(getSTSEndpoint(cp.StsRegion)).
			SetSessionExpiration(3600)

	default:
		return nil, fmt.Errorf("unexcepted certificate mode: %s", cp.Mode)
	}

	if proxyHost != nil {
		config.SetProxy(*proxyHost)
	} else {
		proxy := util.GetFromEnv("HTTPS_PROXY", "https_proxy")
		if proxy != "" {
			config.SetProxy(proxy)
		}
	}

	return credentialsv2.NewCredential(config)
}

func IsRegion(region string) bool {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9-]*$", region); !match {
		return false
	}
	return true
}
