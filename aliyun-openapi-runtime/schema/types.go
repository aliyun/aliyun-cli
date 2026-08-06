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

// Package schema defines the on-disk JSON structures of the shared
// aliyun-openapi-meta "canonical" dataset consumed by aliyun-openapi-runtime.
//
// Layout (relative to the embedded FS root):
//
//	canonical/<product>/<version>/<APIName>.json   per-API command definition
//	canonical/<product>/<version>/version.json     per-version API index
//	metadatas/products.json                        product catalog + endpoints
//
// This is the *data contract*; it is intentionally decoupled from the
// runtime's in-memory model (aliyun-openapi-runtime/meta) so the disk format
// can evolve without dragging the abstraction along. The mapping lives
// in aliyun-openapi-runtime/format.
package schema

// ============================================================================
// On-disk layout constants
// ============================================================================

const (
	// CanonicalRoot holds per-product/version API definitions and the
	// per-version index file.
	CanonicalRoot = "canonical"

	// MetadatasRoot holds the product catalog.
	MetadatasRoot = "metadatas"

	// ProductsFile is the product catalog path relative to the FS root.
	ProductsFile = "metadatas/products.json"

	// VersionFileName is the per-version index basename inside a
	// canonical/<product>/<version>/ directory.
	VersionFileName = "version.json"

	MetadataIndexFile = "metadata.index.json"
	MetadataDataFile  = "metadata.jsonl"
)

// Shared metadata-plugin contract identity. Layout-specific names
// (e.g. format/jsonl.LayoutName, format/pbmeta.LayoutName) live under format/.
// Shared index + ReadAt access lives in format/indexed.
const (
	SchemaName    = "aliyun-openapi-meta"
	SchemaVersion = 1
	LayoutVersion = 1

	FormatJSON     = "json"
	FormatProtobuf = "protobuf"
)

// MetadataDescriptor declares the encoding, schema and physical layout of an interpreted metadata plugin.
type MetadataDescriptor struct {
	Format        string `json:"format"` // FormatJSON or FormatProtobuf
	Schema        string `json:"schema"`
	SchemaVersion int    `json:"schemaVersion"`
	Layout        string `json:"layout"`
	LayoutVersion int    `json:"layoutVersion"`
	Index         string `json:"index"`
	Data          string `json:"data"`
}

type ManifestAPIVersions struct {
	Default   string   `json:"default,omitempty"`
	Supported []string `json:"supported,omitempty"`
}

// PluginManifest is the subset of the package manifest needed by the runtime.
type PluginManifest struct {
	Name             string              `json:"name"`
	Version          string              `json:"version"`
	Type             string              `json:"type,omitempty"`
	ProductCode      string              `json:"productCode,omitempty"`
	ProductName      map[string]string   `json:"productName,omitempty"`
	Command          string              `json:"command"`
	ShortDescription string              `json:"shortDescription"`
	Description      string              `json:"description"`
	MinCliVersion    string              `json:"minCliVersion,omitempty"`
	APIVersions      ManifestAPIVersions `json:"apiVersions,omitempty"`
	Metadata         *MetadataDescriptor `json:"metadata,omitempty"`
	Bin              struct {
		Path string `json:"path"`
	} `json:"bin,omitempty"`
}

// Distribution values on a product entry.
// A product ships as exactly one form:
// a compiled Go binary plugin ("go", run as a separate process)
// or interpreted JSON metadata ("meta", the aliyun-openapi-runtime engine).
// An empty value means "not marked" and is treated as engine-served (so an unpopulated catalog behaves as today).
const (
	DistributionGo   = "go"
	DistributionMeta = "meta"
)

// ============================================================================
// Product catalog (metadatas/products.json)
// ============================================================================

// ProductsIndex is the deserialized shape of metadatas/products.json.
type ProductsIndex struct {
	Products []ProductEntry `json:"products"`
}

