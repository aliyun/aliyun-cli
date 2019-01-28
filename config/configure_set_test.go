/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/cli"
)

func TestDoConfigureSet(t *testing.T) {
	fs := cli.NewFlagSet()
	AddFlags(fs)
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config Configuration) error) func(config Configuration) error {
		return func(config Configuration) error {
			return nil
		}
	}
	w := new(bytes.Buffer)
	doConfigureSet(w, fs)
	assert.Equal(t, "\x1b[1;31mfail to set configuration: region can't be empty\x1b[0m", w.String())
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())
}
