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

func TestDoConfigureDelete(t *testing.T) {
	w := new(bytes.Buffer)
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()

	//testcase 1
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "bbb", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config Configuration) error) func(config Configuration) error {
		return func(config Configuration) error {
			return nil
		}
	}
	doConfigureDelete(w, "bbb")
	assert.Empty(t, w.String())

	//testcase 2
	w.Reset()
	doConfigureDelete(w, "aaa")
	assert.Equal(t, "\x1b[1;31mError: configuration profile `aaa` not found\n\x1b[0m", w.String())

	//testcase 3
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{}, errors.New("error")
		}
	}

	w.Reset()
	doConfigureDelete(w, "bbb")
	assert.Equal(t, "\x1b[1;31mERROR: load configure failed: error\n\x1b[0m\x1b[1;31mError: configuration profile `bbb` not found\n\x1b[0m", w.String())

	//testcase 4
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "bbb", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config Configuration) error) func(config Configuration) error {
		return func(config Configuration) error {
			return errors.New("save error")
		}
	}
	w.Reset()
	doConfigureDelete(w, "bbb")
	assert.Equal(t, "\x1b[1;31mError: save configuration failed save error\n\x1b[0m", w.String())
}