// ProductEntry is one product's catalog record. Endpoints live here at the product level.
type ProductEntry struct {
	Code                 string            `json:"code"`
	Name                 map[string]string `json:"name"` // en / zh
	APIStyle             string            `json:"api_style"`
	GlobalEndpoint       string            `json:"global_endpoint"`
	RegionalEndpoints    map[string]string `json:"regional_endpoints"`
	RegionalVPCEndpoints map[string]string `json:"regional_vpc_endpoints"`
	LocationServiceCode  string            `json:"location_service_code"`
	PluginDefaultVersion string            `json:"plugin_default_version"`
	Version              string            `json:"version"`
	Versions             []string          `json:"versions"`
	APIs                 []string          `json:"apis"`

	// Distribution is "go" | "meta" | "" (unset). "go" makes the engine abstain so the product routes to its Go plugin instead.
	Distribution string `json:"distribution,omitempty"`
}

// ============================================================================
// Per-version index (canonical/<product>/<version>/version.json)
// ============================================================================

type VersionIndex struct {
	APIs    map[string]VersionAPIEntry `json:"apis"` // key = APIName (PascalCase)
	Style   string                     `json:"style"`
	Version string                     `json:"version"`
}

// VersionAPIEntry is one API's lightweight index entry:
// enough to build command stubs and the kebab->API route table without loading the full per-API JSON.
type VersionAPIEntry struct {
	CmdName       string `json:"cmd_name"`
	Deprecated    bool   `json:"deprecated"`
	DescriptionZH string `json:"description_zh"`
	DescriptionEN string `json:"description_en"`
}

// ============================================================================
// Per-API command definition (canonical/<product>/<version>/<APIName>.json)
// ============================================================================

// CommandDefinition is the per-API JSON shape. Most root fields are optional,
// but composite parameters are strict: arrays require element and maps require
// value at every nesting level.
type CommandDefinition struct {
	Name          string               `json:"name"`
	CmdName       string               `json:"cmd_name"`
	CmdFullName   string               `json:"cmd_full_name"`
	DescriptionZH string               `json:"description_zh,omitempty"`
	DescriptionEN string               `json:"description_en,omitempty"`
	Method        string               `json:"method,omitempty"`
	MultiVersion  bool                 `json:"multi_version"`
	Deprecated    bool                 `json:"deprecated"`
	KebabExample  string               `json:"kebab_example,omitempty"`
	CamelExample  string               `json:"camel_example,omitempty"`
	Operation     *OperationConfig     `json:"operation"`
	Parameters    []ArgumentDefinition `json:"parameters,omitempty"`
}

// OperationConfig describes the HTTP request shape for one API.
type OperationConfig struct {
	Action          string `json:"action"`
	APIStyle        string `json:"api_style,omitempty"`
	APIVersion      string `json:"api_version"`
	Method          string `json:"method"`
	Protocol        string `json:"protocol,omitempty"`
	URL             string `json:"url"`
	IsSSE           bool   `json:"is_sse,omitempty"`
	ReqBodyType     string `json:"req_body_type,omitempty"`
	ContentType     string `json:"content_type,omitempty"`
	HasWildcardPath bool   `json:"has_wildcard_path,omitempty"`
}

// TypeShape recursively describes an array element or map value. Its structure
// is independent of nesting depth: arrays use Element, maps use Value, and
// objects use Fields.
type TypeShape struct {
	Type    string               `json:"type"`
	Fields  []ArgumentDefinition `json:"fields,omitempty"`
	Element *TypeShape           `json:"element,omitempty"`
	Value   *TypeShape           `json:"value,omitempty"`
}

// ArgumentDefinition describes a single CLI parameter.
type ArgumentDefinition struct {
	Name       string               `json:"name"`
	RawName    string               `json:"raw_name"`
	Type       string               `json:"type"`
	Options    []string             `json:"options,omitempty"`
	HelpZH     string               `json:"help_zh,omitempty"`
	HelpEN     string               `json:"help_en,omitempty"`
	Example    string               `json:"example,omitempty"`
	Required   bool                 `json:"required"`
	Location   string               `json:"location,omitempty"`
	ParamStyle string               `json:"param_style,omitempty"`
	IsWildcard bool                 `json:"is_wildcard,omitempty"`
	Fields     []ArgumentDefinition `json:"fields,omitempty"`
	Element    *TypeShape           `json:"element,omitempty"`
	Value      *TypeShape           `json:"value,omitempty"`
}
