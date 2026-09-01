package help

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

const (
	defaultLogicalLineBudget = 100
	productReservedLines     = 15
	searchResultLimit        = 20
)

// Service builds structured documents without depending on a concrete loader.
type Service struct {
	Data      DataSource
	Responses ResponseSource
}

type provenanceSource interface {
	Provenance(product string) *source.Provenance
}

func (s Service) BuildProduct(code, requestedVersion string, options HelpOptions) (*ProductDocument, error) {
	if s.Data == nil {
		return nil, fmt.Errorf("help data source is nil")
	}
	code = strings.ToLower(strings.TrimSpace(code))
	if err := s.Data.EnsureProduct(code); err != nil {
		return nil, err
	}
	product := s.Data.LookupProduct(code)
	if product == nil {
		return nil, fmt.Errorf("product %q is unavailable after loading", code)
	}
	version, err := s.Data.ResolveVersion(code, requestedVersion)
	if err != nil {
		return nil, err
	}
	index, err := s.Data.GetAPIIndex(code, version)
	if err != nil {
		return nil, err
	}
	document := BuildProductDocument(product, index, options)
	document.Provenance = serviceProvenance(s.Data, code)
	return document, nil
}

// BuildAPI is retained for compatibility and returns the complete request
// section. New callers should choose BuildAction or BuildRequest explicitly.
func (s Service) BuildAPI(productCode, version, apiName string, options HelpOptions) (*APIRequestDocument, error) {
	return s.BuildRequest(productCode, version, apiName, options)
}

func (s Service) BuildAction(productCode, version, apiName string, options HelpOptions) (*ActionDocument, error) {
	product, api, response, resolved, err := s.loadAPI(productCode, version, apiName)
	if err != nil {
		return nil, err
	}
	document := BuildActionDocument(product, api, response, options)
	document.Provenance = serviceProvenance(s.Data, strings.ToLower(strings.TrimSpace(productCode)))
	document.Target.APIVersion = resolved
	return document, nil
}

func (s Service) BuildRequest(productCode, version, apiName string, options HelpOptions) (*RequestDocument, error) {
	product, api, response, resolved, err := s.loadAPI(productCode, version, apiName)
	if err != nil {
		return nil, err
	}
	document := BuildRequestDocument(product, api, response, options)
	document.Provenance = serviceProvenance(s.Data, strings.ToLower(strings.TrimSpace(productCode)))
	document.Target.APIVersion = resolved
	return document, nil
}

func (s Service) loadAPI(productCode, version, apiName string) (*meta.Product, *meta.API, *ResponseDocumentation, string, error) {
	if s.Data == nil {
		return nil, nil, nil, "", fmt.Errorf("help data source is nil")
	}
	productCode = strings.ToLower(strings.TrimSpace(productCode))
	if err := s.Data.EnsureProduct(productCode); err != nil {
		return nil, nil, nil, "", err
	}
	product := s.Data.LookupProduct(productCode)
	resolved, err := s.Data.ResolveVersion(productCode, version)
	if err != nil {
		return nil, nil, nil, "", err
	}
	api, err := s.Data.GetAPI(productCode, resolved, apiName)
	if err != nil {
		return nil, nil, nil, "", err
	}
	var response *ResponseDocumentation
	if s.Responses != nil {
		response, err = s.Responses.GetResponseDocumentation(productCode, resolved, api.Name)
		if err != nil {
			return nil, nil, nil, "", err
		}
	}
	return product, api, response, resolved, nil
}

func (s Service) BuildParameter(productCode, version, apiName, parameterName string, options HelpOptions) (*APIParameterDocument, error) {
	if s.Data == nil {
		return nil, fmt.Errorf("help data source is nil")
	}
	productCode = strings.ToLower(strings.TrimSpace(productCode))
	if err := s.Data.EnsureProduct(productCode); err != nil {
		return nil, err
	}
	product := s.Data.LookupProduct(productCode)
	resolved, err := s.Data.ResolveVersion(productCode, version)
	if err != nil {
		return nil, err
	}
	api, err := s.Data.GetAPI(productCode, resolved, apiName)
	if err != nil {
		return nil, err
	}
	document, err := BuildAPIParameterDocument(product, api, parameterName, options)
	if err == nil {
		document.Provenance = serviceProvenance(s.Data, productCode)
	}
	return document, err
}

