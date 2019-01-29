/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/cli"

	"github.com/stretchr/testify/assert"
)

func TestNewConfiguration(t *testing.T) {
	excf := Configuration{
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
		Mode:         AK,
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
	p, ok = cf.GetProfile("default")
	assert.Equal(t, Profile{Name: "default", Mode: AK, OutputFormat: "json", Language: "en"}, p)

	//PutProfile
	assert.Len(t, cf.Profiles, 1)
	cf.PutProfile(Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: AK, OutputFormat: "json", Language: "en"}, cf.Profiles[1])
	cf.PutProfile(Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"})
	assert.Len(t, cf.Profiles, 2)
	assert.Equal(t, Profile{Name: "test", Mode: StsToken, OutputFormat: "json", Language: "en"}, cf.Profiles[1])

	//GetCurrentProfile

}

func TestLoadProfile(t *testing.T) {
	originhook := hookLoadConfiguration
	w := new(bytes.Buffer)
	defer func() {
		hookLoadConfiguration = originhook
	}()
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	//testcase 1
	p, err := LoadProfile(w, "")
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 2
	_, err = LoadProfile(w, "hello")
	assert.EqualError(t, err, "unknown profile hello, run configure to check")

	//LoadCurrentProfile testcase
	w.Reset()
	p, err = LoadCurrentProfile(w)
	assert.Nil(t, err)
	p.parent = nil
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, p)

	//testcase 3
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{}, errors.New("error")
		}
	}
	w.Reset()
	p, err = LoadProfile(w, "")
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
	assert.Equal(t, Configuration{Profiles: []Profile{Profile{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}, conf)
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
	conf := Configuration{Profiles: []Profile{Profile{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	bytes, err := json.Marshal(conf)
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
	cf, err := LoadConfiguration(w)
	assert.Nil(t, err)
	assert.Equal(t, Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: "AK", OutputFormat: "json", Language: "en"}}}, cf)
	conf := Configuration{Profiles: []Profile{Profile{Language: "en", Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}}}
	err = SaveConfiguration(conf)
	assert.Nil(t, err)

	//testcase 2
	w.Reset()
	cf, err = LoadConfiguration(w)
	assert.Equal(t, Configuration{CurrentProfile: "", Profiles: []Profile{Profile{Name: "default", Mode: "AK", AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json", Language: "en"}}}, cf)
	assert.Nil(t, err)

}

func TestLoadProfileWithContext(t *testing.T) {
	originhook := hookLoadConfiguration
	defer func() {
		hookLoadConfiguration = originhook
	}()
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	AddFlags(ctx.Flags())

	//testcase 1
	_, err := LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")

	//testcase 2
	ctx.Flags().Get("profile").SetAssigned(true)
	_, err = LoadProfileWithContext(ctx)
	assert.EqualError(t, err, "region can't be empty")

}
