/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"net/http"
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"encoding/json"
	"github.com/jmespath/go-jmespath"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/signers"
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
	Site            string           `json:"Site"`
}

func NewProfile(name string) (Profile) {
	return Profile{
		Name:         name,
		Mode:         AK,
		OutputFormat: "json",
		Language:     "en",
	}
}

func (cp *Profile) Validate() error {
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
			return fmt.Errorf("invailed ram_role_name")
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
	if cp.RegionId == "" {
		return fmt.Errorf("region can't be empty")
	}
	return nil
}

func (cp *Profile) OverwriteWithFlags(ctx *cli.Context) {
	cp.Mode = AuthenticateMode(ModeFlag.GetValueOrDefault(ctx, string(cp.Mode)))
	cp.AccessKeyId = AccessKeyIdFlag.GetValueOrDefault(ctx, cp.AccessKeyId)
	cp.AccessKeySecret = AccessKeySecretFlag.GetValueOrDefault(ctx, cp.AccessKeySecret)
	cp.StsToken = StsTokenFlag.GetValueOrDefault(ctx, cp.StsToken)
	cp.RamRoleName = RamRoleNameFlag.GetValueOrDefault(ctx, cp.RamRoleName)
	cp.RamRoleArn = RamRoleArnFlag.GetValueOrDefault(ctx, cp.RamRoleArn)
	cp.RoleSessionName = RoleSessionNameFlag.GetValueOrDefault(ctx, cp.RoleSessionName)
	cp.KeyPairName = KeyPairNameFlag.GetValueOrDefault(ctx, cp.KeyPairName)
	cp.PrivateKey = PrivateKeyFlag.GetValueOrDefault(ctx, cp.PrivateKey)
	cp.RegionId = RegionFlag.GetValueOrDefault(ctx, cp.RegionId)
	cp.Language = LanguageFlag.GetValueOrDefault(ctx, cp.Language)

	if cp.AccessKeyId != "" && cp.AccessKeySecret != "" {
		cp.Mode = AK
		if cp.StsToken != "" {
			cp.Mode = StsToken
		} else if cp.RamRoleArn != "" {
			cp.Mode = RamRoleArn
		}
	}
	if cp.PrivateKey != "" && cp.KeyPairName != "" {
		cp.Mode = RsaKeyPair
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

func (cp *Profile) GetClient() (*sdk.Client, error) {
	switch cp.Mode {
	case AK:
		return cp.GetClientByAK()
	case StsToken:
		return cp.GetClientBySts()
	case RamRoleArn:
		return cp.GetClientByRoleArn()
	case EcsRamRole:
		return cp.GetClientByEcsRamRole()
	case RsaKeyPair:
		return cp.GetClientByPrivateKey()
	default:
		return nil, fmt.Errorf("unexcepted certificate mode: %s", cp.Mode)
	}
}

func (cp *Profile) GetSessionCredential() (*signers.SessionCredential, error) {
	switch cp.Mode {
	case AK:
		return &signers.SessionCredential{
			AccessKeyId:cp.AccessKeyId,
			AccessKeySecret:cp.AccessKeySecret,
		}, nil
	case StsToken:
		return &signers.SessionCredential{
			AccessKeyId:cp.AccessKeyId,
			AccessKeySecret:cp.AccessKeySecret,
			StsToken:cp.StsToken,
		}, nil
	case RamRoleArn:
		return cp.GetSessionCredentialByRoleArn()
	case EcsRamRole:
		return cp.GetSessionCredentialByEcsRamRole()
	default:
		return nil, fmt.Errorf("unsupported mode '%s' to GetSessionCredential", cp.Mode)
	}
}

func (cp *Profile) GetClientByAK() (*sdk.Client, error) {
	if cp.AccessKeyId == "" || cp.AccessKeySecret == "" {
		return nil, fmt.Errorf("AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")
	}

	if cp.RegionId == "" {
		return nil, fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
	}

	client, err := sdk.NewClientWithAccessKey(cp.RegionId, cp.AccessKeyId, cp.AccessKeySecret)
	return client, err
}

func (cp *Profile) GetClientBySts() (*sdk.Client, error) {
	cred := credentials.NewStsTokenCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.StsToken)
	config := sdk.NewConfig()
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
	request.DurationSeconds = requests.NewInteger(600)
	request.Scheme = "https"

	response, err := client.AssumeRole(request)
	if err != nil {
		return nil, fmt.Errorf("sts:AssumeRole() failed %s", err)
	}

	return &signers.SessionCredential{
		AccessKeyId: response.Credentials.AccessKeyId,
		AccessKeySecret: response.Credentials.AccessKeySecret,
		StsToken: response.Credentials.AccessKeySecret,
	}, nil
}


func (cp *Profile) GetClientByRoleArn() (*sdk.Client, error) {
	sc, err := cp.GetSessionCredentialByRoleArn()
	if err != nil {
		return nil, fmt.Errorf("get session credential failed %s", err)
	}
	config := sdk.NewConfig()
	cred := credentials.NewStsTokenCredential(sc.AccessKeyId, sc.AccessKeySecret, sc.StsToken)
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetSessionCredentialByEcsRamRole() (*signers.SessionCredential, error) {
	if cp.RamRoleName == "" {
		return nil, fmt.Errorf("RamRole is empty! run `aliyun configure` first")
	}

	requestUrl := "http://100.100.100.200/latest/meta-data/ram/security-credentials/" + cp.RamRoleName
	httpRequest, err := http.NewRequest(requests.GET, requestUrl, strings.NewReader(""))
	if err != nil {
		return nil, fmt.Errorf("new http request failed %s", err)
	}

	httpClient := &http.Client{}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed get credentials from meta-data %s, please check RAM settings", err)
	}

	response := responses.NewCommonResponse()
	err = responses.Unmarshal(response, httpResponse, "")

	if response.GetHttpStatus() != http.StatusOK {
		return nil, fmt.Errorf("get meta-data status=%d please check RAM settings", response.GetHttpStatus())
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
		AccessKeyId: accessKeyId.(string),
		AccessKeySecret: accessKeySecret.(string),
		StsToken: securityToken.(string),
	}, nil
}


func (cp *Profile) GetClientByEcsRamRole() (*sdk.Client, error) {
	sc, err := cp.GetSessionCredentialByEcsRamRole()
	if err != nil {
		return nil, fmt.Errorf("get session credential failed %s", err)
	}

	cred := credentials.NewStsTokenCredential(sc.AccessKeyId, sc.AccessKeySecret, sc.StsToken)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}

func (cp *Profile) GetClientByPrivateKey() (*sdk.Client, error) {
	cred := credentials.NewRsaKeyPairCredential(cp.PrivateKey, cp.KeyPairName, cp.ExpiredSeconds)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, cred)
	return client, err
}
