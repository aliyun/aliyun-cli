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
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoConfigureSet_CliSignVersion(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())

	origLoad := hookLoadOrCreateConfiguration
	origSave := hookSaveConfigurationWithContext
	defer func() {
		hookLoadOrCreateConfiguration = origLoad
		hookSaveConfigurationWithContext = origSave
	}()

	var saved Profile
	hookSaveConfigurationWithContext = func(fn func(ctx *cli.Context, config *Configuration) error) func(ctx *cli.Context, config *Configuration) error {
		return func(ctx *cli.Context, config *Configuration) error {
			p, _ := config.GetProfile(config.CurrentProfile)
			saved = p
			return nil
		}
	}
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:            "default",
						RegionId:        "cn-hangzhou",
						Mode:            AK,
						AccessKeyId:     "ak",
						AccessKeySecret: "sk",
					},
				},
			}, nil
		}
	}

	f := CliSignVersionFlag(ctx.Flags())
	require.NotNil(t, f)
	f.SetAssigned(true)
	f.SetValue("v4")
	require.NoError(t, doConfigureSet(ctx))
	assert.Equal(t, SignatureVersionV4, saved.SignatureVersion)

	f.SetValue("V4")
	require.NoError(t, doConfigureSet(ctx))
	assert.Equal(t, SignatureVersionV4, saved.SignatureVersion)

	f.SetValue("v3")
	err := doConfigureSet(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported signature_version")

	// Flag not assigned: keep existing profile.signature_version.
	f.SetAssigned(false)
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:             "default",
						RegionId:         "cn-hangzhou",
						Mode:             AK,
						AccessKeyId:      "ak",
						AccessKeySecret:  "sk",
						SignatureVersion: SignatureVersionV4,
					},
				},
			}, nil
		}
	}
	require.NoError(t, doConfigureSet(ctx))
	assert.Equal(t, SignatureVersionV4, saved.SignatureVersion)
}
