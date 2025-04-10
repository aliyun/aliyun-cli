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
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"

	"github.com/stretchr/testify/assert"
)

func TestNewConfiguration(t *testing.T) {
	excf := &Configuration{
		CurrentProfile: DefaultConfigProfileName,
		Profiles: []Profile{
			NewProfile(DefaultConfigProfileName),
		},
	}
	cf := NewConfiguration()
	assert.Equal(t, excf, cf)
}

func TestCFNewProfile(t *testing.T) {
	cf := Configuration{
		CurrentProfile: "",
	}
	assert.Len(t, cf.Profiles, 0)
	p := cf.NewProfile("default")
	assert.Len(t, cf.Profiles, 1)
	exp := Profile{
		Name:         "default",
		Mode:         "",
		OutputFormat: "json",
		Language:     "en",
	}
	assert.Equal(t, exp, p)
}

func TestConfiguration(t *testing.T) {
	cf := NewConfiguration()
	//GetProfile
	p, ok := cf.GetProfile("hh")
	assert.False(t, ok)
	assert.Equal(t, Profile{Name: "hh"}, p)
	p, _ = cf.GetProfile("default")
	assert.Equal(t, Profile{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}, p)

	//PutProfile
	assert.Len(t, cf.Profiles, 1)
	cf.PutProfile(Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"}, cf.Profiles[1])
	cf.PutProfile(Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, cf.Profiles[1])

	//GetCurrentProfile
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	os.Setenv("ALIBABACLOUD_PROFILE", "test")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	cf.PutProfile(Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"})
	cf.CurrentProfile = "test2"
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	cf.CurrentProfile = "default"
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABA_CLOUD_PROFILE", "test2")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABACLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALICLOUD_PROFILE", "test")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test2", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALIBABA_CLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, p)
	os.Setenv("ALICLOUD_PROFILE", "")
	p = cf.GetCurrentProfile(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}, p)
}

func TestLoadProfile(t *testing.T) {
	originhook := hookLoadConfiguration
	w := new(bytes.Buffer)
	defer func() {
		hookLoadConfiguration = originhook
	}()
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	//testcase 1
	p, err := LoadProfile(GetConfigPath()+"/"+configFile, "")
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 2
	_, err = LoadProfile(GetConfigPath()+"/"+configFile, "hello")
	assert.EqualError(t, err, "unknown profile hello, run configure to check")

	//LoadCurrentProfile testcase
	w.Reset()
	p, err = LoadCurrentProfile()
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 3
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}
	w.Reset()
	p, err = LoadProfile(GetConfigPath()+"/"+configFile, "")
	assert.Empty(t, p)
	assert.EqualError(t, err, "init config failed error")
}

func TestHomePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, os.Getenv("USERPROFILE"), GetHomePath())
	} else {
		assert.Equal(t, os.Getenv("HOME"), GetHomePath())
	}
}

func TestGetConfigPath(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	assert.Equal(t, "./.aliyun", GetConfigPath())
}

func TestNewConfigFromBytes(t *testing.T) {
	bytesConf := `{
		"current": "",
		"profiles": [
			{
				"name": "default",
				"mode": "AK",
				"access_key_id": "access_key_id",
				"access_key_secret": "access_key_secret",
				"sts_token": "",
				"ram_role_name": "",
				"ram_role_arn": "",
				"ram_session_name": "",
				"private_key": "",
				"key_pair_name": "",
				"expired_seconds": 0,
				"verified": "",
				"region_id": "cn-hangzhou",
				"output_format": "json",
				"language": "en",
				"site": "",
				"retry_timeout": 0,
				"retry_count": 0
			}
		],
		"meta_path": ""
	}`

	conf, err := NewConfigFromBytes([]byte(bytesConf))
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}, conf)
}

func TestSaveConfiguration(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	conf := &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	bytes, err := json.MarshalIndent(conf, "", "\t")
	assert.Nil(t, err)
	err = SaveConfiguration(conf)
	assert.Nil(t, err)
	file, err := os.Open(GetConfigPath() + "/" + configFile)
	assert.Nil(t, err)
	buf := make([]byte, 1024)
	n, _ := file.Read(buf)
	file.Close()
	assert.Equal(t, string(bytes), string(buf[:n]))
}

func TestLoadConfiguration(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyun")
		hookGetHomePath = orighookGetHomePath
	}()
	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}
	w := new(bytes.Buffer)

	//testcase 1
	cf, err := LoadConfiguration(GetConfigPath() + "/" + configFile)
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: "", OutputFormat: "json", Language: "en"}}}, cf)
	conf := &Configuration{Profiles: []Profile{{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	err = SaveConfiguration(conf)
	assert.Nil(t, err)

	//testcase 2
	w.Reset()
	cf, err = LoadConfiguration(GetConfigPath() + "/" + configFile)
	assert.Equal(t, &Configuration{CurrentProfile: "", Profiles: []Profile{{Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en"}}}, cf)
	assert.Nil(t, err)

}

func TestLoadProfileWithContext(t *testing.T) {
	originhook := hookLoadConfiguration
	defer func() {
		hookLoadConfiguration = originhook
	}()
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())

	//testcase 1
	_, err := LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")

	//testcase 2
	ctx.Flags().Get("profile").SetAssigned(true)
	_, err = LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")
}

func TestLoadProfileWithContextWhenIGNORE_PROFILE(t *testing.T) {
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "TRUE")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.SetInConfigureMode(true)
	AddFlags(ctx.Flags())
	ctx.Flags().Get("access-key-id").SetAssigned(true)
	ctx.Flags().Get("access-key-id").SetValue("test-ak-id")
	ctx.Flags().Get("access-key-secret").SetAssigned(true)
	ctx.Flags().Get("access-key-secret").SetValue("test-ak-secret")
	p, err := LoadProfileWithContext(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "default", p.Name)
	assert.Equal(t, "cn-hangzhou", p.RegionId)
	assert.Equal(t, AK, p.Mode)
	// reset
	os.Setenv("ALIBABA_CLOUD_IGNORE_PROFILE", "")
}

func TestGetHomePath(t *testing.T) {
	home := GetHomePath()
	assert.NotEqual(t, "", home)
}

func TestGetProfileName(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	// default case: no flag, no env
	name := getProfileName(ctx)
	assert.Equal(t, name, "")

	// case 1: with flag
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("FromProfileFlag")
	name = getProfileName(ctx)
	assert.Equal(t, name, "FromProfileFlag")

	// case 2: with env
	ctx.Flags().Get("profile").SetAssigned(false)
	ctx.Flags().Get("profile").SetValue("")
	name = getProfileName(ctx)
	assert.Equal(t, name, "") // reset flag
	os.Setenv("ALIBABA_CLOUD_PROFILE", "profileName")
	name = getProfileName(ctx)
	assert.Equal(t, name, "profileName")
	os.Setenv("ALIBABA_CLOUD_PROFILE", "") // reset env
}

func TestGetConfigurePath(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	// default case: no flag, no env
	p := getConfigurePath(ctx)
	assert.Contains(t, p, ".aliyun/config.json")

	// case 1: with flag
	ctx.Flags().Get("config-path").SetAssigned(true)
	ctx.Flags().Get("config-path").SetValue("/path/to/config.json")
	p = getConfigurePath(ctx)
	assert.Equal(t, p, "/path/to/config.json")
}
