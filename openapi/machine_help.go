package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

const machineHelpSchemaVersion = "v1"

type machineHelpRepository interface {
	GetProducts() (*canonicalmeta.ProductsIndex, error)
	GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error)
	GetAPI(product, version, apiName string) (*canonicalmeta.API, error)
}

type machineHelpService struct {
	repository machineHelpRepository
}

type machineHelpErrorBody struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Target      []string `json:"target"`
	Suggestions []string `json:"suggestions"`
}

type machineHelpErrorDocument struct {
	SchemaVersion string               `json:"schemaVersion"`
	Error         machineHelpErrorBody `json:"error"`
}

type machineHelpError struct {
	document machineHelpErrorDocument
	cause    error
}

func newMachineHelpError(code, message string, target, suggestions []string) *machineHelpError {
	return &machineHelpError{document: machineHelpErrorDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Error: machineHelpErrorBody{
			Code:        code,
			Message:     message,
			Target:      append([]string(nil), target...),
			Suggestions: append([]string(nil), suggestions...),
		},
	}}
}

func (e *machineHelpError) Error() string {
	if e.cause != nil {
		return e.document.Error.Message + ": " + e.cause.Error()
	}
	return e.document.Error.Message
}

func (e *machineHelpError) Unwrap() error { return e.cause }

func (e *machineHelpError) RenderError(w io.Writer) error {
	return encodeMachineHelpJSON(w, e.document)
}

func (e *machineHelpError) ExitCode() int { return 2 }

func newMachineHelpService(repository machineHelpRepository) *machineHelpService {
	return &machineHelpService{repository: repository}
}

type machineHelpTarget struct {
	Path           []string `json:"path"`
	RequestedStyle string   `json:"requestedStyle"`
}

type machineHelpLocalizedText struct {
	EN string `json:"en"`
	ZH string `json:"zh"`
}

type machineHelpCommandSummary struct {
	Name        string                   `json:"name"`
	Description machineHelpLocalizedText `json:"description"`
}

type machineHelpProductSummary struct {
	Code          string                   `json:"code"`
	Name          machineHelpLocalizedText `json:"name"`
	CommandStyles []string                 `json:"commandStyles"`
	CanonicalHelp bool                     `json:"canonicalHelp"`
	Distribution  string                   `json:"distribution,omitempty"`
}

type machineHelpRootDocument struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Kind          string                      `json:"kind"`
	Target        machineHelpTarget           `json:"target"`
	Commands      []machineHelpCommandSummary `json:"commands"`
	Products      []machineHelpProductSummary `json:"products"`
	Listing       *machineHelpListing         `json:"listing"`
	AIModeHint    *machineHelpAIModeHint      `json:"aiModeHint"`
}

type machineHelpProduct struct {
	Code                 string                   `json:"code"`
	Name                 machineHelpLocalizedText `json:"name"`
	APIStyle             string                   `json:"apiStyle"`
	LegacyDefaultVersion string                   `json:"legacyDefaultVersion"`
	PluginDefaultVersion string                   `json:"pluginDefaultVersion"`
	SupportedVersions    []string                 `json:"supportedVersions"`
	SelectedVersion      string                   `json:"selectedVersion"`
	Distribution         string                   `json:"distribution,omitempty"`
}

type machineHelpAPISummary struct {
	Name        string                   `json:"name"`
	CmdName     string                   `json:"cmdName"`
	Description machineHelpLocalizedText `json:"description"`
	Deprecated  bool                     `json:"deprecated"`
}

type machineHelpProductDocument struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Kind          string                  `json:"kind"`
	Target        machineHelpTarget       `json:"target"`
	Product       machineHelpProduct      `json:"product"`
	APIs          []machineHelpAPISummary `json:"apis"`
	Listing       *machineHelpListing     `json:"listing"`
	AIModeHint    *machineHelpAIModeHint  `json:"aiModeHint"`
}

