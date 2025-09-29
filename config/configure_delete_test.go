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

func TestDoConfigureDelete(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())

	originhook := hookLoadConfigurationWithContext
	originhookSave := hookSaveConfigurationWithContext
	defer func() {
		hookLoadConfigurationWithContext = originhook
		hookSaveConfigurationWithContext = originhookSave
	}()

	//testcase 1
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "bbb", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfigurationWithContext = func(fn func(ctx *cli.Context, config *Configuration) error) func(ctx *cli.Context, config *Configuration) error {
		return func(ctx *cli.Context, config *Configuration) error {
			return nil
		}
	}

	err := doConfigureDelete(ctx, "bbb")
	assert.Empty(t, stdout.String())
	assert.Nil(t, err)

	//testcase 2
	stdout.Reset()
	stderr.Reset()
	err = doConfigureDelete(ctx, "aaa")
	assert.NotNil(t, err)
	assert.Equal(t, "error: configuration profile `aaa` not found", err.Error())

	//testcase 3
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}

	stdout.Reset()
	stderr.Reset()
	err = doConfigureDelete(ctx, "bbb")
	assert.NotNil(t, err)
	assert.Equal(t, "ERROR: load configure failed: error", err.Error())

	//testcase 4
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "bbb", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfigurationWithContext = func(fn func(ctx *cli.Context, config *Configuration) error) func(ctx *cli.Context, config *Configuration) error {
		return func(ctx *cli.Context, config *Configuration) error {
			return errors.New("save error")
		}
	}
	stdout.Reset()
	stderr.Reset()
	err = doConfigureDelete(ctx, "bbb")
	assert.NotNil(t, err)
	assert.Equal(t, "error: save configuration failed save error", err.Error())
}
