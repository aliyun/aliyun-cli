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
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/openapi/runtimehost"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

// pluginAwareHelpRepository serves Machine Help from the bundled canonical
// tree, except for products owned by an installed metadata plugin: those are
// served through the engine loader (user plugin > override > baseline), so a
// hot-updated plugin immediately wins over the packed baseline and Help stays
// byte-compatible with the no-plugin kebab experience.
type pluginAwareHelpRepository struct {
	baseline machineHelpRepository
}

func newPluginAwareHelpRepository(baseline machineHelpRepository) machineHelpRepository {
	if baseline == nil {
		return nil
	}
	return pluginAwareHelpRepository{baseline: baseline}
}

func (r pluginAwareHelpRepository) GetProducts() (*canonicalmeta.ProductsIndex, error) {
	catalog, err := r.baseline.GetProducts()
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		catalog = &canonicalmeta.ProductsIndex{}
	}
	for _, entry := range metaPluginCatalogEntries() {
		merged := false
		for i := range catalog.Products {
			if strings.EqualFold(catalog.Products[i].Code, entry.Code) {
				catalog.Products[i] = entry
				merged = true
				break
			}
		}
		if !merged {
			catalog.Products = append(catalog.Products, entry)
		}
	}
	return catalog, nil
}

func (r pluginAwareHelpRepository) GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	if metaPluginOwnsProduct(product) {
		return engineVersionIndex(product, version)
	}
	return r.baseline.GetVersionIndex(product, version)
}

func (r pluginAwareHelpRepository) GetAPI(product, version, apiName string) (*canonicalmeta.API, error) {
	if metaPluginOwnsProduct(product) {
		return engineAPI(product, version, apiName)
	}
	return r.baseline.GetAPI(product, version, apiName)
}

