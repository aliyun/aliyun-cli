/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoConfigureList(t *testing.T) {
	w := new(bytes.Buffer)
	originhook := hookLoadConfiguration
	defer func() {
		hookLoadConfiguration = originhook
	}()

	//testcase 1
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
			return Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					Profile{
						Name:            "default",
						Mode:            AK,
						AccessKeyId:     "default_aliyun_access_key_id",
						AccessKeySecret: "default_aliyun_access_key_secret",
						OutputFormat:    "json",
					},
					Profile{
						Name:            "aaa",
						Mode:            StsToken,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						StsToken:        "StsToken",
					},
					Profile{
						Name:            "bbb",
						Mode:            RamRoleArn,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						RamRoleArn:      "RamRoleArn",
						RoleSessionName: "RoleSessionName",
					},
					Profile{
						Name:            "ccc",
						Mode:            EcsRamRole,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						RamRoleName:     "RamRoleName",
					},
					Profile{
						Name:            "ddd",
						Mode:            RsaKeyPair,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						KeyPairName:     "KeyPairName",
					},
				},
			}, nil
		}
	}
	doConfigureList(w)
	assert.Equal(t, "Profile   | Credential             | Valid   | Region           | Language\n"+
		"--------- | ------------------     | ------- | ---------------- | --------\n"+
		"default * | AK:***_id              | Invalid |                  | \n"+
		"aaa       | StsToken:******        | Invalid |                  | \n"+
		"bbb       | RamRoleArn:******      | Invalid |                  | \n"+
		"ccc       | EcsRamRole:RamRoleName | Invalid |                  | \n"+
		"ddd       | RsaKeyPair:KeyPairName | Invalid |                  | \n", w.String())

	//testcase 2
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
			return Configuration{}, errors.New("error")
		}
	}
	w.Reset()
	doConfigureList(w)
	assert.Equal(t, "\x1b[1;31mERROR: load configure failed: error\n\x1b[0mProfile   | Credential         | Valid   | Region           | Language\n--------- | ------------------ | ------- | ---------------- | --------\n", w.String())
}