func (s Service) BuildResponse(productCode, version, apiName string, options HelpOptions) (*APIResponseDocument, error) {
	if s.Data == nil || s.Responses == nil {
		return nil, fmt.Errorf("help data and response sources are required")
	}
	productCode = strings.ToLower(strings.TrimSpace(productCode))
	if err := s.Data.EnsureProduct(productCode); err != nil {
		return nil, err
	}
	resolved, err := s.Data.ResolveVersion(productCode, version)
	if err != nil {
		return nil, err
	}
	api, err := s.Data.GetAPI(productCode, resolved, apiName)
	if err != nil {
		return nil, err
	}
	response, err := s.Responses.GetResponseDocumentation(productCode, resolved, api.Name)
	if err != nil {
		return nil, err
	}
	document, err := BuildAPIResponseDocument(api, response, options)
	if err == nil {
		document.Provenance = serviceProvenance(s.Data, productCode)
	}
	return document, err
}

func serviceProvenance(data DataSource, product string) *MetadataProvenance {
	provider, ok := data.(provenanceSource)
	if !ok {
		return nil
	}
	value := provider.Provenance(product)
	if value == nil {
		return nil
	}
	return &MetadataProvenance{
		Kind: value.Kind.String(), PluginName: value.PluginName,
		PluginVersion: value.PluginVersion, APIVersion: value.APIVersion,
		BundledBy: value.BundledBy, Origin: value.Origin,
	}
}

func BuildProductDocument(product *meta.Product, index *meta.APIIndex, options HelpOptions) *ProductDocument {
	options = options.normalized()
	if product == nil || index == nil {
		return nil
	}
	code := strings.ToLower(product.Code)
	apis := make([]APISummary, 0, len(index.Entries))
	for name, entry := range index.Entries {
		command := entry.CmdName
		if command == "" {
			command = kebabCase(name)
		}
		apis = append(apis, APISummary{
			Name: name, Command: command, Title: localized(entry.Title),
			Description: localized(entry.Description), Deprecated: entry.Deprecated,
		})
	}
	sort.Slice(apis, func(i, j int) bool {
		return strings.ToLower(apis[i].Command) < strings.ToLower(apis[j].Command)
	})
	allAPIs := append([]APISummary(nil), apis...)
	if options.Search != "" {
		apis = searchAPIs(apis, options.Search)
	} else if options.All {
		sort.SliceStable(apis, func(i, j int) bool {
			if apis[i].Deprecated != apis[j].Deprecated {
				return !apis[i].Deprecated
			}
			return false
		})
	} else if !options.All {
		visible := apis[:0]
		for _, api := range apis {
			if !api.Deprecated {
				visible = append(visible, api)
			}
		}
		apis = visible
	}
	total := len(apis)
	omittedDeprecated := 0
	if options.Search == "" && !options.All {
		for _, entry := range index.Entries {
			if entry.Deprecated {
				omittedDeprecated++
			}
		}
	}
	if options.AIMode && !options.All && options.Search == "" {
		limit := defaultLogicalLineBudget - productReservedLines
		if len(apis) > limit {
			apis = apis[:limit]
		}
	}
	if options.Search != "" && !options.All && len(apis) > searchResultLimit {
		apis = apis[:searchResultLimit]
	}
	document := &ProductDocument{
		SchemaVersion: SchemaVersion,
		Kind:          "product",
		Target:        Target{Product: code, APIVersion: index.Version},
		Query:         options.Search,
		Product: Product{
			Code: code, Name: localized(product.Name), Description: localized(product.Description),
			SelectedVersion: index.Version, Versions: append([]string(nil), product.Versions...),
		},
		APIs: apis,
		Result: Result{
			Shown: len(apis), Total: total, Truncated: len(apis) < total,
			OmittedDeprecated: omittedDeprecated,
		},
	}
	document.Next = buildHelpNext(
		helpHintTargetForProduct(document, options), options, document.Result,
		document.Result.Truncated || document.Result.OmittedDeprecated > 0,
		uniqueExactAPIChild(allAPIs, options.Search),
	)
	return document
}