type machineHelpOperation struct {
	Action          string `json:"action"`
	APIStyle        string `json:"apiStyle"`
	APIVersion      string `json:"apiVersion"`
	Method          string `json:"method"`
	Protocol        string `json:"protocol"`
	URL             string `json:"url"`
	IsSSE           bool   `json:"isSSE"`
	ReqBodyType     string `json:"requestBodyType"`
	ContentType     string `json:"contentType"`
	HasWildcardPath bool   `json:"hasWildcardPath"`
}

type machineHelpAPI struct {
	Name         string                   `json:"name"`
	CmdName      string                   `json:"cmdName"`
	CmdFullName  string                   `json:"cmdFullName"`
	Description  machineHelpLocalizedText `json:"description"`
	Deprecated   bool                     `json:"deprecated"`
	MultiVersion bool                     `json:"multiVersion"`
	Operation    machineHelpOperation     `json:"operation"`
}

type machineHelpConstraints struct {
	Enum    any `json:"enum"`
	Pattern any `json:"pattern"`
	Minimum any `json:"minimum"`
	Maximum any `json:"maximum"`
}

type machineHelpShape struct {
	Type    string                 `json:"type"`
	Fields  []machineHelpParameter `json:"fields,omitempty"`
	Element *machineHelpShape      `json:"element,omitempty"`
	Value   *machineHelpShape      `json:"value,omitempty"`
}

type machineHelpParameter struct {
	Name          string                   `json:"name"`
	RawName       string                   `json:"rawName"`
	Options       []string                 `json:"options"`
	Type          string                   `json:"type"`
	Location      string                   `json:"location"`
	Required      bool                     `json:"required"`
	Serialization string                   `json:"serialization"`
	Help          machineHelpLocalizedText `json:"help"`
	Example       string                   `json:"example"`
	Constraints   machineHelpConstraints   `json:"constraints"`
	Fields        []machineHelpParameter   `json:"fields,omitempty"`
	Element       *machineHelpShape        `json:"element,omitempty"`
	Value         *machineHelpShape        `json:"value,omitempty"`
}

type machineHelpParameterSets struct {
	Camel []machineHelpParameter `json:"camel"`
	Kebab []machineHelpParameter `json:"kebab"`
}

type machineHelpExamples struct {
	Camel string `json:"camel"`
	Kebab string `json:"kebab"`
}

type machineHelpAPIDocument struct {
	SchemaVersion      string                   `json:"schemaVersion"`
	Kind               string                   `json:"kind"`
	Section            string                   `json:"section"`
	Target             machineHelpTarget        `json:"target"`
	Product            machineHelpProduct       `json:"product"`
	API                machineHelpAPI           `json:"api"`
	ActiveParameterSet string                   `json:"activeParameterSet"`
	ParameterSets      machineHelpParameterSets `json:"parameterSets"`
	GlobalParameters   []machineHelpParameter   `json:"globalParameters"`
	Examples           machineHelpExamples      `json:"examples"`
	OutputSchema       any                      `json:"outputSchema"`
	Pagination         any                      `json:"pagination"`
	Risk               any                      `json:"risk"`
	Recovery           any                      `json:"recovery"`
	ResponseQuery      *machineHelpQueryExample `json:"responseQueryExample"`
	AIModeHint         *machineHelpAIModeHint   `json:"aiModeHint"`
}

type machineHelpListing struct {
	Shown int    `json:"shown"`
	Total int    `json:"total"`
	Hint  string `json:"hint"`
}

type machineHelpAIModeHint struct {
	Command string `json:"command"`
	Message string `json:"message"`
}

type machineHelpQueryExample struct {
	Path          string `json:"path"`
	SchemaCommand string `json:"schemaCommand"`
	QueryCommand  string `json:"queryCommand"`
}

type machineHelpComponents struct {
	Schemas map[string]json.RawMessage `json:"schemas"`
}

type machineHelpOutputSchema struct {
	StatusCode  string                 `json:"statusCode"`
	ContentType string                 `json:"contentType"`
	Schema      json.RawMessage        `json:"schema"`
	Components  *machineHelpComponents `json:"components"`
}

