/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var lcfgText = []byte(`
[default]
aliyun_access_key_id = **************************
aliyun_access_key_secret = *************************
[profile aaa]
aliyun_access_key_secret = ddf
aliyun_access_key_id = sdf
`)

func TestMigrateCredentials(t *testing.T) {
	conf, err := MigrateCredentials("http://nicai")
	assert.Empty(t, conf)
	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, " parse failed: open http://nicai: The filename, directory name, or volume label syntax is incorrect.\n")
	} else {
		assert.EqualError(t, err, " parse failed: open http://nicai: no such file or directory\n")

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
	assert.Equal(t, Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, conf)

	defer func() {
		if _, err := os.Stat("test.ini"); err == nil {
			os.Remove("test.ini")
		}

	}()
}

func TestMigrateConfigure(t *testing.T) {
	w := new(bytes.Buffer)
	conf := &Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}
	err := MigrateConfigure(w, "http://nici", conf)
	if runtime.GOOS == "windows" {
		assert.Equal(t, "\x1b[1;31m parse failed: open http://nici: The filename, directory name, or volume label syntax is incorrect.\n\x1b[0m", w.String())

	} else {
		assert.Equal(t, "\x1b[1;31m parse failed: open http://nici: no such file or directory\n\x1b[0m", w.String())

	}

	test, err := os.Create("testconf.ini")
	assert.Nil(t, err)
	_, err = test.WriteString(`
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
	w.Reset()
	err = MigrateConfigure(w, "testconf.ini", conf)
	assert.Nil(t, err)
	assert.Equal(t, &Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, conf)

	defer func() {
		if _, err := os.Stat("testconf.ini"); err == nil {
			os.Remove("testconf.ini")
		}
	}()
}
