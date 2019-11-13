package credentials

import (
	"errors"
	"os"
)

type instanceCredentialsProvider struct{}

var providerInstance = new(instanceCredentialsProvider)

func newInstanceCredentialsProvider() Provider {
	return &instanceCredentialsProvider{}
}

func (p *instanceCredentialsProvider) resolve() (*Configuration, error) {
	roleName, ok := os.LookupEnv(ENVEcsMetadata)
	if !ok {
		return nil, nil
	}
	if roleName == "" {
		return nil, errors.New(ENVEcsMetadata + " cannot be empty")
	}

	config := &Configuration{
		Type:     "ecs_ram_role",
		RoleName: roleName,
	}
	return config, nil
}
