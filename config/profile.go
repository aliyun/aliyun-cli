package config
/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)

type CertificateMode string

const (
	AK = CertificateMode("AK")
	StsToken = CertificateMode("StsToken")
	RamRoleArn = CertificateMode("RamRoleArn")
	EcsRamRole = CertificateMode("EcsRamRole")
	RsaKeyPair = CertificateMode("RsaKeyPair")
)

type Profile struct {
	Name            string          `json:"name"`
	Mode            CertificateMode `json:"mode"`
	AccessKeyId     string          `json:"access_key_id"`
	AccessKeySecret string          `json:"access_key_secret"`
	StsToken		string			`json:"sts_token"`
	RamRoleName		string          `json:"ram_role_name"`
	RamRoleArn		string			`json:"ram_role_arn"`
	RoleSessionName	string 			`json:"ram_session_name"`
	PrivateKey		string 			`json:"private_key"`
	KeyPairName		string 			`json:"key_pair_name"`
	ExpiredSeconds	int				`json:"expired_seconds"`
	Verified		string			`json:"verified"`
	RegionId        string          `json:"region_id"`
	OutputFormat    string          `json:"output_format"`
	Language        string          `json:"language"`
}

func NewProfile(name string) (Profile) {
	return Profile {
		Name: name,
		Mode: AK,
		OutputFormat: "json",
		Language: "en",
	}
}

func (cp *Profile) Validate() error {
	if cp.Mode == "" {
		return fmt.Errorf("not configure yet")
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

func (cp *Profile) ValidateAK() error {
	if len(cp.AccessKeyId) != 16 {
		return fmt.Errorf("invalid access_key_id: %s", cp.AccessKeyId)
	}
	if len(cp.AccessKeySecret) != 30 {
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

func (cp *Profile) GetClientByEcsRamRole() (*sdk.Client, error) {
	if cp.RamRoleName == "" {
		return nil, fmt.Errorf("RamRole is empty! run `aliyun configure` first")
	}

	cred := credentials.NewStsRoleNameOnEcsCredential(cp.RamRoleName)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, &cred)
	return client, err
}

func (cp *Profile) GetClientBySts() (*sdk.Client, error) {
	cred := credentials.NewStsTokenCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.StsToken)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, &cred)
	return client, err
}

func (cp *Profile) GetClientByRoleArn() (*sdk.Client, error) {
	cred := credentials.NewStsRoleArnCredential(cp.AccessKeyId, cp.AccessKeySecret, cp.RamRoleArn, cp.RoleSessionName, cp.ExpiredSeconds)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, &cred)
	return client, err
}

func (cp *Profile) GetClientByPrivateKey() (*sdk.Client, error) {
	cred := credentials.NewRsaKeyPairCredential(cp.PrivateKey, cp.KeyPairName, cp.ExpiredSeconds)
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, &cred)
	return client, err
}