func BuildActionDocument(product *meta.Product, api *meta.API, response *ResponseDocumentation, options HelpOptions) *ActionDocument {
	options = options.normalized()
	request := BuildRequestDocument(product, api, response, options)
	if request == nil {
		return nil
	}
	parameters := request.Parameters
	globals := request.GlobalParameters
	total := len(api.Parameters) + len(globals)
	if options.Search != "" {
		all := make([]Parameter, 0, len(api.Parameters))
		for i := range api.Parameters {
			all = append(all, projectParameter(api.Parameters[i]))
		}
		total = len(searchParameters(all, options.Search)) + len(globals)
	}
	if options.AIMode && options.Search == "" && !options.All {
		parameters = compactParameters(parameters, defaultLogicalLineBudget-8)
	} else if options.Search != "" && !options.All && len(parameters) > searchResultLimit {
		parameters = parameters[:searchResultLimit]
	}
	document := &ActionDocument{
		SchemaVersion:    SchemaVersion,
		Kind:             "api",
		Section:          SectionRequest,
		Target:           request.Target,
		Query:            request.Query,
		Name:             request.Name,
		Command:          request.Command,
		CmdFullName:      request.CmdFullName,
		Title:            request.Title,
		Description:      request.Description,
		Deprecated:       request.Deprecated,
		MultiVersion:     request.MultiVersion,
		Operation:        request.Operation,
		Parameters:       parameters,
		GlobalParameters: globals,
		QueryOptions:     request.QueryOptions,
		Examples:         request.Examples,
		ResponseQuery:    request.ResponseQuery,
		Result: Result{
			Shown: len(parameters) + len(globals), Total: total,
			Truncated: len(parameters)+len(globals) < total,
		},
	}
	if document.Title.EN != "" || document.Title.ZH != "" {
		document.Description = LocalizedText{}
	}
	if document.Operation.Style != meta.StyleROA {
		document.Operation.Method = ""
		document.Operation.Protocol = ""
		document.Operation.URL = ""
		document.Operation.RequestBodyType = ""
		document.Operation.ContentType = ""
		document.Operation.IsSSE = false
		document.Operation.HasWildcardPath = false
	}
	allParameters := make([]Parameter, 0, len(api.Parameters))
	for i := range api.Parameters {
		allParameters = append(allParameters, projectParameter(api.Parameters[i]))
	}
	actionOptions := options
	actionOptions.ExplicitSection = false
	document.Next = buildHelpNext(
		helpHintTargetForAPI(document.Target, actionOptions), actionOptions, document.Result,
		document.Result.Truncated,
		uniqueExactParameterChild(allParameters, projectGlobalParameters(), options.Search),
	)
	return document
}

func BuildAPIRequestDocument(product *meta.Product, api *meta.API, response *ResponseDocumentation, options HelpOptions) *APIRequestDocument {
	return BuildRequestDocument(product, api, response, options)
}

