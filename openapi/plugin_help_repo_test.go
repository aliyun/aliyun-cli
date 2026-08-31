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
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineAPIToCanonicalMapsIdentityOperationAndParameters(t *testing.T) {
	engineMeta := &meta.API{
		Name:        "DescribePluginDemo",
		CmdName:     "describe-plugin-demo",
		CmdFullName: "ecs describe-plugin-demo",
		ProductCode: "ecs",
		Version:     "2014-05-26",
		Method:      "POST",
		URL:         "/demo",
		Style:       meta.StyleROA,
		Protocol:    "HTTPS",
		Deprecated:  true,
		Title:       meta.Description{ZH: "中文标题", EN: "english title"},
		Description: meta.Description{ZH: "中文描述", EN: "english description"},
		Examples:    []string{"aliyun ecs describe-plugin-demo --region-id cn-hangzhou"},
		Responses:   []byte(`{"200":{"description":"ok"}}`),
		Components:  []byte(`{"schemas":{"Result":{"type":"object"}}}`),
		Parameters: []meta.Parameter{{
			Name:        "region_id",
			RawName:     "RegionId",
			Type:        meta.TypeString,
			Position:    meta.PosQuery,
			Required:    true,
			Enum:        []string{"a", "b"},
			Options:     []string{"--region-id"},
			Description: meta.Description{ZH: "地域", EN: "region"},
			Example:     "cn-hangzhou",
			ItemType:    &meta.Parameter{Type: meta.TypeInteger, Maximum: "100"},
		}},
	}

	converted := engineAPIToCanonical(engineMeta)

	assert.Equal(t, "DescribePluginDemo", converted.Name)
	assert.Equal(t, "describe-plugin-demo", converted.CmdName)
	assert.Equal(t, "ecs describe-plugin-demo", converted.CmdFullName)
	assert.Equal(t, "ecs", converted.ProductCode)
	assert.True(t, converted.Deprecated)
	assert.Equal(t, "中文标题", converted.TitleZh)
	assert.Equal(t, "english title", converted.TitleEn)
	assert.Equal(t, "中文描述", converted.DescriptionZh)
	assert.Equal(t, "english description", converted.DescriptionEn)
	assert.Equal(t, "aliyun ecs describe-plugin-demo --region-id cn-hangzhou", converted.KebabExample)
	assert.JSONEq(t, `{"200":{"description":"ok"}}`, string(converted.Responses))
	assert.JSONEq(t, `{"schemas":{"Result":{"type":"object"}}}`, string(converted.Components))
	require.NotNil(t, converted.Operation)
	assert.Equal(t, "ROA", converted.Operation.APIStyle)
	assert.Equal(t, "2014-05-26", converted.Operation.APIVersion)
	assert.Equal(t, "POST", converted.Operation.Method)
	assert.Equal(t, "HTTPS", converted.Operation.Protocol)
	assert.Equal(t, "/demo", converted.Operation.URL)

	require.Len(t, converted.Parameters, 1)
	parameter := converted.Parameters[0]
	assert.Equal(t, "region_id", parameter.Name)
	assert.Equal(t, "RegionId", parameter.RawName)
	assert.Equal(t, "string", parameter.Type)
	assert.Equal(t, "query", parameter.Location)
	assert.True(t, parameter.Required)
	assert.Equal(t, []string{"--region-id"}, parameter.Options)
	assert.Equal(t, []string{"a", "b"}, parameter.Enum)
	assert.Equal(t, "地域", parameter.HelpZh)
	assert.Equal(t, "region", parameter.HelpEn)
	assert.Equal(t, "cn-hangzhou", parameter.Example)
}

func TestPluginAwareHelpRepositoryRoutesByOwnership(t *testing.T) {
	baseline := &stubMachineHelpRepository{apis: map[string]*canonicalmeta.API{
		"baseline": {Name: "BaselineAPI"},
	}}
	repo := newPluginAwareHelpRepository(baseline)

	originalOwns := metaPluginOwnsProduct
	originalEntries := metaPluginCatalogEntries
	t.Cleanup(func() {
		metaPluginOwnsProduct = originalOwns
		metaPluginCatalogEntries = originalEntries
	})
	metaPluginOwnsProduct = func(product string) bool { return product == "plugin" }
	metaPluginCatalogEntries = func() []canonicalmeta.ProductEntry {
		return []canonicalmeta.ProductEntry{{Code: "plugin", Distribution: "meta", PluginDefaultVersion: "2026-01-01"}}
	}

	catalog, err := repo.GetProducts()
	require.NoError(t, err)
	codes := make([]string, 0, len(catalog.Products))
	for _, entry := range catalog.Products {
		codes = append(codes, entry.Code)
	}
	assert.ElementsMatch(t, []string{"baseline", "plugin"}, codes)

	// Non-plugin products stay on the baseline repository.
	api, err := repo.GetAPI("baseline", "2026-01-01", "BaselineAPI")
	require.NoError(t, err)
	assert.Equal(t, "BaselineAPI", api.Name)

	// Plugin-owned products go through the engine loader; an unknown product
	// surfaces the engine error instead of the baseline fallback.
	_, err = repo.GetAPI("plugin", "2026-01-01", "Anything")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "stub")
}

type stubMachineHelpRepository struct {
	apis map[string]*canonicalmeta.API
}

func (s *stubMachineHelpRepository) GetProducts() (*canonicalmeta.ProductsIndex, error) {
	return &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{{Code: "baseline"}}}, nil
}

func (s *stubMachineHelpRepository) GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	return &canonicalmeta.VersionIndex{Version: version}, nil
}

func (s *stubMachineHelpRepository) GetAPI(product, version, apiName string) (*canonicalmeta.API, error) {
	api, ok := s.apis[product]
	if !ok {
		return nil, fmt.Errorf("stub baseline has no product %q", product)
	}
	return api, nil
}
