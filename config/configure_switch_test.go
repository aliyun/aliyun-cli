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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestDoConfigureSwitch(t *testing.T) {
	stdout := new(bytes.Buffer)
	fs := cli.NewFlagSet()
	AddFlags(fs)
	originhook := hookLoadConfiguration
	originSaveHook := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originSaveHook
	}()

	// test case 1: load configuration failed
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("mock error")
		}
	}
	stdout.Reset()
	err := doConfigureSwitch(stdout, fs)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "load configuration failed: mock error")

	//testcase 2: no --profile flag
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
				},
			}, nil
		}
	}
	err = doConfigureSwitch(stdout, fs)
	assert.EqualError(t, err, "the --profile <profileName> is required")

	// test case 3: inexist profile
	fs.Get(ProfileFlagName).SetAssigned(true)
	fs.Get(ProfileFlagName).SetValue("inexist")
	err = doConfigureSwitch(stdout, fs)
	assert.EqualError(t, err, "the profile `inexist` is inexist")

	// test case 4: save configuration failed
	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return errors.New("mock save error")
		}
	}
	fs.Get(ProfileFlagName).SetValue("aaa")
	err = doConfigureSwitch(stdout, fs)
	assert.EqualError(t, err, "save configuration failed: mock save error")

	// test case 5: it ok
	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}
	fs.Get(ProfileFlagName).SetValue("aaa")
	err = doConfigureSwitch(stdout, fs)
	assert.Nil(t, err)
	assert.Equal(t, "The default profile is `aaa` now.\n", stdout.String())
}
