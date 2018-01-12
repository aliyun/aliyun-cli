package core

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
)


type CertificateMode string

const (
	AK = CertificateMode("AK")
	EcsRamUser = CertificateMode("EcsRamUser")
)

type Profile struct {
	Name            string          `json:"name"`
	Mode            CertificateMode `json:"mode"`
	RamRole         string          `json:"ram_role"`
	AccessKeyId     string          `json:"access_key_id"`
	AccessKeySecret string          `json:"access_key_secret"`
	RegionId        string          `json:"region_id"`
	OutputFormat    string          `json:"output_format"`
	Language        string          `json:"language"`
}

func NewProfile(name string) (Profile) {
	return Profile {
		Name: name,
		Mode: AK,
		OutputFormat: "json",
		Language: "en-US",
	}
}

func (cp *Profile) GetClient() (*sdk.Client, error) {
	switch cp.Mode {
	case AK:
		return cp.GetClientByAK()
	case EcsRamUser:
		return cp.GetClientByEcsRamUser()
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

func (cp *Profile) GetClientByEcsRamUser() (*sdk.Client, error) {
	if cp.RamRole == "" {
		return nil, fmt.Errorf("RamUser is empty! run `aliyun configure` first")
	}

	cred := credentials.EcsInstanceCredential{
		RoleName: cp.RamRole,
	}
	config := sdk.NewConfig()
	client, err := sdk.NewClientWithOptions(cp.RegionId, config, &cred)
	return client, err
}