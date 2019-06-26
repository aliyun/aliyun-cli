// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/signers"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/aliyun-cli/cli"
	jmespath "github.com/jmespath/go-jmespath"
)

type AuthenticateMode string

const (
	AK         = AuthenticateMode("AK")
	StsToken   = AuthenticateMode("StsToken")
	RamRoleArn = AuthenticateMode("RamRoleArn")
	EcsRamRole = AuthenticateMode("EcsRamRole")
	RsaKeyPair = AuthenticateMode("RsaKeyPair")
)

type Profile struct {
	Name            string           `json:"name"`
	Mode            AuthenticateMode `json:"mode"`
	AccessKeyId     string           `json:"access_key_id"`
	AccessKeySecret string           `json:"access_key_secret"`
	StsToken        string           `json:"sts_token"`
	RamRoleName     string           `json:"ram_role_name"`
	RamRoleArn      string           `json:"ram_role_arn"`
	RoleSessionName string           `json:"ram_session_name"`
	PrivateKey      string           `json:"private_key"`
	KeyPairName     string           `json:"key_pair_name"`
	ExpiredSeconds  int              `json:"expired_seconds"`
	Verified        string           `json:"verified"`
	RegionId        string           `json:"region_id"`
	OutputFormat    string           `json:"output_format"`
	Language        string           `json:"language"`
	Site            string           `json:"site"`
	RetryTimeout    int              `json:"retry_timeout"`
	RetryCount      int              `json:"retry_count"`
	parent          *Configuration   //`json:"-"`
}

var hookAssumeRole = func(fn func(request *sts.AssumeRoleRequest) (response *sts.AssumeRoleResponse, err error)) func(request *sts.AssumeRoleRequest) (response *sts.AssumeRoleResponse, err error) {
	return fn
}

var hookHTTPGet = func(fn func(url string) (resp *http.Response, err error)) func(url string) (resp *http.Response, err error) {
	return fn
}

var hookUnmarshal = func(fn func(response responses.AcsResponse, httpResponse *http.Response, format string) (err error)) func(response responses.AcsResponse, httpResponse *http.Response, format string) (err error) {
	return fn
}

func NewProfile(name string) Profile {
	return Profile{
		Name:         name,
		Mode:         AK,
		OutputFormat: "json",
		Language:     "en",
	}
}

