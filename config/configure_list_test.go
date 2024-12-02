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
	"errors"
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
						Mode:            StsToken,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						StsToken:        "StsToken",
					},
					{
						Name:            "bbb",
						Mode:            RamRoleArn,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						RamRoleArn:      "RamRoleArn",
						RoleSessionName: "RoleSessionName",
					},
					{
						Name:            "bbbe",
						Mode:            RamRoleArn,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						RamRoleArn:      "RamRoleArn",
						RoleSessionName: "RoleSessionName",
						ExternalId:      "ExternalId",
					},
					{
						Name:            "ccc",
						Mode:            EcsRamRole,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						RamRoleName:     "RamRoleName",
					},
					{
						Name:            "ddd",
						Mode:            RsaKeyPair,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
						KeyPairName:     "KeyPairName",
					},
					{
						Name:                 "eee",
						Mode:                 CloudSSO,
						AccessKeyId:          "sdf",
						CloudSSOAccountId:    "a",
						CloudSSOAccessConfig: "b",
						CloudSSOSignInUrl:    "c",
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
		"bbbe      | RamRoleArn:******:lId  | Invalid |                  | \n"+
		"ccc       | EcsRamRole:RamRoleName | Invalid |                  | \n"+
		"ddd       | RsaKeyPair:KeyPairName | Invalid |                  | \n"+
		"eee       | CloudSSO:a@b           | Invalid |                  | \n", w.String())

	//testcase 2
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}
	w.Reset()
	doConfigureList(w)
	assert.Equal(t, "\x1b[1;31mERROR: load configure failed: error\n\x1b[0mProfile   | Credential         | Valid   | Region           | Language\n--------- | ------------------ | ------- | ---------------- | --------\n", w.String())
}
