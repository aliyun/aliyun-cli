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

// Package meta defines the in-memory data model of the aliyun-openapi-runtime.
//
// This is the *only* representation of OpenAPI metadata that upper
// layers (Loader/Runtime/Builder/CLI) consume; every codec/format
// implementation must map its wire format to these types. That
// makes it safe to swap the on-disk format (JSON <-> Protobuf <->
// binary cache) without cascading changes.
//
// Design rules:
//   - String enums are preferred over int enums for stability across
//     schema evolutions.
//   - Fields are additive: existing consumers must tolerate zero
//     values for newly added fields.
//   - No JSON tags are declared here. The disk format is decoupled
//     from the Model; mapping lives in ../format.
package meta

// Position describes where a Parameter is carried in the wire
// request. It is only meaningful for top-level parameters; nested
// object fields inherit their carrier's position implicitly.
type Position string

// Enumerated Positions. Additional values may be introduced without
// removing existing ones.
const (
	PosQuery    Position = "query"
	PosBody     Position = "body"
	PosHeader   Position = "header"
	PosPath     Position = "path"
	PosFormData Position = "formData"
	PosHost     Position = "host"
)

// DataType is the logical parameter type. This is the type the CLI
// argument parser expects, not the wire encoding.
type DataType string

// Supported DataTypes. Composite types (Object/Array/Map) require
// further descriptors on Parameter (Fields / ItemType / ValueType).
const (
	TypeString  DataType = "string"
	TypeInteger DataType = "integer"
	TypeLong    DataType = "long"
	TypeFloat   DataType = "float"
	TypeBoolean DataType = "boolean"
	TypeObject  DataType = "object"
	TypeArray   DataType = "array"
	TypeMap     DataType = "map"
	TypeAny     DataType = "any"
)

// APIStyle categorises how an API is invoked and signed.
type APIStyle string

// Enumerated APIStyles. Canonical values match the Go plugin /
// darabonba client: "RPC" and "ROA". Upstream meta may say
// "restful"/"RESTful" (product catalog) or "ROA" (per-API
// operation.api_style); mapStyle normalises restful spellings to
// StyleROA so debug/dry-run/SDK all see "ROA".
const (
	StyleRPC APIStyle = "RPC"
	StyleROA APIStyle = "ROA"

	// StyleRESTful is a deprecated alias of StyleROA.
	StyleRESTful = StyleROA
)

// Description carries bilingual help text. Empty strings are legal
// and MUST be tolerated by consumers.
type Description struct {
	ZH string
	EN string
}

// Localized returns ZH or EN depending on lang ("zh" / "en"),
// falling back to whichever is non-empty when the requested locale
// is missing.
func (d Description) Localized(lang string) string {
	switch lang {
	case "zh":
		if d.ZH != "" {
			return d.ZH
		}
		return d.EN
	default:
		if d.EN != "" {
			return d.EN
		}
		return d.ZH
	}
}

// Endpoints models the region -> host mappings for a product's API.
// Global is the fallback used when neither Public nor VPC has a
// match; either sub-map may be nil.
type Endpoints struct {
	Global string
	Public map[string]string
	VPC    map[string]string
}

// Resolve looks up a hostname for the given region. It prefers a VPC
// endpoint when useVPC is true; falls back to Public; finally to
// Global. Returns "" if nothing matches.
func (e Endpoints) Resolve(region string, useVPC bool) string {
	if useVPC {
		if h, ok := e.VPC[region]; ok {
			return h
		}
	}
	if h, ok := e.Public[region]; ok {
		return h
	}
	return e.Global
}
