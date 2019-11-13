package credentials

import (
	"errors"
	"os"
)

type envProvider struct{}

var providerEnv = new(envProvider)

const (
	// EnvVarAccessKeyID is a name of ALIBABA_CLOUD_ACCESS_KEY_ID
	EnvVarAccessKeyID = "ALIBABA_CLOUD_ACCESS_KEY_ID"
	// EnvVarAccessKeySecret is a name of ALIBABA_CLOUD_ACCESS_KEY_SECRET
	EnvVarAccessKeySecret = "ALIBABA_CLOUD_ACCESS_KEY_SECRET"
)

func newEnvProvider() Provider {
	return &envProvider{}
}

func (p *envProvider) resolve() (*Configuration, error) {
	accessKeyID, ok1 := os.LookupEnv(EnvVarAccessKeyID)
	accessKeySecret, ok2 := os.LookupEnv(EnvVarAccessKeySecret)
	if !ok1 || !ok2 {
		return nil, nil
	}
	if accessKeyID == "" {
		return nil, errors.New(EnvVarAccessKeyID + " cannot be empty")
	}
	if accessKeySecret == "" {
		return nil, errors.New(EnvVarAccessKeySecret + " cannot be empty")
	}
	config := &Configuration{
		Type:            "access_key",
		AccessKeyID:     accessKeyID,
		AccessKeySecret: accessKeySecret,
	}
	return config, nil
}
