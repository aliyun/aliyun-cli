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
	assert.Equal(t, `"aos" is not a valid command or product. See `+"`aliyun help`"+`.`, err.Error())

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
	assert.Equal(t, `"aos" is not a valid command or product. See `+"`aliyun help`"+`.`, err.Error())

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
	assert.Contains(t, out, "(Recommended) Command Style:")
	assert.Contains(t, out, "aliyun demo create-thing --name foo")
	assert.Contains(t, out, "PascalCase Style:")
	assert.Contains(t, out, "aliyun demo CreateThing --Name foo")
}

func TestLibrary_PrintApiUsage_RestfulExampleLabel(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "demo", Version: "2026-01-01", ApiStyle: "restful", ApiNames: []string{"ValidateThing"}},
	})
	assert.Nil(t, err)

	w := new(bytes.Buffer)
	library := &Library{
		builtinRepo: repo,
		canonicalRepo: &byNameOnlyCanonicalRepo{apis: map[string]*canonicalmeta.API{
			"ValidateThing": {
				Name:         "ValidateThing",
				Protocol:     "HTTPS",
				Method:       "POST",
				PathPattern:  "/things/validate",
				KebabExample: "aliyun demo validate-thing --code foo",
				CamelExample: "aliyun demo POST /things/validate --body {}",
			},
		}},
		writer: w,
	}

	err = library.PrintApiUsage("demo", "ValidateThing")
	assert.Nil(t, err)
	out := w.String()
	assert.Contains(t, out, "(Recommended) Command Style:")
	assert.Contains(t, out, "RESTful Style:")
	assert.NotContains(t, out, "PascalCase Style:")
}

func TestLibrary_PrintProductUsage_RestfulListShowsSummary(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "demo", Version: "2026-01-01", ApiStyle: "restful", ApiNames: []string{"ValidateThing"}},
	})
	assert.Nil(t, err)

	w := new(bytes.Buffer)
	library := &Library{
		builtinRepo: repo,
		canonicalRepo: &byNameOnlyCanonicalRepo{apis: map[string]*canonicalmeta.API{
			"ValidateThing": {
				Name:          "ValidateThing",
				Method:        "POST",
				PathPattern:   "/things/validate",
				DescriptionZh: "验证一个对象",
				DescriptionEn: "Validates a thing",
			},
		}},
		writer: w,
	}

	err = library.PrintProductUsage("demo", true)
	assert.Nil(t, err)
	out := w.String()
	assert.Contains(t, out, "POST /things/validate")
	assert.True(t, strings.Contains(out, "Validates a thing") || strings.Contains(out, "验证一个对象"), out)
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

func TestLibrary_GetCanonicalApi_UsesProductDefaultVersion(t *testing.T) {
	repo, err := meta.MockLoadRepository([]meta.Product{
		{Code: "bailian", Version: "2023-12-29", ApiStyle: "restful", ApiNames: []string{"ListIndices"}},
	})
	assert.Nil(t, err)

	canonicalRepo := newFakeCanonicalRepo()
	canonicalRepo.AddAPI("bailian", "2023-06-01", &canonicalmeta.API{
		Name:        "ListIndices",
		Method:      "POST",
		PathPattern: "",
	})
	canonicalRepo.AddAPI("bailian", "2023-12-29", &canonicalmeta.API{
		Name:        "ListIndices",
		Method:      "GET",
		PathPattern: "/[WorkspaceId]/index/list_indices",
	})
	library := &Library{
		builtinRepo:   repo,
		canonicalRepo: canonicalRepo,
	}

	product, ok := library.GetProduct("bailian")
	assert.True(t, ok)
	api := library.GetCanonicalApi(product.Code, product.Version, "ListIndices")
	assert.NotNil(t, api)
	assert.Equal(t, "GET", api.Method)
	assert.Equal(t, "/[WorkspaceId]/index/list_indices", api.PathPattern)
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

func (r *byNameOnlyCanonicalRepo) GetVersionIndex(productCode, version string) (*canonicalmeta.VersionIndex, error) {
	return nil, fmt.Errorf("version index not found")
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
	t.Setenv("NO_COLOR", "1")

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
	// Body sub-fields are shown indented under --body with a "|-" marker, no flag prefix
	assert.Contains(t, out, "|- WorkspaceId")
	assert.Contains(t, out, "|- ReportName")
}

func TestPrintLegacyViews_DisplaysCanonicalLowercaseArrayType(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	api := &canonicalmeta.API{
		Parameters: []canonicalmeta.Parameter{
			{RawName: "Items", Type: "array", ParamStyle: "repeatList"},
		},
	}

	w := new(bytes.Buffer)
	printLegacyViews(w, api.LegacyTopLevelParameters(), "", nil)
	out := w.String()
	assert.Contains(t, out, "--Items.n")
	assert.Contains(t, out, "\tarray\t")
	assert.False(t, strings.Contains(out, "\tRepeatList\t"), out)
}

func getRepository() *meta.Repository {
	repository := meta.LoadRepository()
	return repository
}
