// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
package help

import (
	"encoding/json"
	"io"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

const SchemaVersion = "v1"

type Section string

const (
	SectionRequest  Section = "request"
	SectionResponse Section = "response"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// HelpOptions contains renderer-independent Runtime Help v1 policy inputs.
type HelpOptions struct {
	Section          Section
	ExplicitSection  bool
	Search           string
	All              bool
	AIMode           bool
	Format           Format
	Language         string
	RequestedVersion string
}

func (o HelpOptions) normalized() HelpOptions {
	if o.Section == "" {
		o.Section = SectionRequest
	}
	if o.Format == "" {
		o.Format = FormatText
	}
	if o.Language != "zh" {
		o.Language = "en"
	}
	return o
}

type LocalizedText struct {
	EN string `json:"en,omitempty"`
	ZH string `json:"zh,omitempty"`
}

func (t LocalizedText) Text(language string) string {
	if language == "zh" {
		if t.ZH != "" {
			return t.ZH
		}
		return t.EN
	}
	if t.EN != "" {
		return t.EN
	}
	return t.ZH
}

func localized(value meta.Description) LocalizedText {
	return LocalizedText{EN: value.EN, ZH: value.ZH}
}

type Target struct {
	Product    string `json:"product"`
	API        string `json:"api,omitempty"`
	APIVersion string `json:"apiVersion"`
}

type Result struct {
	Shown             int  `json:"shown"`
	Total             int  `json:"total"`
	Truncated         bool `json:"truncated"`
	OmittedDeprecated int  `json:"omittedDeprecated,omitempty"`
}

type Next struct {
	ShowAll     string `json:"showAll,omitempty"`
	Search      string `json:"search,omitempty"`
	SearchAll   string `json:"searchAll,omitempty"`
	ChildSearch string `json:"childSearch,omitempty"`

	operation string
	childKind helpHintKind
	childName string
}

type AIModeHint struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

// MetadataProvenance identifies the metadata layer that supplied a Help
// document. It intentionally contains stable strings rather than source.Kind.
type MetadataProvenance struct {
	Kind          string `json:"kind"`
	PluginName    string `json:"pluginName,omitempty"`
	PluginVersion string `json:"pluginVersion,omitempty"`
	APIVersion    string `json:"apiVersion,omitempty"`
	BundledBy     string `json:"bundledBy,omitempty"`
	Origin        string `json:"origin,omitempty"`
}

type Product struct {
	Code            string        `json:"code"`
	Name            LocalizedText `json:"name"`
	Description     LocalizedText `json:"description"`
	SelectedVersion string        `json:"selectedVersion"`
	Versions        []string      `json:"versions"`
}

type APISummary struct {
	Name        string        `json:"name"`
	Command     string        `json:"command"`
	Title       LocalizedText `json:"title"`
	Description LocalizedText `json:"description"`
	Deprecated  bool          `json:"deprecated,omitempty"`
}

type ProductDocument struct {
	SchemaVersion string              `json:"schemaVersion"`
	Kind          string              `json:"helpLevel"`
	Target        Target              `json:"-"`
	Provenance    *MetadataProvenance `json:"-"`
	Query         string              `json:"query,omitempty"`
	Product       Product             `json:"product"`
	APIs          []APISummary        `json:"apis"`
	Result        Result              `json:"result"`
	Next          *Next               `json:"next,omitempty"`
	AIModeHint    *AIModeHint         `json:"aiModeHint,omitempty"`
}

type Operation struct {
	Style           meta.APIStyle `json:"style"`
	Method          string        `json:"method,omitempty"`
	Protocol        string        `json:"protocol,omitempty"`
	URL             string        `json:"url,omitempty"`
	RequestBodyType string        `json:"requestBodyType,omitempty"`
	ContentType     string        `json:"contentType,omitempty"`
	IsSSE           bool          `json:"isSSE,omitempty"`
	HasWildcardPath bool          `json:"hasWildcardPath,omitempty"`
}

type Constraints struct {
	Enum      []string `json:"enum,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Minimum   string   `json:"minimum,omitempty"`
	Maximum   string   `json:"maximum,omitempty"`
	MinLength string   `json:"minLength,omitempty"`
	MaxLength string   `json:"maxLength,omitempty"`
}

type Parameter struct {
	Name          string        `json:"name"`
	RawName       string        `json:"rawName,omitempty"`
	Options       []string      `json:"options,omitempty"`
	Type          meta.DataType `json:"type"`
	Location      meta.Position `json:"location,omitempty"`
	Required      bool          `json:"required"`
	Serialization string        `json:"serialization,omitempty"`
	Help          LocalizedText `json:"help"`
	Example       string        `json:"example,omitempty"`
	Constraints   Constraints   `json:"constraints"`
	Fields        []Parameter   `json:"fields,omitempty"`
	Element       *Parameter    `json:"element,omitempty"`
	Value         *Parameter    `json:"value,omitempty"`
}

type GlobalParameter struct {
	Name string        `json:"name"`
	Type string        `json:"type"`
	Help LocalizedText `json:"help"`
}

type Examples struct {
	Kebab string `json:"kebab,omitempty"`
}

type QueryExample struct {
	Path          string `json:"path"`
	SchemaCommand string `json:"schemaCommand"`
	QueryCommand  string `json:"queryCommand"`
}

type QueryOption struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Required   bool          `json:"required"`
	HasDefault bool          `json:"hasDefault,omitempty"`
	Default    string        `json:"default,omitempty"`
	Help       LocalizedText `json:"help"`
}

// ActionDocument is the compact default view returned by
// `aliyun <product> <kebab-action> --help`.
type ActionDocument struct {
	SchemaVersion    string              `json:"schemaVersion"`
	Kind             string              `json:"helpLevel"`
	Section          Section             `json:"section"`
	Target           Target              `json:"-"`
	Provenance       *MetadataProvenance `json:"-"`
	Query            string              `json:"query,omitempty"`
	Name             string              `json:"name"`
	Command          string              `json:"command"`
	CmdFullName      string              `json:"cmdFullName,omitempty"`
	Title            LocalizedText       `json:"title"`
	Description      LocalizedText       `json:"description,omitempty"`
	Deprecated       bool                `json:"deprecated,omitempty"`
	MultiVersion     bool                `json:"multiVersion,omitempty"`
	Operation        Operation           `json:"operation"`
	Parameters       []Parameter         `json:"parameters"`
	GlobalParameters []GlobalParameter   `json:"globalParameters"`
	QueryOptions     []QueryOption       `json:"queryOptions"`
	Examples         Examples            `json:"examples"`
	ResponseQuery    *QueryExample       `json:"responseQueryExample,omitempty"`
	Result           Result              `json:"result"`
	Next             *Next               `json:"next,omitempty"`
	AIModeHint       *AIModeHint         `json:"aiModeHint,omitempty"`
}

// RequestDocument is the complete request metadata view selected explicitly
// with `--cli-section request`.
type RequestDocument struct {
	SchemaVersion    string              `json:"schemaVersion"`
	Kind             string              `json:"helpLevel"`
	Section          Section             `json:"section"`
	Target           Target              `json:"-"`
	Provenance       *MetadataProvenance `json:"-"`
	Query            string              `json:"query,omitempty"`
	Product          Product             `json:"product"`
	Name             string              `json:"name"`
	Command          string              `json:"command"`
	CmdFullName      string              `json:"cmdFullName,omitempty"`
	Title            LocalizedText       `json:"title"`
	Description      LocalizedText       `json:"description"`
	Deprecated       bool                `json:"deprecated,omitempty"`
	MultiVersion     bool                `json:"multiVersion,omitempty"`
	Operation        Operation           `json:"operation"`
	Parameters       []Parameter         `json:"parameters"`
	GlobalParameters []GlobalParameter   `json:"globalParameters"`
	QueryOptions     []QueryOption       `json:"queryOptions"`
	Examples         Examples            `json:"examples"`
	ResponseQuery    *QueryExample       `json:"responseQueryExample,omitempty"`
	Result           Result              `json:"result"`
	Next             *Next               `json:"next,omitempty"`
	AIModeHint       *AIModeHint         `json:"aiModeHint,omitempty"`
}

// APIRequestDocument is retained as a source-compatible name for callers that
// adopted the pre-v1 draft API.
type APIRequestDocument = RequestDocument

// APIParameterDocument is the kebab-only detailed view of one top-level API
// parameter. Nested fields, elements and map values remain attached to
// Parameter so composite request shapes are preserved.
type APIParameterDocument struct {
	SchemaVersion string              `json:"schemaVersion"`
	Kind          string              `json:"helpLevel"`
	Section       Section             `json:"section"`
	Target        Target              `json:"-"`
	Provenance    *MetadataProvenance `json:"-"`
	Product       Product             `json:"product"`
	Name          string              `json:"name"`
	Command       string              `json:"command"`
	Parameter     Parameter           `json:"parameter"`
	Query         string              `json:"query,omitempty"`
	Matches       []ParameterMatch    `json:"matches,omitempty"`
	Result        Result              `json:"result"`
	Next          *Next               `json:"next,omitempty"`
	AIModeHint    *AIModeHint         `json:"aiModeHint,omitempty"`
	Candidates    []string            `json:"candidates,omitempty"`
}

type ParameterMatch struct {
	Path      string    `json:"path"`
	Parameter Parameter `json:"parameter"`
}

// ResponseDocumentation is the format-neutral response metadata supplied by
// a loader/codec adapter. Raw JSON is retained losslessly until localization.
type ResponseDocumentation struct {
	Responses                json.RawMessage
	Schema                   json.RawMessage
	StatusCode               string
	ContentType              string
	Components               map[string]json.RawMessage
	PaginationCollectionPath string
	Warnings                 []string
}

type OutputSchema struct {
	StatusCode  string                     `json:"statusCode,omitempty"`
	ContentType string                     `json:"contentType,omitempty"`
	Schema      json.RawMessage            `json:"schema"`
	Components  map[string]json.RawMessage `json:"components,omitempty"`
}

type APIResponseDocument struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Kind          string                     `json:"helpLevel"`
	Section       Section                    `json:"section"`
	Target        Target                     `json:"-"`
	Provenance    *MetadataProvenance        `json:"-"`
	Query         string                     `json:"query,omitempty"`
	Responses     json.RawMessage            `json:"responses,omitempty"`
	Components    map[string]json.RawMessage `json:"components,omitempty"`
	OutputSchema  *OutputSchema              `json:"outputSchema,omitempty"`
	Matches       []string                   `json:"matches,omitempty"`
	Warnings      []string                   `json:"warnings,omitempty"`
	Notice        LocalizedText              `json:"notice,omitempty"`
	ResponseQuery *QueryExample              `json:"responseQueryExample,omitempty"`
	Result        Result                     `json:"result"`
	Next          *Next                      `json:"next,omitempty"`
	AIModeHint    *AIModeHint                `json:"aiModeHint,omitempty"`
}

// DataSource is deliberately compatible with the useful subset of loader.Loader.
type DataSource interface {
	EnsureProduct(code string) error
	LookupProduct(code string) *meta.Product
	ResolveVersion(product, requested string) (string, error)
	GetAPIIndex(product, version string) (*meta.APIIndex, error)
	GetAPI(product, version, name string) (*meta.API, error)
}

// ResponseSource isolates response-schema storage from request metadata.
type ResponseSource interface {
	GetResponseDocumentation(product, version, api string) (*ResponseDocumentation, error)
}

type Renderer interface {
	Render(io.Writer, any, HelpOptions) error
}
