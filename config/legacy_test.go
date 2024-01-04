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
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrateCredentials(t *testing.T) {
	conf, err := MigrateCredentials("http://nicai")
	assert.Empty(t, conf)
	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, " parse failed: open http://nicai: The filename, directory name, or volume label syntax is incorrect.")
	} else {
		assert.EqualError(t, err, " parse failed: open http://nicai: no such file or directory")
	}

	test, err := os.Create("test.ini")
	assert.Nil(t, err)
	_, err = test.WriteString(`
	[DEFAULT]
	aliyun_access_key_id = DEFAULT_aliyun_access_key_id
	aliyun_access_key_secret = DEFAULT_aliyun_access_key_secret
	[default]
	aliyun_access_key_id = default_aliyun_access_key_id
	aliyun_access_key_secret = default_aliyun_access_key_secret
	[profile aaa]
	aliyun_access_key_id = sdf
	aliyun_access_key_secret = ddf
	`)
	assert.Nil(t, err)
	test.Close()

	conf, err = MigrateCredentials("test.ini")
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{
		CurrentProfile: "default",
		Profiles: []Profile{
			{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
			{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, conf)

	defer func() {
		if _, err := os.Stat("test.ini"); err == nil {
			os.Remove("test.ini")
		}
	}()
}

func TestMigrateConfigure(t *testing.T) {
	conf := &Configuration{CurrentProfile: "default", Profiles: []Profile{
		{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
		{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}
	err := MigrateConfigure("http://nici", conf)
	if runtime.GOOS == "windows" {
		assert.Equal(t, "parse failed: open http://nici: The filename, directory name, or volume label syntax is incorrect.", err.Error())
	} else {
		assert.Equal(t, "parse failed: open http://nici: no such file or directory", err.Error())
	}

	test, err := os.Create("testconf.ini")
	assert.Nil(t, err)
	test.WriteString(`
	[DEFAULT]
	aliyun_access_key_id = DEFAULT_aliyun_access_key_id
	aliyun_access_key_secret = DEFAULT_aliyun_access_key_secret
	region = cn-hangzhou
	[default]
	aliyun_access_key_id = default_aliyun_access_key_id
	aliyun_access_key_secret = default_aliyun_access_key_secret
	region = cn-hangzhou
	[profile aaa]
	aliyun_access_key_id = sdf
	aliyun_access_key_secret = ddf
	`)

	test.Close()
	err = MigrateConfigure("testconf.ini", conf)
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{
		CurrentProfile: "default",
		Profiles: []Profile{
			{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"},
			{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, conf)

	defer func() {
		if _, err := os.Stat("testconf.ini"); err == nil {
			os.Remove("testconf.ini")
		}
	}()
}

func TestMigrateLegacyConfiguration(t *testing.T) {
	orighookGetHomePath := hookGetHomePath
	defer func() {
		os.RemoveAll("./.aliyuncli")
		hookGetHomePath = orighookGetHomePath
	}()

	hookGetHomePath = func(fn func() string) func() string {
		return func() string {
			return "."
		}
	}

	err := os.Mkdir("./.aliyuncli", os.ModePerm)
	assert.Nil(t, err)

	test, err := os.Create("./.aliyuncli/credentials")
	assert.Nil(t, err)
	_, err = test.WriteString(`
	[DEFAULT]
	aliyun_access_key_id = DEFAULT_aliyun_access_key_id
	aliyun_access_key_secret = DEFAULT_aliyun_access_key_secret
	[default]
	aliyun_access_key_id = default_aliyun_access_key_id
	aliyun_access_key_secret = default_aliyun_access_key_secret
	[profile aaa]
	aliyun_access_key_id = sdf
	aliyun_access_key_secret = ddf
	`)
	assert.Nil(t, err)
	test.Close()
	conf, err := MigrateLegacyConfiguration()
	assert.Nil(t, err)
	assert.NotNil(t, conf)
}
