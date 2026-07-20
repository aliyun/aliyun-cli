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

package format

import (
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	crschema "github.com/aliyun/aliyun-openapi-runtime/schema"
)

// This file is the ONLY place that knows both the on-disk schema
// (aliyun-openapi-runtime/schema) and the runtime model (../meta).
// Every mapping decision lives here so it can evolve independently of
// both sides.

// schemaToAPI converts one CommandDefinition into meta.API. Endpoints
// are NOT set here — they live at the product level in the canonical
// layout and are injected by the Source after decoding.
func schemaToAPI(def *crschema.CommandDefinition) *meta.API {
	api := &meta.API{
		Name:        def.Name,
		CmdName:     def.CmdName,
		CmdFullName: def.CmdFullName,
		Description: meta.Description{ZH: def.DescriptionZH, EN: def.DescriptionEN},
		Deprecated:  def.Deprecated,
		Examples:    exampleList(def),
	}
	if def.Operation != nil {
		api.Version = def.Operation.APIVersion
		api.Method = strings.ToUpper(def.Operation.Method)
		api.URL = def.Operation.URL
		api.Style = mapStyle(def.Operation.APIStyle)
		api.Protocol = def.Operation.Protocol
	}
	api.Parameters = mapArguments(def.Parameters)
	return api
}

// exampleList prefers the kebab example (matches the CLI's kebab command form); falls back to the camel example.
func exampleList(def *crschema.CommandDefinition) []string {
	var out []string
	if def.KebabExample != "" {
		out = append(out, def.KebabExample)
	} else if def.CamelExample != "" {
		out = append(out, def.CamelExample)
	}
	return out
}

// ProductEntryToProduct maps a products.json catalog entry into a
// meta.Product, with endpoints attached. code is the canonical
// (lower-case) product directory name used as the routing identifier.
func ProductEntryToProduct(e *crschema.ProductEntry, code string) *meta.Product {
	p := &meta.Product{
		Code:           code,
		DefaultVersion: e.PluginDefaultVersion,
		Versions:       append([]string(nil), e.Versions...),
		Description:    meta.Description{ZH: mapLookup(e.Name, "zh"), EN: mapLookup(e.Name, "en")},
		Endpoints: meta.Endpoints{
			Global: e.GlobalEndpoint,
			Public: e.RegionalEndpoints,
			VPC:    e.RegionalVPCEndpoints,
		},
	}
	if p.DefaultVersion == "" {
		if e.Version != "" {
			p.DefaultVersion = e.Version
		} else if len(p.Versions) > 0 {
			p.DefaultVersion = chooseDefaultVersion(p.Versions)
		}
	}
	return p
}

func mapLookup(m map[string]string, k string) string {
	if m == nil {
		return ""
	}
	return m[k]
}

func mapArguments(args []crschema.ArgumentDefinition) []meta.Parameter {
	if len(args) == 0 {
		return nil
	}
	out := make([]meta.Parameter, 0, len(args))
	for i := range args {
		out = append(out, mapArgument(&args[i]))
	}
	return out
}

// Handles all four composite shapes: scalar / object / array / map.
func mapArgument(a *crschema.ArgumentDefinition) meta.Parameter {
	p := meta.Parameter{
		Name:        a.Name,
		RawName:     a.RawName,
		Type:        mapType(a.Type),
		Position:    mapPosition(a.Location),
		Required:    a.Required,
		Default:     a.Default,
		Options:     a.Options,
		Description: meta.Description{ZH: a.HelpZH, EN: a.HelpEN},
		ParamStyle:  a.ParamStyle,
	}

	switch p.Type {
	case meta.TypeObject:
		p.Fields = mapArguments(a.Fields)
	case meta.TypeArray:
		p.ItemType = mapItemType(a)
	case meta.TypeMap:
		p.ValueType = mapValueType(a)
	}
	return p
}

// mapItemType builds the ItemType descriptor for an array.
// If the element type is object, ElementFields describes its structure;
// otherwise ElementType names a scalar.
func mapItemType(a *crschema.ArgumentDefinition) *meta.Parameter {
	item := meta.Parameter{
		Type: mapType(a.ElementType),
	}
	if item.Type == meta.TypeObject {
		item.Fields = mapArguments(a.ElementFields)
	}
	return &item
}

// mapValueType builds the ValueType descriptor for a map.
// Analogous to mapItemType but for map<string, T>.
func mapValueType(a *crschema.ArgumentDefinition) *meta.Parameter {
	val := meta.Parameter{
		Type: mapType(a.ValueType),
	}
	if val.Type == meta.TypeObject {
		val.Fields = mapArguments(a.ValueFields)
	}
	return &val
}

// mapType normalises the schema's free-form Type string to a
// meta.DataType. Unknown values fall through to TypeAny so a stale
// generator does not crash the runtime.
func mapType(t string) meta.DataType {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string", "":
		return meta.TypeString
	case "int", "int32", "integer":
		return meta.TypeInteger
	case "long", "int64":
		return meta.TypeLong
	case "float", "double", "number":
		return meta.TypeFloat
	case "bool", "boolean":
		return meta.TypeBoolean
	case "object", "struct":
		return meta.TypeObject
	case "array", "list", "repeatlist":
		return meta.TypeArray
	case "map":
		return meta.TypeMap
	case "any":
		return meta.TypeAny
	default:
		return meta.TypeAny
	}
}

func mapPosition(loc string) meta.Position {
	switch strings.ToLower(strings.TrimSpace(loc)) {
	case "query", "":
		return meta.PosQuery
	case "body":
		return meta.PosBody
	case "header":
		return meta.PosHeader
	case "path":
		return meta.PosPath
	case "formdata":
		return meta.PosFormData
	case "host":
		return meta.PosHost
	default:
		return meta.PosQuery
	}
}

func mapStyle(s string) meta.APIStyle {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "RPC", "":
		return meta.StyleRPC
	case "ROA", "RESTFUL":
		return meta.StyleROA
	default:
		return meta.StyleRPC
	}
}