type machineHelpAPIResponseDocument struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Kind          string                   `json:"kind"`
	Section       string                   `json:"section"`
	Target        machineHelpTarget        `json:"target"`
	Product       machineHelpProduct       `json:"product"`
	API           machineHelpAPI           `json:"api"`
	OutputSchema  *machineHelpOutputSchema `json:"outputSchema"`
	Matches       []string                 `json:"matches"`
	Notice        string                   `json:"notice"`
	Warnings      []string                 `json:"warnings"`
	ResponseQuery *machineHelpQueryExample `json:"responseQueryExample"`
	AIModeHint    *machineHelpAIModeHint   `json:"aiModeHint"`
}

type resolvedMachineHelpAPI struct {
	Product  canonicalmeta.ProductEntry
	Versions []string
	Selected string
	Style    string
	Command  string
	API      *canonicalmeta.API
}

func (s *machineHelpService) buildRoot(root *cli.Command) (*machineHelpRootDocument, error) {
	if root == nil {
		return nil, fmt.Errorf("root command is nil")
	}
	catalog, err := s.repository.GetProducts()
	if err != nil {
		return nil, err
	}

	metadata := map[string]*cli.Metadata{}
	root.GetMetadata(metadata)
	commands := make([]machineHelpCommandSummary, 0, len(root.SubCommandNames()))
	for _, name := range root.SubCommandNames() {
		entry := metadata[root.Name+" "+name]
		if entry == nil || entry.Hidden {
			continue
		}
		commands = append(commands, machineHelpCommandSummary{
			Name:        name,
			Description: localizedText(entry.Short),
		})
	}
	sort.Slice(commands, func(i, j int) bool { return commands[i].Name < commands[j].Name })

	products := make([]machineHelpProductSummary, 0, len(catalog.Products))
	for _, product := range catalog.Products {
		versions := normalizedVersions(product)
		styles := []string(nil)
		if len(versions) > 0 {
			styles = []string{"camel", "kebab"}
		}
		products = append(products, machineHelpProductSummary{
			Code:          strings.ToLower(product.Code),
			Name:          localizedText(product.Name),
			CommandStyles: styles,
			CanonicalHelp: len(versions) > 0,
			Distribution:  product.Distribution,
		})
	}
	sort.Slice(products, func(i, j int) bool { return products[i].Code < products[j].Code })

	return &machineHelpRootDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "root",
		Target:        machineHelpTarget{Path: []string{"aliyun"}, RequestedStyle: "root"},
		Commands:      commands,
		Products:      products,
	}, nil
}

func (s *machineHelpService) buildProduct(code, requestedVersion string) (*machineHelpProductDocument, error) {
	product, err := s.findProduct(code)
	if err != nil {
		return nil, err
	}
	versions := normalizedVersions(*product)
	selected, err := selectProductVersion(*product, versions, requestedVersion)
	if err != nil {
		return nil, err
	}
	index, err := s.repository.GetVersionIndex(product.Code, selected)
	if err != nil {
		return nil, err
	}

	apis := make([]machineHelpAPISummary, 0, len(index.APIs))
	for name, entry := range index.APIs {
		apis = append(apis, machineHelpAPISummary{
			Name:        name,
			CmdName:     entry.CmdName,
			Description: machineHelpLocalizedText{EN: entry.DescriptionEn, ZH: entry.DescriptionZh},
			Deprecated:  entry.Deprecated,
		})
	}
	sort.Slice(apis, func(i, j int) bool {
		if apis[i].CmdName == apis[j].CmdName {
			return apis[i].Name < apis[j].Name
		}
		return apis[i].CmdName < apis[j].CmdName
	})

	productDoc := buildMachineHelpProduct(*product, versions, selected)
	code = productDoc.Code
	return &machineHelpProductDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "product",
		Target:        machineHelpTarget{Path: []string{"aliyun", code}, RequestedStyle: "product"},
		Product:       productDoc,
		APIs:          apis,
	}, nil
}