func BuildRequestDocument(product *meta.Product, api *meta.API, response *ResponseDocumentation, options HelpOptions) *RequestDocument {
	options = options.normalized()
	if api == nil {
		return nil
	}
	code := strings.ToLower(api.ProductCode)
	if product != nil && product.Code != "" {
		code = strings.ToLower(product.Code)
	}
	parameters := make([]Parameter, 0, len(api.Parameters))
	for i := range api.Parameters {
		parameters = append(parameters, projectParameter(api.Parameters[i]))
	}
	sort.SliceStable(parameters, func(i, j int) bool {
		if parameters[i].Required != parameters[j].Required {
			return parameters[i].Required
		}
		return parameters[i].Name < parameters[j].Name
	})
	allParameters := append([]Parameter(nil), parameters...)
	if options.Search != "" {
		parameters = searchParameters(parameters, options.Search)
	}
	globals := projectGlobalParameters()
	allGlobals := append([]GlobalParameter(nil), globals...)
	if options.Search != "" {
		globals = searchGlobalParameters(globals, options.Search)
	}
	total := len(parameters) + len(globals)
	if options.Search != "" && !options.All {
		if len(parameters) > searchResultLimit {
			parameters = parameters[:searchResultLimit]
		}
		remaining := searchResultLimit - len(parameters)
		if remaining < len(globals) {
			globals = globals[:remaining]
		}
	}
	command := api.CmdName
	if command == "" {
		command = kebabCase(api.Name)
	}
	cmdFullName := strings.TrimSpace(api.CmdFullName)
	if cmdFullName == "" {
		cmdFullName = strings.TrimSpace(code + " " + command)
	}
	var productDTO Product
	if product != nil {
		productDTO = Product{
			Code: strings.ToLower(product.Code), Name: localized(product.Name), Description: localized(product.Description),
			SelectedVersion: api.Version, Versions: append([]string(nil), product.Versions...),
		}
	}
	document := &RequestDocument{
		SchemaVersion: SchemaVersion, Kind: "api", Section: SectionRequest,
		Target: Target{Product: code, API: command, APIVersion: api.Version},
		Query:  options.Search, Product: productDTO, Name: api.Name, Command: command,
		CmdFullName: cmdFullName, Title: localized(api.Title), Description: localized(api.Description),
		Deprecated: api.Deprecated, MultiVersion: api.MultiVersion,
		Operation: Operation{
			Style: api.Style, Method: api.Method, Protocol: api.Protocol, URL: api.URL,
			RequestBodyType: api.ReqBodyType, ContentType: api.ContentType, IsSSE: api.IsSSE,
			HasWildcardPath: api.HasWildcardPath,
		},
		Parameters:       parameters,
		GlobalParameters: globals,
		QueryOptions:     projectQueryOptions(),
		Result: Result{
			Shown: len(parameters) + len(globals), Total: total,
			Truncated: len(parameters)+len(globals) < total,
		},
	}
	if len(api.Examples) > 0 {
		document.Examples.Kebab = api.Examples[0]
	}
	if response != nil {
		required := requiredFlags(api.Parameters)
		document.ResponseQuery, _ = BuildResponseQueryExample(ResponseQueryContext{
			Document: *response, Product: code, API: api.Name, APIVersion: options.RequestedVersion,
			RequiredFlags:            required,
			PaginationCollectionPath: response.PaginationCollectionPath,
		})
	}
	sectionOptions := options
	sectionOptions.ExplicitSection = true
	sectionOptions.Section = SectionRequest
	document.Next = buildHelpNext(
		helpHintTargetForAPI(document.Target, sectionOptions), sectionOptions, document.Result, false,
		uniqueExactParameterChild(allParameters, allGlobals, options.Search),
	)
	return document
}

func projectQueryOptions() []QueryOption {
	return []QueryOption{
		{
			Name: "--cli-section", Type: "string", HasDefault: true, Default: "request",
			Help: LocalizedText{
				EN: "inspect request parameters or the response schema",
				ZH: "查看 request 请求参数或 response 响应结构",
			},
		},
		{
			Name: "--help-search", Type: "string",
			Help: LocalizedText{EN: "search this API's metadata", ZH: "搜索该 API 的元数据"},
		},
		{
			Name: "--help-all", Type: "boolean",
			Help: LocalizedText{EN: "show every matching entry", ZH: "显示全部匹配项"},
		},
		{
			Name: "--cli-query", Type: "string",
			Help: LocalizedText{EN: "filter command output with JMESPath", ZH: "使用 JMESPath 过滤命令输出"},
		},
	}
}

func projectGlobalParameters() []GlobalParameter {
	flags := argparser.ReservedFlags()
	result := make([]GlobalParameter, 0, len(flags))
	for _, flag := range flags {
		parameterType := "bool"
		if flag.TakesValue {
			parameterType = "string"
		}
		result = append(result, GlobalParameter{
			Name: "--" + flag.Name,
			Type: parameterType,
			Help: LocalizedText{EN: flag.DescEN, ZH: flag.DescZH},
		})
	}
	return result
}

