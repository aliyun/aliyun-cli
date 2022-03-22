// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
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
package openapi

import (
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/stretchr/testify/assert"

	"bytes"
	"fmt"
	"testing"
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

func TestLibrary_PrintProducts(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")

	_, isexist := library.GetApi("aos", "v1.0", "describe")
	assert.False(t, isexist)

	products := library.GetProducts()
	assert.NotNil(t, products)

	product := meta.Product{
		Code:    "ecs",
		Version: "v1.0",
		Name:    map[string]string{"zh": "test"},
	}
	library.printProduct(product)

	library.builtinRepo.Products = []meta.Product{
		{
			Code: "ecs",
		},
	}
	library.PrintProducts()
}

func TestLibrary_PrintProductUsage(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	content := `{"products":[{"code":"ecs","api_style":"rpc","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err := library.PrintProductUsage("aos", true)
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)

	content = `{"products":[{"code":"ecs","api_style":"restful","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsage(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	content := `{"products":[{"code":"ecs","api_style":"rpc","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err := library.PrintApiUsage("aos", "DescribeRegions")
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)

	content = `{"products":[{"code":"ecs","api_style":"restful","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)
}

func Test_printParameters(t *testing.T) {
	w := new(bytes.Buffer)
	params := []meta.Parameter{
		{
			Hidden: true,
		},
		{
			Position: "Domain",
		},
		{
			Type:     "RepeatList",
			Required: true,
		},
		{
			Required: false,
		},
		{
			SubParameters: []meta.Parameter{
				{
					Name: "test",
				},
			},
		},
	}
	printParameters(w, params, "")
}

func getRepository(content string) *meta.Repository {
	reader := &reader_test{
		content: content,
	}
	repository := meta.LoadRepository(reader)
	return repository
}