func (s *machineHelpService) buildAPI(code, command, requestedVersion string) (*machineHelpAPIDocument, error) {
	resolved, err := s.resolveAPI(code, command, requestedVersion)
	if err != nil {
		return nil, err
	}
	api := resolved.API

	operation := projectMachineHelpOperation(api)
	camelParameters := make([]machineHelpParameter, 0)
	for _, view := range api.LegacyTopLevelParameters() {
		camelParameters = append(camelParameters, projectLegacyParameter(view, ""))
	}
	kebabParameters := make([]machineHelpParameter, 0, len(api.Parameters))
	for i := range api.Parameters {
		kebabParameters = append(kebabParameters, projectCanonicalParameter(&api.Parameters[i]))
	}

	productDoc := buildMachineHelpProduct(resolved.Product, resolved.Versions, resolved.Selected)
	document := &machineHelpAPIDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "api",
		Section:       helpSectionRequest,
		Target: machineHelpTarget{
			Path:           []string{"aliyun", productDoc.Code, command},
			RequestedStyle: resolved.Style,
		},
		Product: productDoc,
		API: machineHelpAPI{
			Name:         api.Name,
			CmdName:      api.CmdName,
			CmdFullName:  api.CmdFullName,
			Description:  machineHelpLocalizedText{EN: api.DescriptionEn, ZH: api.DescriptionZh},
			Deprecated:   api.Deprecated,
			MultiVersion: api.MultiVersion,
			Operation:    operation,
		},
		ActiveParameterSet: resolved.Style,
		ParameterSets: machineHelpParameterSets{
			Camel: camelParameters,
			Kebab: kebabParameters,
		},
		GlobalParameters: make([]machineHelpParameter, 0),
		Examples: machineHelpExamples{
			Camel: api.CamelExample,
			Kebab: api.KebabExample,
		},
		OutputSchema: nil,
		Pagination:   nil,
		Risk:         nil,
		Recovery:     nil,
	}
	document.ResponseQuery = projectCanonicalResponseQueryExample(api, productDoc.Code, command, resolved.Style, requestedVersion)
	return document, nil
}

func (s *machineHelpService) buildAPIResponse(code, command, requestedVersion string) (*machineHelpAPIResponseDocument, error) {
	resolved, err := s.resolveAPI(code, command, requestedVersion)
	if err != nil {
		return nil, err
	}
	api := resolved.API
	response, err := api.ResponseSchema()
	if err != nil {
		return nil, err
	}

	productDoc := buildMachineHelpProduct(resolved.Product, resolved.Versions, resolved.Selected)
	document := &machineHelpAPIResponseDocument{
		SchemaVersion: machineHelpSchemaVersion,
		Kind:          "api",
		Section:       helpSectionResponse,
		Target: machineHelpTarget{
			Path:           []string{"aliyun", productDoc.Code, command},
			RequestedStyle: resolved.Style,
		},
		Product: productDoc,
		API: machineHelpAPI{
			Name:         api.Name,
			CmdName:      api.CmdName,
			CmdFullName:  api.CmdFullName,
			Description:  machineHelpLocalizedText{EN: api.DescriptionEn, ZH: api.DescriptionZh},
			Deprecated:   api.Deprecated,
			MultiVersion: api.MultiVersion,
			Operation:    projectMachineHelpOperation(api),
		},
		Warnings: response.Warnings,
	}
	if !response.HasSchema() {
		document.Notice = "No response schema is available for this API."
		return document, nil
	}

	output := &machineHelpOutputSchema{
		StatusCode:  response.StatusCode,
		ContentType: response.ContentType,
		Schema:      response.Schema,
	}
	if len(response.Components) > 0 {
		output.Components = &machineHelpComponents{Schemas: response.Components}
	}
	document.OutputSchema = output
	document.ResponseQuery = projectResponseQueryExample(
		HelpResponseSchema{Schema: response.Schema, Components: response.Components},
		productDoc.Code,
		command,
		resolved.Style,
		requestedVersion,
	)
	return document, nil
}