func searchGlobalParameters(parameters []GlobalParameter, keyword string) []GlobalParameter {
	query := searchText(keyword)
	result := make([]GlobalParameter, 0, len(parameters))
	for _, parameter := range parameters {
		if bestSearchRank(query, parameter.Name) > 0 ||
			textContains(parameter.Help.EN+" "+parameter.Help.ZH, query) {
			result = append(result, parameter)
		}
	}
	return result
}

func BuildAPIParameterDocument(product *meta.Product, api *meta.API, parameterName string, options HelpOptions) (*APIParameterDocument, error) {
	options = options.normalized()
	if api == nil {
		return nil, fmt.Errorf("API is required")
	}
	parameterName = strings.TrimLeft(strings.TrimSpace(parameterName), "-")
	matches, candidates := findTopLevelParameters(api.Parameters, parameterName)
	if len(matches) != 1 {
		return nil, &UnknownParameterError{
			API: api.CmdName, Parameter: parameterName,
			Candidates: candidateParameters(candidates, parameterName),
		}
	}
	selected := matches[0]
	projected := projectParameter(*selected)
	command := api.CmdName
	if command == "" {
		command = kebabCase(api.Name)
	}

	code := strings.ToLower(api.ProductCode)
	var productDTO Product
	if product != nil {
		code = strings.ToLower(product.Code)
		productDTO = Product{
			Code: code, Name: localized(product.Name), Description: localized(product.Description),
			SelectedVersion: api.Version, Versions: append([]string(nil), product.Versions...),
		}
	}
	document := &APIParameterDocument{
		SchemaVersion: SchemaVersion,
		Kind:          "parameter",
		Section:       SectionRequest,
		Target:        Target{Product: code, API: command, APIVersion: api.Version},
		Product:       productDTO,
		Name:          api.Name,
		Command:       command,
		Parameter:     projected,
		Result:        Result{Shown: 1, Total: 1},
	}
	if options.Search != "" {
		allMatches := searchParameterDescendants(projected, options.Search)
		total := len(allMatches)
		if !options.All && len(allMatches) > searchResultLimit {
			allMatches = allMatches[:searchResultLimit]
		}
		document.Query = options.Search
		document.Matches = allMatches
		document.Parameter.Fields = nil
		document.Parameter.Element = nil
		document.Parameter.Value = nil
		document.Result = Result{Shown: len(allMatches), Total: total, Truncated: len(allMatches) < total}
	}
	document.Next = buildHelpNext(
		helpHintTargetForParameter(document.Target, projected.Name, options), options, document.Result, false, nil,
	)
	return document, nil
}

// UnknownParameterError exposes stable candidate names for interactive hosts.
type UnknownParameterError struct {
	API        string
	Parameter  string
	Candidates []string
}

func (e *UnknownParameterError) Error() string {
	if len(e.Candidates) == 0 {
		return fmt.Sprintf("unknown parameter --%s for %s", e.Parameter, e.API)
	}
	return fmt.Sprintf("unknown parameter --%s for %s; candidates: %s",
		e.Parameter, e.API, strings.Join(e.Candidates, ", "))
}

type parameterCandidate struct {
	path      string
	parameter *meta.Parameter
	aliases   []string
}

func findTopLevelParameters(parameters []meta.Parameter, query string) ([]*meta.Parameter, []parameterCandidate) {
	var candidates []parameterCandidate
	for i := range parameters {
		parameter := &parameters[i]
		name := strings.TrimLeft(firstOption(parameter.Options, "--"+kebabCase(firstNonEmpty(parameter.RawName, parameter.Name))), "-")
		aliases := []string{name, parameter.Name, parameter.RawName}
		for _, option := range parameter.Options {
			aliases = append(aliases, strings.TrimLeft(option, "-"))
		}
		candidates = append(candidates, parameterCandidate{path: name, parameter: parameter, aliases: aliases})
	}
	var matches []*meta.Parameter
	for _, candidate := range candidates {
		for _, alias := range candidate.aliases {
			if alias != "" && strings.EqualFold(strings.TrimLeft(alias, "-"), query) {
				matches = append(matches, candidate.parameter)
				break
			}
		}
	}
	return matches, candidates
}

type rankedParameterMatch struct {
	match ParameterMatch
	rank  int
}

