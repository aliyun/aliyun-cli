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
	"errors"
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

func TestPluginAwareHelpRepositoryCatalogMergeAndErrors(t *testing.T) {
	originalEntries := metaPluginCatalogEntries
	t.Cleanup(func() { metaPluginCatalogEntries = originalEntries })
	metaPluginCatalogEntries = func() []canonicalmeta.ProductEntry {
		return []canonicalmeta.ProductEntry{
			{Code: "ECS", Distribution: "meta", PluginDefaultVersion: "v2"},
			{Code: "new", Distribution: "meta", PluginDefaultVersion: "v1"},
		}
	}

	t.Run("nil baseline is preserved", func(t *testing.T) {
		assert.Nil(t, newPluginAwareHelpRepository(nil))
	})

	t.Run("catalog entries replace case-insensitively and append", func(t *testing.T) {
		baseline := &stubMachineHelpRepository{products: &canonicalmeta.ProductsIndex{
			Products: []canonicalmeta.ProductEntry{{Code: "ecs", Distribution: "go"}},
		}}
		repo := newPluginAwareHelpRepository(baseline)
		catalog, err := repo.GetProducts()
		require.NoError(t, err)
		require.Len(t, catalog.Products, 2)
		assert.Equal(t, "ECS", catalog.Products[0].Code)
		assert.Equal(t, "meta", catalog.Products[0].Distribution)
		assert.Equal(t, "new", catalog.Products[1].Code)
	})

	t.Run("nil catalog is initialized", func(t *testing.T) {
		repo := newPluginAwareHelpRepository(&stubMachineHelpRepository{returnNilProducts: true})
		catalog, err := repo.GetProducts()
		require.NoError(t, err)
		require.Len(t, catalog.Products, 2)
	})

	t.Run("baseline errors are returned", func(t *testing.T) {
		want := errors.New("products unavailable")
		repo := newPluginAwareHelpRepository(&stubMachineHelpRepository{productsErr: want})
		catalog, err := repo.GetProducts()
		assert.Nil(t, catalog)
		assert.ErrorIs(t, err, want)
	})
}

func TestPluginAwareHelpRepositoryRoutesVersionIndexToBaseline(t *testing.T) {
	originalOwns := metaPluginOwnsProduct
	t.Cleanup(func() { metaPluginOwnsProduct = originalOwns })
	metaPluginOwnsProduct = func(string) bool { return false }

	baseline := &stubMachineHelpRepository{versionIndex: &canonicalmeta.VersionIndex{Version: "v1"}}
	repo := newPluginAwareHelpRepository(baseline)
	index, err := repo.GetVersionIndex("ecs", "v1")
	require.NoError(t, err)
	assert.Same(t, baseline.versionIndex, index)
}