func installedMetaPluginEntries() []canonicalmeta.ProductEntry {
	mgr, err := plugin.NewManager()
	if err != nil {
		return nil
	}
	manifest, err := mgr.GetLocalManifest()
	if err != nil || manifest == nil {
		return nil
	}
	entries := make([]canonicalmeta.ProductEntry, 0, len(manifest.Plugins))
	for _, local := range manifest.Plugins {
		if !local.IsMeta() || strings.TrimSpace(local.ProductCode) == "" {
			continue
		}
		entry, ok := engineProductEntry(strings.ToLower(local.ProductCode))
		if ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// The two hooks below are package vars so tests can force plugin ownership
// without installing a real metadata plugin package.
var (
	metaPluginOwnsProduct    = defaultMetaPluginOwnsProduct
	metaPluginCatalogEntries = installedMetaPluginEntries
)

func defaultMetaPluginOwnsProduct(product string) bool {
	provenance := runtimehost.MetaPluginProvenance(product)
	return provenance != nil && provenance.Kind != source.KindBaseline
}

func engineProductEntry(product string) (canonicalmeta.ProductEntry, bool) {
	ldr, err := runtimehost.Engine().Loader()
	if err != nil {
		return canonicalmeta.ProductEntry{}, false
	}
	if err := ldr.EnsureProduct(product); err != nil {
		return canonicalmeta.ProductEntry{}, false
	}
	loaded := ldr.LookupProduct(product)
	if loaded == nil {
		return canonicalmeta.ProductEntry{}, false
	}
	return canonicalmeta.ProductEntry{
		Code:                 strings.ToLower(loaded.Code),
		Name:                 map[string]string{"zh": loaded.Name.ZH, "en": loaded.Name.EN},
		PluginDefaultVersion: loaded.DefaultVersion,
		Versions:             append([]string(nil), loaded.Versions...),
		Distribution:         "meta",
	}, true
}

func engineVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	ldr, err := runtimehost.Engine().Loader()
	if err != nil {
		return nil, err
	}
	resolved, err := ldr.ResolveVersion(product, version)
	if err != nil {
		return nil, err
	}
	index, err := ldr.GetAPIIndex(product, resolved)
	if err != nil {
		return nil, err
	}
	converted := &canonicalmeta.VersionIndex{
		APIs:    make(map[string]canonicalmeta.VersionAPIEntry, len(index.Entries)),
		Version: index.Version,
	}
	for name, entry := range index.Entries {
		converted.APIs[name] = canonicalmeta.VersionAPIEntry{
			CmdName:       entry.CmdName,
			Deprecated:    entry.Deprecated,
			DescriptionZh: entry.Description.ZH,
			DescriptionEn: entry.Description.EN,
		}
	}
	return converted, nil
}

func engineAPI(product, version, apiName string) (*canonicalmeta.API, error) {
	ldr, err := runtimehost.Engine().Loader()
	if err != nil {
		return nil, err
	}
	resolved, err := ldr.ResolveVersion(product, version)
	if err != nil {
		return nil, err
	}
	loaded, err := ldr.GetAPI(product, resolved, apiName)
	if err != nil {
		return nil, err
	}
	return engineAPIToCanonical(loaded), nil
}

func engineAPIToCanonical(api *meta.API) *canonicalmeta.API {
	converted := &canonicalmeta.API{
		Name:          api.Name,
		CmdName:       api.CmdName,
		CmdFullName:   api.CmdFullName,
		ProductCode:   api.ProductCode,
		Deprecated:    api.Deprecated,
		Protocol:      api.Protocol,
		Method:        api.Method,
		PathPattern:   api.URL,
		DescriptionZh: api.Description.ZH,
		DescriptionEn: api.Description.EN,
		Operation: &canonicalmeta.Operation{
			Action:          api.Name,
			APIStyle:        engineAPIStyle(api.Style),
			APIVersion:      api.Version,
			Method:          api.Method,
			Protocol:        api.Protocol,
			URL:             api.URL,
			IsSSE:           api.IsSSE,
			ReqBodyType:     api.ReqBodyType,
			ContentType:     api.ContentType,
			HasWildcardPath: api.HasWildcardPath,
			BodyMapping:     api.BodyMapping,
		},
		Parameters: engineParametersToCanonical(api.Parameters),
	}
	// The engine keeps only the kebab-preferred example; kebab is the only
	// command style metadata plugins expose.
	if len(api.Examples) > 0 {
		converted.KebabExample = api.Examples[0]
	}
	return converted
}

func engineAPIStyle(style meta.APIStyle) string {
	if style == meta.StyleROA {
		return "ROA"
	}
	return "RPC"
}

func engineParametersToCanonical(parameters []meta.Parameter) []canonicalmeta.Parameter {
	if len(parameters) == 0 {
		return nil
	}
	out := make([]canonicalmeta.Parameter, 0, len(parameters))
	for i := range parameters {
		out = append(out, engineParameterToCanonical(&parameters[i]))
	}
	return out
}

func engineParameterToCanonical(p *meta.Parameter) canonicalmeta.Parameter {
	if p == nil {
		return canonicalmeta.Parameter{}
	}
	converted := canonicalmeta.Parameter{
		Name:        p.Name,
		RawName:     p.RawName,
		Type:        engineTypeName(p.Type),
		Required:    p.Required,
		DocRequired: p.DocRequired,
		Location:    enginePositionName(p.Position),
		ParamStyle:  p.ParamStyle,
		IsWildcard:  p.IsWildcard,
		Enum:        append([]string(nil), p.Enum...),
		Minimum:     p.Minimum,
		Maximum:     p.Maximum,
		MinLength:   p.MinLength,
		MaxLength:   p.MaxLength,
		Pattern:     p.Pattern,
		Example:     p.Example,
		Options:     append([]string(nil), p.Options...),
		HelpZh:      p.Description.ZH,
		HelpEn:      p.Description.EN,
	}
	if p.Type == meta.TypeObject {
		converted.Fields = engineFieldsToCanonical(p.Fields)
	} else if p.Type == meta.TypeArray {
		converted.Element = engineShapeToCanonical(p.ItemType)
	} else if p.Type == meta.TypeMap {
		converted.Value = engineShapeToCanonical(p.ValueType)
	}
	return converted
}

func engineFieldsToCanonical(fields []meta.Parameter) []canonicalmeta.Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]canonicalmeta.Field, 0, len(fields))
	for i := range fields {
		f := engineParameterToCanonical(&fields[i])
		converted := canonicalmeta.Field{
			Name:          f.Name,
			RawName:       f.RawName,
			Type:          f.Type,
			Required:      f.Required,
			DocRequired:   f.DocRequired,
			Format:        f.Format,
			Minimum:       f.Minimum,
			Maximum:       f.Maximum,
			MinLength:     f.MinLength,
			MaxLength:     f.MaxLength,
			Pattern:       f.Pattern,
			Example:       f.Example,
			Enum:          f.Enum,
			DescriptionZh: f.DescriptionZh,
			DescriptionEn: f.DescriptionEn,
			HelpZh:        f.HelpZh,
			HelpEn:        f.HelpEn,
		}
		if f.Type == "object" {
			converted.Fields = engineFieldsToCanonical(fields[i].Fields)
		} else if f.Type == "array" {
			converted.Element = engineShapeToCanonical(fields[i].ItemType)
		} else if f.Type == "map" {
			converted.Value = engineShapeToCanonical(fields[i].ValueType)
		}
		out = append(out, converted)
	}
	return out
}

func engineShapeToCanonical(shape *meta.Parameter) *canonicalmeta.TypeShape {
	if shape == nil {
		return nil
	}
	converted := &canonicalmeta.TypeShape{
		Type:      engineTypeName(shape.Type),
		Enum:      append([]string(nil), shape.Enum...),
		Minimum:   shape.Minimum,
		Maximum:   shape.Maximum,
		MinLength: shape.MinLength,
		MaxLength: shape.MaxLength,
		Pattern:   shape.Pattern,
	}
	if shape.Type == meta.TypeObject {
		for _, field := range engineFieldsToCanonical(shape.Fields) {
			converted.Fields = append(converted.Fields, field)
		}
	} else if shape.Type == meta.TypeArray {
		converted.Element = engineShapeToCanonical(shape.ItemType)
	} else if shape.Type == meta.TypeMap {
		converted.Value = engineShapeToCanonical(shape.ValueType)
	}
	return converted
}

func engineTypeName(t meta.DataType) string {
	switch t {
	case meta.TypeString:
		return "string"
	case meta.TypeInteger:
		return "int"
	case meta.TypeLong:
		return "long"
	case meta.TypeFloat:
		return "float"
	case meta.TypeBoolean:
		return "bool"
	case meta.TypeObject:
		return "object"
	case meta.TypeArray:
		return "array"
	case meta.TypeMap:
		return "map"
	default:
		return "any"
	}
}

func enginePositionName(pos meta.Position) string {
	switch pos {
	case meta.PosBody:
		return "body"
	case meta.PosHeader:
		return "header"
	case meta.PosPath:
		return "path"
	case meta.PosFormData:
		return "formData"
	case meta.PosHost:
		return "host"
	default:
		return "query"
	}
}