func searchParameterDescendants(root Parameter, keyword string) []ParameterMatch {
	query := searchText(keyword)
	var ranked []rankedParameterMatch
	var walk func(Parameter, string, bool)
	walk = func(parameter Parameter, path string, include bool) {
		if include {
			if rank := directParameterSearchRank(parameter, query); rank > 0 {
				ranked = append(ranked, rankedParameterMatch{
					match: ParameterMatch{Path: path, Parameter: parameter},
					rank:  rank,
				})
			}
		}
		for _, field := range parameter.Fields {
			walk(field, path+"."+field.Name, true)
		}
		if parameter.Element != nil {
			walk(*parameter.Element, path+"[]", true)
		}
		if parameter.Value != nil {
			walk(*parameter.Value, path+".*", true)
		}
	}
	walk(root, root.Name, true)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].rank != ranked[j].rank {
			return ranked[i].rank < ranked[j].rank
		}
		return strings.ToLower(ranked[i].match.Path) < strings.ToLower(ranked[j].match.Path)
	})
	result := make([]ParameterMatch, len(ranked))
	for i := range ranked {
		result[i] = ranked[i].match
	}
	return result
}

func directParameterSearchRank(parameter Parameter, query normalizedSearchText) int {
	names := append([]string{parameter.Name, parameter.RawName}, parameter.Options...)
	if rank := bestSearchRank(query, names...); rank > 0 {
		return rank
	}
	text := parameter.Help.EN + " " + parameter.Help.ZH + " " + parameter.Example +
		" " + strings.Join(parameter.Constraints.Enum, " ") + " " +
		parameter.Constraints.Pattern + " " + parameter.Constraints.Minimum + " " +
		parameter.Constraints.Maximum + " " + parameter.Constraints.MinLength + " " +
		parameter.Constraints.MaxLength
	if textContains(text, query) {
		return 4
	}
	return 0
}

func candidateParameters(candidates []parameterCandidate, query string) []string {
	normalized := searchText(query)
	type ranked struct {
		name string
		rank int
	}
	var values []ranked
	for _, candidate := range candidates {
		rank := bestSearchRank(normalized, candidate.aliases...)
		if rank == 0 {
			rank = 9
		}
		values = append(values, ranked{name: "--" + candidate.path, rank: rank})
	}
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].rank != values[j].rank {
			return values[i].rank < values[j].rank
		}
		return values[i].name < values[j].name
	})
	if len(values) > 5 {
		values = values[:5]
	}
	result := make([]string, len(values))
	for i := range values {
		result[i] = values[i].name
	}
	return result
}

func BuildAPIResponseDocument(api *meta.API, response *ResponseDocumentation, options HelpOptions) (*APIResponseDocument, error) {
	options = options.normalized()
	if api == nil || response == nil {
		return nil, fmt.Errorf("API and response documentation are required")
	}
	command := api.CmdName
	if command == "" {
		command = kebabCase(api.Name)
	}
	document := &APIResponseDocument{
		SchemaVersion: SchemaVersion, Kind: "api", Section: SectionResponse,
		Target: Target{Product: strings.ToLower(api.ProductCode), API: command, APIVersion: api.Version},
		Query:  options.Search, Warnings: append([]string(nil), response.Warnings...),
	}
	if options.Search == "" {
		hasSchema := len(strings.TrimSpace(string(response.Schema))) > 0
		if !hasSchema {
			document.Notice = LocalizedText{
				EN: "No response body schema is available for this API.",
				ZH: "此 API 没有可用的响应正文结构。",
			}
		}
		if ((options.AIMode && !options.All) || len(strings.TrimSpace(string(response.Responses))) == 0) && hasSchema {
			document.OutputSchema = &OutputSchema{
				StatusCode: response.StatusCode, ContentType: response.ContentType,
				Schema: cloneRaw(response.Schema), Components: reachableComponentRaw(response.Schema, response.Components),
			}
		} else {
			document.Responses = cloneRaw(response.Responses)
			document.Components = cloneRawMap(response.Components)
		}
		document.Result = Result{Shown: 1, Total: 1}
	} else {
		found, err := SearchResponseSchema(ResponseSchema{
			Schema: response.Schema, Components: response.Components,
		}, options.Search, options.All)
		if err != nil {
			return nil, err
		}
		document.Matches = found.Paths
		document.Result = found.Result
		if len(found.Paths) > 0 {
			document.OutputSchema = &OutputSchema{
				StatusCode: response.StatusCode, ContentType: response.ContentType,
				Schema: found.Schema, Components: found.Components,
			}
		} else {
			document.Notice = LocalizedText{
				EN: fmt.Sprintf("No response schema fields matched %q.", options.Search),
				ZH: fmt.Sprintf("没有响应结构字段匹配 %q。", options.Search),
			}
		}
	}
	document.ResponseQuery, _ = BuildResponseQueryExample(ResponseQueryContext{
		Document: *response, Product: strings.ToLower(api.ProductCode), API: api.Name,
		APIVersion: options.RequestedVersion, RequiredFlags: requiredFlags(api.Parameters),
		PaginationCollectionPath: response.PaginationCollectionPath,
	})
	sectionOptions := options
	sectionOptions.ExplicitSection = true
	sectionOptions.Section = SectionResponse
	document.Next = buildHelpNext(
		helpHintTargetForAPI(document.Target, sectionOptions), sectionOptions, document.Result, false, nil,
	)
	return document, nil
}

