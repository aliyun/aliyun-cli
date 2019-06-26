// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
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
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
			return Configuration{}, errors.New("error")
		}
	}

	w.Reset()
	doConfigureDelete(w, "bbb")
	assert.Equal(t, "\x1b[1;31mERROR: load configure failed: error\n\x1b[0m\x1b[1;31mError: configuration profile `bbb` not found\n\x1b[0m", w.String())

	//testcase 4
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
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