func projectCanonicalResponseQueryExample(api *canonicalmeta.API, product, command, style, requestedVersion string) *machineHelpQueryExample {
	if api == nil {
		return nil
	}
	response, err := api.ResponseSchema()
	if err != nil || !response.HasSchema() {
		return nil
	}
	return projectResponseQueryExample(
		HelpResponseSchema{Schema: response.Schema, Components: response.Components},
		product,
		command,
		style,
		requestedVersion,
	)
}

func projectResponseQueryExample(schema HelpResponseSchema, product, command, style, requestedVersion string) *machineHelpQueryExample {
	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document:   schema,
		Product:    product,
		API:        command,
		APIVersion: requestedVersion,
		Style:      responseCommandStyle(style),
	})
	if err != nil || example == nil {
		return nil
	}
	return &machineHelpQueryExample{
		Path:          example.Path,
		SchemaCommand: example.SchemaCommand,
		QueryCommand:  example.QueryCommand,
	}
}

func projectMachineHelpOperation(api *canonicalmeta.API) machineHelpOperation {
	if api == nil {
		return machineHelpOperation{}
	}
	if api.Operation == nil {
		return machineHelpOperation{Method: api.Method, Protocol: api.Protocol, URL: api.PathPattern}
	}
	return machineHelpOperation{
		Action:          api.Operation.Action,
		APIStyle:        api.Operation.APIStyle,
		APIVersion:      api.Operation.APIVersion,
		Method:          api.Operation.Method,
		Protocol:        api.Operation.Protocol,
		URL:             api.Operation.URL,
		IsSSE:           api.Operation.IsSSE,
		ReqBodyType:     api.Operation.ReqBodyType,
		ContentType:     api.Operation.ContentType,
		HasWildcardPath: api.Operation.HasWildcardPath,
	}
}

func (s *machineHelpService) resolveAPI(code, command, requestedVersion string) (*resolvedMachineHelpAPI, error) {
	product, err := s.findProduct(code)
	if err != nil {
		return nil, err
	}
	style := "camel"
	if strings.ToLower(command) == command {
		style = "kebab"
	}
	versions := normalizedVersions(*product)
	selected, err := selectAPIVersion(*product, versions, requestedVersion, style)
	if err != nil {
		return nil, err
	}
	index, err := s.repository.GetVersionIndex(product.Code, selected)
	if err != nil {
		return nil, err
	}
	apiName := resolveAPIName(index, command, style)
	if apiName == "" {
		return nil, newMachineHelpError(
			"UNKNOWN_API",
			fmt.Sprintf("unknown API command %q for product %q version %q", command, product.Code, selected),
			[]string{"aliyun", strings.ToLower(product.Code), command},
			[]string{"inspect product help to list available APIs"},
		)
	}
	api, err := s.repository.GetAPI(product.Code, selected, apiName)
	if err != nil {
		return nil, err
	}
	return &resolvedMachineHelpAPI{
		Product:  *product,
		Versions: versions,
		Selected: selected,
		Style:    style,
		Command:  command,
		API:      api,
	}, nil
}

func buildMachineHelpProduct(product canonicalmeta.ProductEntry, versions []string, selected string) machineHelpProduct {
	return machineHelpProduct{
		Code:                 strings.ToLower(product.Code),
		Name:                 localizedText(product.Name),
		APIStyle:             product.APIStyle,
		LegacyDefaultVersion: product.Version,
		PluginDefaultVersion: product.PluginDefaultVersion,
		SupportedVersions:    versions,
		SelectedVersion:      selected,
		Distribution:         product.Distribution,
	}
}

func selectAPIVersion(product canonicalmeta.ProductEntry, versions []string, requested, style string) (string, error) {
	if requested != "" {
		return selectProductVersion(product, versions, requested)
	}
	if style == "camel" && product.Version != "" {
		return product.Version, nil
	}
	return selectProductVersion(product, versions, "")
}

func resolveAPIName(index *canonicalmeta.VersionIndex, command, style string) string {
	for name, entry := range index.APIs {
		if style == "camel" && strings.EqualFold(name, command) {
			return name
		}
		if style == "kebab" && entry.CmdName == command {
			return name
		}
	}
	return ""
}