func projectParameter(source meta.Parameter) Parameter {
	name := source.RawName
	if name == "" {
		name = source.Name
	}
	name = strings.TrimLeft(firstOption(source.Options, "--"+kebabCase(name)), "-")
	result := Parameter{
		Name: name, RawName: source.RawName, Options: append([]string(nil), source.Options...),
		Type: source.Type, Location: source.Position, Required: source.Required || source.DocRequired,
		Serialization: source.ParamStyle, Help: localized(source.Description), Example: source.Example,
		Constraints: Constraints{
			Enum: append([]string(nil), source.Enum...), Pattern: source.Pattern,
			Minimum: source.Minimum, Maximum: source.Maximum, MinLength: source.MinLength, MaxLength: source.MaxLength,
		},
	}
	for i := range source.Fields {
		result.Fields = append(result.Fields, projectParameter(source.Fields[i]))
	}
	if source.ItemType != nil {
		item := projectParameter(*source.ItemType)
		result.Element = &item
	}
	if source.ValueType != nil {
		value := projectParameter(*source.ValueType)
		result.Value = &value
	}
	return result
}

func compactParameters(parameters []Parameter, budget int) []Parameter {
	selected := make([]Parameter, 0, len(parameters))
	used := 0
	for _, parameter := range parameters {
		if parameter.Required {
			selected = append(selected, parameter)
			used += parameterLines(parameter)
		}
	}
	for _, parameter := range parameters {
		if parameter.Required {
			continue
		}
		lines := parameterLines(parameter)
		if used+lines > budget {
			break
		}
		selected = append(selected, parameter)
		used += lines
	}
	return selected
}

func parameterLines(parameter Parameter) int {
	lines := 1
	for _, field := range parameter.Fields {
		lines += parameterLines(field)
	}
	if parameter.Element != nil {
		lines += parameterLines(*parameter.Element)
	}
	if parameter.Value != nil {
		lines += parameterLines(*parameter.Value)
	}
	return lines
}

func firstOption(options []string, fallback string) string {
	if len(options) > 0 && options[0] != "" {
		return options[0]
	}
	return fallback
}

func requiredFlags(parameters []meta.Parameter) []string {
	const maxRequiredFlags = 4
	var result []string
	for _, parameter := range parameters {
		if len(result) >= maxRequiredFlags {
			break
		}
		if !parameter.Required && !parameter.DocRequired {
			continue
		}
		name := parameter.RawName
		if name == "" {
			name = parameter.Name
		}
		name = strings.TrimLeft(firstOption(parameter.Options, "--"+kebabCase(name)), "-")
		result = append(result, "--"+strings.TrimLeft(name, "-"))
	}
	return result
}

func shellToken(value string) string {
	if safeToken(value) {
		return value
	}
	return shellQuote(value)
}