func (cp *Profile) Validate() error {

	if cp.RegionId == "" {
		return fmt.Errorf("region can't be empty")
	}

	if !IsRegion(cp.RegionId) {
		return fmt.Errorf("invailed region %s", cp.RegionId)
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
			return fmt.Errorf("invailed sts_token")
		}
	case RamRoleArn:
		err := cp.ValidateAK()
		if err != nil {
			return err
		}
		if cp.RamRoleArn == "" {
			return fmt.Errorf("invailed ram_role_arn")
		}
		if cp.RoleSessionName == "" {
			return fmt.Errorf("invailed role_session_name")
		}
	case EcsRamRole:
		if cp.RamRoleName == "" {
			//return fmt.Errorf("invailed ram_role_name")
		}
	case RsaKeyPair:
		if cp.PrivateKey == "" {
			return fmt.Errorf("invailed private_key")
		}
		if cp.KeyPairName == "" {
			return fmt.Errorf("invailed key_pair_name")
		}
	default:
		return fmt.Errorf("invailed mode: %s", cp.Mode)
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
	cp.RamRoleName = RamRoleNameFlag(ctx.Flags()).GetStringOrDefault(cp.RamRoleName)
	cp.RamRoleArn = RamRoleArnFlag(ctx.Flags()).GetStringOrDefault(cp.RamRoleArn)
	cp.RoleSessionName = RoleSessionNameFlag(ctx.Flags()).GetStringOrDefault(cp.RoleSessionName)
	cp.KeyPairName = KeyPairNameFlag(ctx.Flags()).GetStringOrDefault(cp.KeyPairName)
	cp.PrivateKey = PrivateKeyFlag(ctx.Flags()).GetStringOrDefault(cp.PrivateKey)
	cp.RegionId = RegionFlag(ctx.Flags()).GetStringOrDefault(cp.RegionId)
	cp.Language = LanguageFlag(ctx.Flags()).GetStringOrDefault(cp.Language)
	cp.RetryTimeout = RetryTimeoutFlag(ctx.Flags()).GetIntegerOrDefault(cp.RetryTimeout)
	cp.RetryCount = RetryTimeoutFlag(ctx.Flags()).GetIntegerOrDefault(cp.RetryCount)

	if cp.AccessKeyId == "" {
		cp.AccessKeyId = os.Getenv("ACCESS_KEY_ID")
	}

	if cp.AccessKeySecret == "" {
		cp.AccessKeySecret = os.Getenv("ACCESS_KEY_SECRET")
	}

	if cp.StsToken == "" {
		cp.StsToken = os.Getenv("SECURITY_TOKEN")
	}

	if cp.RegionId == "" {
		cp.RegionId = os.Getenv("REGION")
	}

	//TODO:remove code below
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

func (cp *Profile) GetClient(ctx *cli.Context) (*sdk.Client, error) {
	config := sdk.NewConfig()
	if cp.RetryTimeout > 0 {
		config.WithTimeout(time.Duration(cp.RetryTimeout) * time.Second)
	}
	if cp.RetryCount > 0 {
		config.WithMaxRetryTime(cp.RetryCount)
	}
	if SkipSecureVerify(ctx.Flags()).IsAssigned() {
		config.HttpTransport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	switch cp.Mode {
	case AK:
		return cp.GetClientByAK(config)
	case StsToken:
		return cp.GetClientBySts(config)
	case RamRoleArn:
		return cp.GetClientByRoleArn(config)
	case EcsRamRole:
		return cp.GetClientByEcsRamRole(config)
	case RsaKeyPair:
		return cp.GetClientByPrivateKey(config)
	default:
		return nil, fmt.Errorf("unexcepted certificate mode: %s", cp.Mode)
	}
}

func (cp *Profile) GetSessionCredential() (*signers.SessionCredential, error) {
	switch cp.Mode {
	case AK:
		return &signers.SessionCredential{
			AccessKeyId:     cp.AccessKeyId,
			AccessKeySecret: cp.AccessKeySecret,
		}, nil
	case StsToken:
		return &signers.SessionCredential{
			AccessKeyId:     cp.AccessKeyId,
			AccessKeySecret: cp.AccessKeySecret,
			StsToken:        cp.StsToken,
		}, nil
	case RamRoleArn:
		return cp.GetSessionCredentialByRoleArn()
	case EcsRamRole:
		return cp.GetSessionCredentialByEcsRamRole()
	default:
		return nil, fmt.Errorf("unsupported mode '%s' to GetSessionCredential", cp.Mode)
	}
}

func (cp *Profile) GetClientByAK(config *sdk.Config) (*sdk.Client, error) {
	if cp.AccessKeyId == "" || cp.AccessKeySecret == "" {
		return nil, fmt.Errorf("AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")
	}

	if cp.RegionId == "" {
		return nil, fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
	}

	cred := credentials.NewAccessKeyCredential(cp.AccessKeyId, cp.AccessKeySecret)
	config.UserAgent = userAgent
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetClientBySts(config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewStsTokenCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.StsToken)
	config.UserAgent = userAgent
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetSessionCredentialByRoleArn() (*signers.SessionCredential, error) {
	client, err := sts.NewClientWithAccessKey(cp.RegionId, cp.AccessKeyId, cp.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("new sts client failed %s", err)
	}
	if client == nil {
		return nil, fmt.Errorf("new sts client with nil")
	}

	request := sts.CreateAssumeRoleRequest()
	request.RoleArn = cp.RamRoleArn
	request.RoleSessionName = cp.RoleSessionName
	request.DurationSeconds = requests.NewInteger(900)
	request.Scheme = "https"

	response, err := hookAssumeRole(client.AssumeRole)(request)
	if err != nil {
		return nil, fmt.Errorf("sts:AssumeRole() failed %s", err)
	}

	return &signers.SessionCredential{
		AccessKeyId:     response.Credentials.AccessKeyId,
		AccessKeySecret: response.Credentials.AccessKeySecret,
		StsToken:        response.Credentials.SecurityToken,
	}, nil
}

func (cp *Profile) GetClientByRoleArn(config *sdk.Config) (*sdk.Client, error) {
	sc, err := cp.GetSessionCredentialByRoleArn()
	if err != nil {
		return nil, fmt.Errorf("get session credential failed %s", err)
	}
	cred := credentials.NewStsTokenCredential(sc.AccessKeyId, sc.AccessKeySecret, sc.StsToken)
	config.UserAgent = userAgent
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetSessionCredentialByEcsRamRole() (*signers.SessionCredential, error) {
	httpClient := &http.Client{}

	baseURL := "http://100.100.100.200/latest/meta-data/ram/security-credentials/"
	ecsRamRoleName := cp.RamRoleName
	if ecsRamRoleName == "" {
		resp, err := hookHTTPGet(httpClient.Get)(baseURL)
		if err != nil {
			return nil, fmt.Errorf("Get default RamRole error: %s. Or Run `aliyun configure` to configure it.", err.Error())
		}

		response := responses.NewCommonResponse()
		err = hookUnmarshal(responses.Unmarshal)(response, resp, "")

		if response.GetHttpStatus() != http.StatusOK {
			return nil, fmt.Errorf("Get meta-data status=%d please check RAM settings. Or Run `aliyun configure` to configure it.", response.GetHttpStatus())
		}

		ecsRamRoleName = response.GetHttpContentString()
	}

	requestUrl := baseURL + ecsRamRoleName
	httpRequest, err := http.NewRequest(requests.GET, requestUrl, strings.NewReader(""))
	if err != nil {
		return nil, fmt.Errorf("new http request failed %s", err)
	}

	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed get credentials from meta-data %s, please check RAM settings", err)
	}

	response := responses.NewCommonResponse()
	err = responses.Unmarshal(response, httpResponse, "")

	if response.GetHttpStatus() != http.StatusOK {
		return nil, fmt.Errorf("get meta-data with role %s ,status=%d, please check RAM settings", ecsRamRoleName, response.GetHttpStatus())
	}

	var data interface{}
	err = json.Unmarshal(response.GetHttpContentBytes(), &data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal meta-data fail %s", err)
	}

	code, err := jmespath.Search("Code", data)
	if err != nil {
		return nil, fmt.Errorf("read code from meta-data failed %s", err)
	}
	if code.(string) != "Success" {
		return nil, fmt.Errorf("unexcepted code = %s", code)
	}
	accessKeyId, err := jmespath.Search("AccessKeyId", data)
	if err != nil || accessKeyId == "" {
		return nil, fmt.Errorf("read AccessKeyId from meta-data failed %s", err)
	}
	accessKeySecret, err := jmespath.Search("AccessKeySecret", data)
	if err != nil || accessKeySecret == "" {
		return nil, fmt.Errorf("read AccessKeySecret from meta-data failed %s", err)
	}
	securityToken, err := jmespath.Search("SecurityToken", data)
	if err != nil || securityToken == "" {
		return nil, fmt.Errorf("read SecurityToken from meta-data failed %s", err)
	}
	return &signers.SessionCredential{
		AccessKeyId:     accessKeyId.(string),
		AccessKeySecret: accessKeySecret.(string),
		StsToken:        securityToken.(string),
	}, nil
}

func (cp *Profile) GetClientByEcsRamRole(config *sdk.Config) (*sdk.Client, error) {
	sc, err := cp.GetSessionCredentialByEcsRamRole()
	if err != nil {
		return nil, fmt.Errorf("get session credential failed %s", err)
	}

	cred := credentials.NewStsTokenCredential(sc.AccessKeyId, sc.AccessKeySecret, sc.StsToken)
	config.UserAgent = userAgent
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetClientByPrivateKey(config *sdk.Config) (*sdk.Client, error) {
	cred := credentials.NewRsaKeyPairCredential(cp.PrivateKey, cp.KeyPairName, cp.ExpiredSeconds)
	config.UserAgent = userAgent
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func IsRegion(region string) bool {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9-]*$", region); !match {
		return false
	}
	return true
}