func projectCanonicalParameter(parameter *canonicalmeta.Parameter) machineHelpParameter {
	if parameter == nil {
		return machineHelpParameter{}
	}
	result := machineHelpParameter{
		Name:          parameter.Name,
		RawName:       parameter.RawName,
		Options:       append([]string(nil), parameter.Options...),
		Type:          parameter.Type,
		Location:      strings.ToLower(parameter.Location),
		Required:      parameter.Required,
		Serialization: parameter.ParamStyle,
		Help:          machineHelpLocalizedText{EN: parameter.HelpEn, ZH: parameter.HelpZh},
		Example:       parameter.Example,
	}
	for i := range parameter.Fields {
		result.Fields = append(result.Fields, projectCanonicalField(&parameter.Fields[i]))
	}
	result.Element = projectCanonicalShape(parameter.Element)
	result.Value = projectCanonicalShape(parameter.Value)
	return result
}

func projectCanonicalField(field *canonicalmeta.Field) machineHelpParameter {
	if field == nil {
		return machineHelpParameter{}
	}
	result := machineHelpParameter{
		Name:        field.Name,
		RawName:     field.RawName,
		Options:     make([]string, 0),
		Type:        field.Type,
		Required:    field.Required,
		Help:        machineHelpLocalizedText{EN: field.HelpEn, ZH: field.HelpZh},
		Example:     field.Example,
		Constraints: machineHelpConstraints{},
	}
	for i := range field.Fields {
		result.Fields = append(result.Fields, projectCanonicalField(&field.Fields[i]))
	}
	result.Element = projectCanonicalShape(field.Element)
	result.Value = projectCanonicalShape(field.Value)
	return result
}

func projectCanonicalShape(shape *canonicalmeta.TypeShape) *machineHelpShape {
	if shape == nil {
		return nil
	}
	result := &machineHelpShape{Type: shape.Type}
	for i := range shape.Fields {
		result.Fields = append(result.Fields, projectCanonicalField(&shape.Fields[i]))
	}
	result.Element = projectCanonicalShape(shape.Element)
	result.Value = projectCanonicalShape(shape.Value)
	return result
}

func projectLegacyParameter(view *canonicalmeta.LegacyParameterView, prefix string) machineHelpParameter {
	name := view.LegacyName()
	optionPath := name
	if prefix != "" {
		optionPath = prefix + "." + name
	}
	result := machineHelpParameter{
		Name:        name,
		RawName:     name,
		Options:     []string{"--" + optionPath},
		Type:        view.LegacyType(),
		Location:    strings.ToLower(view.LegacyPosition()),
		Required:    view.LegacyRequired(),
		Help:        machineHelpLocalizedText{EN: view.LegacyDescription("en"), ZH: view.LegacyDescription("zh")},
		Example:     view.LegacyExample(),
		Constraints: machineHelpConstraints{},
	}
	children := view.LegacyChildren()
	if view.IsLegacyRepeatList() {
		result.Serialization = "repeatList"
		if len(children) > 0 {
			result.Element = &machineHelpShape{Type: "object"}
			for _, child := range children {
				result.Element.Fields = append(result.Element.Fields, projectLegacyParameter(child, optionPath+".#"))
			}
		}
	}
	return result
}

func (s *machineHelpService) findProduct(code string) (*canonicalmeta.ProductEntry, error) {
	catalog, err := s.repository.GetProducts()
	if err != nil {
		return nil, err
	}
	for i := range catalog.Products {
		if strings.EqualFold(catalog.Products[i].Code, code) {
			return &catalog.Products[i], nil
		}
	}
	return nil, newMachineHelpError(
		"UNKNOWN_PRODUCT",
		fmt.Sprintf("unknown product %q", code),
		[]string{"aliyun", strings.ToLower(code)},
		[]string{"inspect root help to list available products"},
	)
}

