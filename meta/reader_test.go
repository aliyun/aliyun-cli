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
package meta

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type reader_test struct {
	content string
}

func (r *reader_test) ReadFrom(path string) ([]byte, error) {
	if path == "" || r.content == "" {
		return nil, fmt.Errorf("Please insert a valid path.")
	}
	return []byte(r.content), nil
}

func (r *reader_test) setcontent(contenth string) {
	r.content = contenth
}

func TestReadYamlFrom(t *testing.T) {
	r := &reader_test{}
	path := ""
	err := ReadYamlFrom(r, path, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "read yaml from  failed <nil>", err.Error())

	api := &Api{}
	path = `{"name":"api""protocol":"http"}`
	r.content = path
	err = ReadYamlFrom(r, path, api)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal yaml {\"name\":\"api\"\"protocol\":\"http\"} failed yaml: did not find expected ',' or '}'", err.Error())

	r.content = `{"name":"api","protocol":"http"}`
	err = ReadYamlFrom(r, path, api)
	assert.Nil(t, err)
}

func TestReadJsonFrom(t *testing.T) {
	r := &reader_test{}
	path := ""
	err := ReadJsonFrom(r, path, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "read json from  failed <nil>", err.Error())

	api := &Api{}
	path = `{"name":"api""protocol":"http"}`
	r.content = path
	err = ReadJsonFrom(r, path, api)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal json {\"name\":\"api\"\"protocol\":\"http\"} failed invalid character '\"' after object key:value pair", err.Error())

	r.content = `{"name":"api","protocol":"http"}`
	err = ReadJsonFrom(r, path, api)
	assert.Nil(t, err)
}
