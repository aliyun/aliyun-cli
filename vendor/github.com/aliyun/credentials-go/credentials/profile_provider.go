package credentials

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	ini "gopkg.in/ini.v1"
)

type profileProvider struct {
	Profile string
}

var providerProfile = newProfileProvider()

var hookOS = func(goos string) string {
	return goos
}

var hookState = func(info os.FileInfo, err error) (os.FileInfo, error) {
	return info, err
}

// NewProfileProvider receive zero or more parameters,
// when length of name is 0, the value of field Profile will be "default",
// and when there are multiple inputs, the function will take the
// first one and  discard the other values.
func newProfileProvider(name ...string) Provider {
	p := new(profileProvider)
	if len(name) == 0 {
		p.Profile = "default"
	} else {
		p.Profile = name[0]
	}
	return p
}

// resolve implements the Provider interface
// when credential type is rsa_key_pair, the content of private_key file
// must be able to be parsed directly into the required string
// that NewRsaKeyPairCredential function needed
func (p *profileProvider) resolve() (*Configuration, error) {
	path, ok := os.LookupEnv(ENVCredentialFile)
	if !ok {
		path, err := checkDefaultPath()
		if err != nil {
			return nil, err
		}
		if path == "" {
			return nil, nil
		}
	} else if path == "" {
		return nil, errors.New(ENVCredentialFile + " cannot be empty")
	}

	value, section, err := getType(path, p.Profile)
	if err != nil {
		return nil, err
	}
	switch value.String() {
	case "access_key":
		config, err := getAccessKey(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	case "sts":
		config, err := getSTS(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	case "bearer":
		config, err := getBearerToken(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	case "ecs_ram_role":
		config, err := getEcsRAMRole(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	case "ram_role_arn":
		config, err := getRAMRoleArn(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	case "rsa_key_pair":
		config, err := getRSAKeyPair(section)
		if err != nil {
			return nil, err
		}
		return config, nil
	default:
		return nil, errors.New("Invalid type option, support: access_key, sts, ecs_ram_role, ram_role_arn, rsa_key_pair")
	}
}

func getRSAKeyPair(section *ini.Section) (*Configuration, error) {
	publicKeyID, err := section.GetKey("public_key_id")
	if err != nil {
		return nil, errors.New("Missing required public_key_id option in profile for rsa_key_pair")
	}
	if publicKeyID.String() == "" {
		return nil, errors.New("public_key_id cannot be empty")
	}
	privateKeyFile, err := section.GetKey("private_key_file")
	if err != nil {
		return nil, errors.New("Missing required private_key_file option in profile for rsa_key_pair")
	}
	if privateKeyFile.String() == "" {
		return nil, errors.New("private_key_file cannot be empty")
	}
	sessionExpiration, _ := section.GetKey("session_expiration")
	expiration := 0
	if sessionExpiration != nil {
		expiration, err = sessionExpiration.Int()
		if err != nil {
			return nil, errors.New("session_expiration must be an int")
		}
	}
	config := &Configuration{
		Type:              "rsa_key_pair",
		PublicKeyID:       publicKeyID.String(),
		PrivateKeyFile:    privateKeyFile.String(),
		SessionExpiration: expiration,
	}
	err = setRuntimeToConfig(config, section)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getRAMRoleArn(section *ini.Section) (*Configuration, error) {
	accessKeyID, err := section.GetKey("access_key_id")
	if err != nil {
		return nil, errors.New("Missing required access_key_id option in profile for ram_role_arn")
	}
	if accessKeyID.String() == "" {
		return nil, errors.New("access_key_id cannot be empty")
	}
	accessKeySecret, err := section.GetKey("access_key_secret")
	if err != nil {
		return nil, errors.New("Missing required access_key_secret option in profile for ram_role_arn")
	}
	if accessKeySecret.String() == "" {
		return nil, errors.New("access_key_secret cannot be empty")
	}
	roleArn, err := section.GetKey("role_arn")
	if err != nil {
		return nil, errors.New("Missing required role_arn option in profile for ram_role_arn")
	}
	if roleArn.String() == "" {
		return nil, errors.New("role_arn cannot be empty")
	}
	roleSessionName, err := section.GetKey("role_session_name")
	if err != nil {
		return nil, errors.New("Missing required role_session_name option in profile for ram_role_arn")
	}
	if roleSessionName.String() == "" {
		return nil, errors.New("role_session_name cannot be empty")
	}
	roleSessionExpiration, _ := section.GetKey("role_session_expiration")
	expiration := 0
	if roleSessionExpiration != nil {
		expiration, err = roleSessionExpiration.Int()
		if err != nil {
			return nil, errors.New("role_session_expiration must be an int")
		}
	}
	config := &Configuration{
		Type:                  "ram_role_arn",
		AccessKeyID:           accessKeyID.String(),
		AccessKeySecret:       accessKeySecret.String(),
		RoleArn:               roleArn.String(),
		RoleSessionName:       roleSessionName.String(),
		RoleSessionExpiration: expiration,
	}
	err = setRuntimeToConfig(config, section)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getEcsRAMRole(section *ini.Section) (*Configuration, error) {
	roleName, err := section.GetKey("role_name")
	if err != nil {
		return nil, errors.New("Missing required role_name option in profile for ecs_ram_role")
	}
	if roleName.String() == "" {
		return nil, errors.New("role_name cannot be empty")
	}
	config := &Configuration{
		Type:     "ecs_ram_role",
		RoleName: roleName.String(),
	}
	err = setRuntimeToConfig(config, section)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func getBearerToken(section *ini.Section) (*Configuration, error) {
	bearerToken, err := section.GetKey("bearer_token")
	if err != nil {
		return nil, errors.New("Missing required bearer_token option in profile for bearer")
	}
	if bearerToken.String() == "" {
		return nil, errors.New("bearer_token cannot be empty")
	}
	config := &Configuration{
		Type:        "bearer",
		BearerToken: bearerToken.String(),
	}
	return config, nil
}

func getSTS(section *ini.Section) (*Configuration, error) {
	accesskeyid, err := section.GetKey("access_key_id")
	if err != nil {
		return nil, errors.New("Missing required access_key_id option in profile for sts")
	}
	if accesskeyid.String() == "" {
		return nil, errors.New("access_key_id cannot be empty")
	}
	accessKeySecret, err := section.GetKey("access_key_secret")
	if err != nil {
		return nil, errors.New("Missing required access_key_secret option in profile for sts")
	}
	if accessKeySecret.String() == "" {
		return nil, errors.New("access_key_secret cannot be empty")
	}
	securityToken, err := section.GetKey("security_token")
	if err != nil {
		return nil, errors.New("Missing required security_token option in profile for sts")
	}
	if securityToken.String() == "" {
		return nil, errors.New("security_token cannot be empty")
	}
	config := &Configuration{
		Type:            "sts",
		AccessKeyID:     accesskeyid.String(),
		AccessKeySecret: accessKeySecret.String(),
		SecurityToken:   securityToken.String(),
	}
	return config, nil
}

func getAccessKey(section *ini.Section) (*Configuration, error) {
	accesskeyid, err := section.GetKey("access_key_id")
	if err != nil {
		return nil, errors.New("Missing required access_key_id option in profile for access_key")
	}
	if accesskeyid.String() == "" {
		return nil, errors.New("access_key_id cannot be empty")
	}
	accessKeySecret, err := section.GetKey("access_key_secret")
	if err != nil {
		return nil, errors.New("Missing required access_key_secret option in profile for access_key")
	}
	if accessKeySecret.String() == "" {
		return nil, errors.New("access_key_secret cannot be empty")
	}
	config := &Configuration{
		Type:            "access_key",
		AccessKeyID:     accesskeyid.String(),
		AccessKeySecret: accessKeySecret.String(),
	}
	return config, nil
}

func getType(path, profile string) (*ini.Key, *ini.Section, error) {
	ini, err := ini.Load(path)
	if err != nil {
		return nil, nil, errors.New("ERROR: Can not open file " + err.Error())
	}

	section, err := ini.GetSection(profile)
	if err != nil {
		return nil, nil, errors.New("ERROR: Can not load section " + err.Error())
	}

	value, err := section.GetKey("type")
	if err != nil {
		return nil, nil, errors.New("Missing required type option " + err.Error())
	}
	return value, section, nil
}

func getHomePath() string {
	if hookOS(runtime.GOOS) == "windows" {
		path, ok := os.LookupEnv("USERPROFILE")
		if !ok {
			return ""
		}
		return path
	}
	path, ok := os.LookupEnv("HOME")
	if !ok {
		return ""
	}
	return path
}

func checkDefaultPath() (path string, err error) {
	path = getHomePath()
	if path == "" {
		return "", errors.New("The default credential file path is invalid")
	}
	path = strings.Replace("~/.alibabacloud/credentials", "~", path, 1)
	_, err = hookState(os.Stat(path))
	if err != nil {
		return "", nil
	}
	return path, nil
}

func setRuntimeToConfig(config *Configuration, section *ini.Section) error {
	rawTimeout, _ := section.GetKey("timeout")
	rawConnectTimeout, _ := section.GetKey("connect_timeout")
	rawProxy, _ := section.GetKey("proxy")
	rawHost, _ := section.GetKey("host")
	if rawProxy != nil {
		config.Proxy = rawProxy.String()
	}
	if rawConnectTimeout != nil {
		connectTimeout, err := rawConnectTimeout.Int()
		if err != nil {
			return fmt.Errorf("Please set connect_timeout with an int value")
		}
		config.ConnectTimeout = connectTimeout
	}
	if rawTimeout != nil {
		timeout, err := rawTimeout.Int()
		if err != nil {
			return fmt.Errorf("Please set timeout with an int value")
		}
		config.Timeout = timeout
	}
	if rawHost != nil {
		config.Host = rawHost.String()
	}
	return nil
}