func selectProductVersion(product canonicalmeta.ProductEntry, versions []string, requested string) (string, error) {
	if requested != "" {
		for _, version := range versions {
			if version == requested {
				return requested, nil
			}
		}
		return "", newMachineHelpError(
			"UNKNOWN_VERSION",
			fmt.Sprintf("product %q does not expose version %q", product.Code, requested),
			[]string{"aliyun", strings.ToLower(product.Code)},
			[]string{"inspect supportedVersions in product help"},
		)
	}
	for _, candidate := range []string{product.PluginDefaultVersion, product.Version} {
		if candidate != "" {
			return candidate, nil
		}
	}
	if len(versions) == 0 {
		return "", newMachineHelpError(
			"UNKNOWN_VERSION",
			fmt.Sprintf("product %q has no API version", product.Code),
			[]string{"aliyun", strings.ToLower(product.Code)},
			nil,
		)
	}
	return versions[0], nil
}

func normalizedVersions(product canonicalmeta.ProductEntry) []string {
	seen := map[string]bool{}
	versions := make([]string, 0, len(product.Versions)+2)
	for _, version := range append(append([]string(nil), product.Versions...), product.Version, product.PluginDefaultVersion) {
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
}

func localizedText(values map[string]string) machineHelpLocalizedText {
	return machineHelpLocalizedText{EN: values["en"], ZH: values["zh"]}
}

func (c *Commando) printMachineHelp(ctx *cli.Context, args []string, format string, opts helpOptions) error {
	target := append([]string{"aliyun"}, args...)
	if format != "json" {
		return newMachineHelpError(
			"INVALID_FORMAT",
			fmt.Sprintf("unsupported help format %q", format),
			target,
			[]string{"use --help=json or help --format json"},
		)
	}
	if len(args) > 2 {
		return newMachineHelpError(
			"INVALID_TARGET",
			fmt.Sprintf("machine help accepts at most a product and an API, got %d arguments", len(args)),
			target,
			nil,
		)
	}
	if c.library == nil || c.library.helpRepo == nil {
		return newMachineHelpUnavailableError(target, errors.New("canonical metadata repository is unavailable"))
	}

	service := newMachineHelpService(c.library.helpRepo)
	aiMode := legacyAIModeEnabled(ctx)
	var (
		document any
		err      error
	)
	switch len(args) {
	case 0:
		document, err = service.buildRoot(ctx.Command())
	case 1:
		document, err = service.buildProduct(args[0], requestedMachineHelpVersion(ctx))
	case 2:
		if opts.Section == helpSectionResponse {
			document, err = service.buildAPIResponse(args[0], args[1], requestedMachineHelpVersion(ctx))
		} else {
			document, err = service.buildAPI(args[0], args[1], requestedMachineHelpVersion(ctx))
		}
	}
	if err != nil {
		var structured *machineHelpError
		if errors.As(err, &structured) {
			return structured
		}
		return newMachineHelpUnavailableError(target, err)
	}
	if api, ok := document.(*machineHelpAPIDocument); ok {
		api.GlobalParameters = projectGlobalParameters(ctx.Flags())
	}
	switch typed := document.(type) {
	case *machineHelpRootDocument:
		applyRootHelpOptions(typed, opts, aiMode)
	case *machineHelpProductDocument:
		applyProductHelpOptions(typed, opts, aiMode)
	case *machineHelpAPIDocument:
		applyRequestHelpOptions(typed, opts)
	case *machineHelpAPIResponseDocument:
		if applyErr := applyResponseHelpOptions(typed, opts); applyErr != nil {
			return newMachineHelpUnavailableError(target, applyErr)
		}
		if opts.Search != "" {
			typed.ResponseQuery = nil
			if typed.OutputSchema != nil {
				typed.ResponseQuery = projectResponseQueryExample(
					helpResponseSchema(typed),
					typed.Product.Code,
					typed.Target.Path[len(typed.Target.Path)-1],
					typed.Target.RequestedStyle,
					requestedMachineHelpVersion(ctx),
				)
			}
		}
	}
	if !aiMode {
		hint := cli.NewAIModeHint()
		projected := &machineHelpAIModeHint{Command: hint.Command, Message: hint.Message}
		switch typed := document.(type) {
		case *machineHelpRootDocument:
			typed.AIModeHint = projected
		case *machineHelpProductDocument:
			typed.AIModeHint = projected
		case *machineHelpAPIDocument:
			typed.AIModeHint = projected
		case *machineHelpAPIResponseDocument:
			typed.AIModeHint = projected
		}
	}
	if err := encodeMachineHelpJSON(ctx.Stdout(), document); err != nil {
		return newMachineHelpUnavailableError(target, err)
	}
	return nil
}

func newMachineHelpUnavailableError(target []string, cause error) *machineHelpError {
	err := newMachineHelpError(
		"MACHINE_HELP_UNAVAILABLE",
		"machine-readable help is unavailable",
		target,
		nil,
	)
	err.cause = cause
	return err
}

func encodeMachineHelpJSON(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var document any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return err
	}
	document, _ = pruneMachineHelpEmptyAt(document, nil)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

// pruneMachineHelpEmpty removes values that carry no machine-readable
// information while preserving false and numeric zero, which are meaningful
// for fields such as required, deprecated, and listing counts.
func pruneMachineHelpEmpty(value any) (any, bool) {
	return pruneMachineHelpEmptyAt(value, nil)
}

func pruneMachineHelpEmptyAt(value any, path []string) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case string:
		return typed, typed != ""
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			// Empty strings can be meaningful members of enum-like arrays. Only
			// omit the array when the array itself is empty.
			if text, ok := item.(string); ok {
				result = append(result, text)
				continue
			}
			if cleaned, keep := pruneMachineHelpEmptyAt(item, path); keep {
				result = append(result, cleaned)
			}
		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if preserveMachineHelpSchema(path, key) {
				result[key] = item
				continue
			}
			if cleaned, keep := pruneMachineHelpEmptyAt(item, append(path, key)); keep {
				result[key] = cleaned
			}
		}
		if len(result) == 0 {
			return nil, false
		}
		return result, true
	default:
		return value, true
	}
}

