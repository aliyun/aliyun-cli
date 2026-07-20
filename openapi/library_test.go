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
package openapi

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"

	"bytes"
	"testing"
)

func TestLibrary_PrintProducts(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")

	_, isexist := library.GetApi("aos", "v1.0", "describe")
	assert.False(t, isexist)

	products := library.GetProducts()
	assert.NotNil(t, products)

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
	library.builtinRepo = getRepository()
	err := library.PrintProductUsage("aos", true)
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)

	library.builtinRepo = getRepository()
	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsage(t *testing.T) {
	w := new(bytes.Buffer)
	library := NewLibrary(w, "en")
	library.builtinRepo = getRepository()
	err := library.PrintApiUsage("aos", "DescribeRegions")
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)

	library.builtinRepo = getRepository()
	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsage_PrintsCanonicalExamples(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "demo", Version: "2026-01-01", ApiStyle: "rpc", ApiNames: []string{"CreateThing"}},
	})
	assert.Nil(t, err)

	w := new(bytes.Buffer)
	library := &Library{
		builtinRepo: repo,
		canonicalRepo: &byNameOnlyCanonicalRepo{apis: map[string]*canonicalmeta.API{
			"CreateThing": {
				Name:         "CreateThing",
				Protocol:     "HTTPS",
				Method:       "POST",
				KebabExample: "aliyun demo create-thing --name foo",
				CamelExample: "aliyun demo CreateThing --Name foo",
			},
		}},
		writer: w,
	}

	err = library.PrintApiUsage("demo", "CreateThing")
	assert.Nil(t, err)
	out := w.String()
	assert.Contains(t, out, "Example:")
	assert.Contains(t, out, "(Recommended) New CLI:")
	assert.Contains(t, out, "aliyun demo create-thing --name foo")
	assert.Contains(t, out, "Legacy CLI:")
	assert.Contains(t, out, "aliyun demo CreateThing --Name foo")
}

func TestLibrary_GetApi_NoLegacyRuntimeFallback(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "ecs", Version: "2014-05-26", ApiNames: []string{"DescribeRegions"}},
	})
	assert.Nil(t, err)
	library := &Library{builtinRepo: repo}

	_, ok := library.GetApi("ecs", "2014-05-26", "DescribeRegions")
	assert.False(t, ok)
	assert.Nil(t, library.GetCanonicalApi("ecs", "2014-05-26", "DescribeRegions"))
}

type byNameOnlyCanonicalRepo struct {
	apis map[string]*canonicalmeta.API
}

func (r *byNameOnlyCanonicalRepo) GetAPI(productCode, version, apiName string) (*canonicalmeta.API, error) {
	if api, ok := r.apis[apiName]; ok {
		return api, nil
	}
	return nil, fmt.Errorf("api not found")
}

func (r *byNameOnlyCanonicalRepo) GetAPIByPath(productCode, version, method, path string, apiNames []string) (*canonicalmeta.API, error) {
	for _, apiName := range apiNames {
		api, ok := r.apis[apiName]
		if ok && api.MatchLegacyPath(method, path) {
			return api, nil
		}
	}
	return nil, fmt.Errorf("api not found")
}

func TestLibrary_GetApiByPath_UsesProductApiList(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "demo", Version: "2026-01-01", ApiStyle: "restful", ApiNames: []string{"ListedMatch"}},
	})
	assert.Nil(t, err)

	library := &Library{
		builtinRepo: repo,
		canonicalRepo: &byNameOnlyCanonicalRepo{apis: map[string]*canonicalmeta.API{
			"ListedMatch": {Name: "ListedMatch", Method: "GET", PathPattern: "/items/[id]"},
		}},
	}

	api, ok := library.GetApiByPath("demo", "2026-01-01", "GET", "/items/123")
	assert.True(t, ok)
	assert.Equal(t, "ListedMatch", api.Name)
}

func TestLibrary_PrintApiUsage_UsesV1BodyParameters(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "demo", Version: "2026-01-01", ApiStyle: "rpc", ApiNames: []string{"CreateReport"}},
	})
	assert.Nil(t, err)

	w := new(bytes.Buffer)
	library := &Library{
		builtinRepo:   repo,
		canonicalRepo: canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata")),
		writer:        w,
	}

	err = library.PrintApiUsage("demo", "CreateReport")
	assert.Nil(t, err)
	out := w.String()
	assert.Contains(t, out, "--body")
	assert.NotContains(t, out, "--ReportName")
	assert.NotContains(t, out, "--WorkspaceId")
}

func TestPrintLegacyViews_DisplaysCanonicalLowercaseArrayType(t *testing.T) {
	api := &canonicalmeta.API{
		Parameters: []canonicalmeta.Parameter{
			{RawName: "Items", Type: "array", ParamStyle: "repeatList"},
		},
	}

	w := new(bytes.Buffer)
	printLegacyViews(w, api.LegacyTopLevelParameters(), "")
	out := w.String()
	assert.Contains(t, out, "--Items.n")
	assert.Contains(t, out, "\tarray\t")
	assert.False(t, strings.Contains(out, "\tRepeatList\t"), out)
}

func getRepository() *meta.Repository {
	repository := meta.LoadRepository()
	return repository
}