func TestEngineMetadataConvertersCoverCompositeShapes(t *testing.T) {
	parameter := meta.Parameter{
		Name: "payload", RawName: "Payload", Type: meta.TypeObject, Position: meta.PosBody,
		Required: true, DocRequired: true, ParamStyle: "json", IsWildcard: true,
		Enum: []string{"one"}, Minimum: "1", Maximum: "9", MinLength: "2", MaxLength: "8",
		Pattern: "^[a-z]+$", Example: "example", Options: []string{"--payload"},
		Description: meta.Description{ZH: "中文", EN: "English"},
		Fields: []meta.Parameter{
			{Name: "name", RawName: "Name", Type: meta.TypeString, Required: true, DocRequired: true,
				Enum: []string{"a"}, Minimum: "1", Maximum: "2", MinLength: "3", MaxLength: "4",
				Pattern: "^a$", Example: "a", Description: meta.Description{ZH: "名称", EN: "name"}},
			{Name: "items", RawName: "Items", Type: meta.TypeArray, ItemType: &meta.Parameter{
				Type: meta.TypeObject, Fields: []meta.Parameter{{Name: "enabled", RawName: "Enabled", Type: meta.TypeBoolean}},
			}},
			{Name: "labels", RawName: "Labels", Type: meta.TypeMap, ValueType: &meta.Parameter{
				Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeLong},
			}},
		},
	}

	converted := engineParameterToCanonical(&parameter)
	assert.Equal(t, "object", converted.Type)
	assert.Equal(t, "body", converted.Location)
	assert.True(t, converted.Required)
	assert.True(t, converted.DocRequired)
	assert.True(t, converted.IsWildcard)
	assert.Equal(t, "json", converted.ParamStyle)
	assert.Equal(t, []string{"one"}, converted.Enum)
	assert.Equal(t, "中文", converted.HelpZh)
	assert.Equal(t, "English", converted.HelpEn)
	require.Len(t, converted.Fields, 3)
	assert.Equal(t, "string", converted.Fields[0].Type)
	assert.Equal(t, []string{"a"}, converted.Fields[0].Enum)
	assert.Equal(t, "名称", converted.Fields[0].HelpZh)
	require.NotNil(t, converted.Fields[1].Element)
	assert.Equal(t, "object", converted.Fields[1].Element.Type)
	require.Len(t, converted.Fields[1].Element.Fields, 1)
	assert.Equal(t, "bool", converted.Fields[1].Element.Fields[0].Type)
	require.NotNil(t, converted.Fields[2].Value)
	assert.Equal(t, "array", converted.Fields[2].Value.Type)
	require.NotNil(t, converted.Fields[2].Value.Element)
	assert.Equal(t, "long", converted.Fields[2].Value.Element.Type)

	array := engineParameterToCanonical(&meta.Parameter{Type: meta.TypeArray, ItemType: &meta.Parameter{
		Type: meta.TypeString, Enum: []string{"x"}, Minimum: "1", Maximum: "3",
		MinLength: "1", MaxLength: "2", Pattern: "x+",
	}})
	require.NotNil(t, array.Element)
	assert.Equal(t, []string{"x"}, array.Element.Enum)
	assert.Equal(t, "x+", array.Element.Pattern)

	mapParameter := engineParameterToCanonical(&meta.Parameter{Type: meta.TypeMap, ValueType: &meta.Parameter{Type: meta.TypeFloat}})
	require.NotNil(t, mapParameter.Value)
	assert.Equal(t, "float", mapParameter.Value.Type)

	assert.Equal(t, canonicalmeta.Parameter{}, engineParameterToCanonical(nil))
	assert.Nil(t, engineParametersToCanonical(nil))
	assert.Nil(t, engineFieldsToCanonical(nil))
	assert.Nil(t, engineShapeToCanonical(nil))
}

func TestEngineTypePositionAndStyleMappings(t *testing.T) {
	types := []struct {
		in   meta.DataType
		want string
	}{
		{meta.TypeString, "string"}, {meta.TypeInteger, "int"}, {meta.TypeLong, "long"},
		{meta.TypeFloat, "float"}, {meta.TypeBoolean, "bool"}, {meta.TypeObject, "object"},
		{meta.TypeArray, "array"}, {meta.TypeMap, "map"}, {meta.TypeAny, "any"},
		{meta.DataType("future"), "any"},
	}
	for _, test := range types {
		assert.Equal(t, test.want, engineTypeName(test.in), string(test.in))
	}

	positions := []struct {
		in   meta.Position
		want string
	}{
		{meta.PosBody, "body"}, {meta.PosHeader, "header"}, {meta.PosPath, "path"},
		{meta.PosFormData, "formData"}, {meta.PosHost, "host"}, {meta.PosQuery, "query"},
		{meta.Position("future"), "query"},
	}
	for _, test := range positions {
		assert.Equal(t, test.want, enginePositionName(test.in), string(test.in))
	}
	assert.Equal(t, "ROA", engineAPIStyle(meta.StyleROA))
	assert.Equal(t, "RPC", engineAPIStyle(meta.StyleRPC))
}

type stubMachineHelpRepository struct {
	apis              map[string]*canonicalmeta.API
	products          *canonicalmeta.ProductsIndex
	productsErr       error
	returnNilProducts bool
	versionIndex      *canonicalmeta.VersionIndex
}

func (s *stubMachineHelpRepository) GetProducts() (*canonicalmeta.ProductsIndex, error) {
	if s.productsErr != nil {
		return nil, s.productsErr
	}
	if s.returnNilProducts {
		return nil, nil
	}
	if s.products != nil {
		return s.products, nil
	}
	return &canonicalmeta.ProductsIndex{Products: []canonicalmeta.ProductEntry{{Code: "baseline"}}}, nil
}

func (s *stubMachineHelpRepository) GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	if s.versionIndex != nil {
		return s.versionIndex, nil
	}
	return &canonicalmeta.VersionIndex{Version: version}, nil
}

func (s *stubMachineHelpRepository) GetAPI(product, version, apiName string) (*canonicalmeta.API, error) {
	api, ok := s.apis[product]
	if !ok {
		return nil, fmt.Errorf("stub baseline has no product %q", product)
	}
	return api, nil
}