func preserveMachineHelpSchema(path []string, key string) bool {
	if len(path) == 1 && path[0] == "outputSchema" && key == "schema" {
		return true
	}
	return len(path) == 2 && path[0] == "outputSchema" && path[1] == "components" && key == "schemas"
}

func requestedMachineHelpVersion(ctx *cli.Context) string {
	if ctx != nil && ctx.UnknownFlags() != nil {
		if value, ok := ctx.UnknownFlags().GetValue("api-version"); ok && value != "" {
			return value
		}
	}
	if ctx != nil {
		if value, ok := VersionFlag(ctx.Flags()).GetValue(); ok {
			return value
		}
	}
	return ""
}

func projectGlobalParameters(flags *cli.FlagSet) []machineHelpParameter {
	parameters := make([]machineHelpParameter, 0)
	if flags == nil {
		return parameters
	}
	for _, flag := range flags.Flags() {
		if flag == nil || flag.Hidden || flag.Name == "help" {
			continue
		}
		parameterType := "string"
		serialization := "once"
		switch flag.AssignedMode {
		case cli.AssignedNone:
			parameterType = "bool"
			serialization = "flag"
		case cli.AssignedRepeatable:
			serialization = "repeatable"
		}
		parameter := machineHelpParameter{
			Name:          flag.Name,
			RawName:       flag.Name,
			Options:       append([]string(nil), flag.GetFormations()...),
			Type:          parameterType,
			Location:      "global",
			Required:      flag.Required,
			Serialization: serialization,
			Example:       flag.DefaultValue,
			Constraints:   machineHelpConstraints{},
		}
		if flag.Short != nil {
			parameter.Help = localizedText(flag.Short.GetData())
		}
		parameters = append(parameters, parameter)
	}
	sort.Slice(parameters, func(i, j int) bool { return parameters[i].Name < parameters[j].Name })
	return parameters
}
