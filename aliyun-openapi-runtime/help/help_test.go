package help

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

func TestBuildProductDocumentProjectionPolicy(t *testing.T) {
	product := &meta.Product{
		Code: "ECS", Name: meta.Description{EN: "Compute", ZH: "云服务器"},
		Description: meta.Description{EN: "Elastic compute"}, Versions: []string{"2026-01-01"},
	}
	entries := map[string]meta.APIIndexEntry{}
	for i := 0; i < 110; i++ {
		name := fmt.Sprintf("DescribeItem%03d", i)
		entries[name] = meta.APIIndexEntry{APIName: name, CmdName: fmt.Sprintf("describe-item-%03d", i)}
	}
	index := &meta.APIIndex{ProductCode: "ecs", Version: "2026-01-01", Entries: entries}

	ordinary := BuildProductDocument(product, index, HelpOptions{})
	if ordinary.Result.Truncated || len(ordinary.APIs) != 110 {
		t.Fatalf("ordinary Help must be complete: %+v", ordinary.Result)
	}
	ai := BuildProductDocument(product, index, HelpOptions{AIMode: true})
	if !ai.Result.Truncated || len(ai.APIs) != 85 {
		t.Fatalf("AI Help must be compact: %+v", ai.Result)
	}
	searched := BuildProductDocument(product, index, HelpOptions{Search: "item"})
	if len(searched.APIs) != 20 || searched.Result.Total != 110 {
		t.Fatalf("search projection mismatch: %+v", searched.Result)
	}
	all := BuildProductDocument(product, index, HelpOptions{AIMode: true, All: true})
	if len(all.APIs) != 110 {
		t.Fatalf("all mode unexpectedly truncated: %d", len(all.APIs))
	}
}

func TestBuildActionKeepsRequiredParametersAndRequestIsComplete(t *testing.T) {
	api := &meta.API{
		Name: "CreateThing", CmdName: "create-thing", CmdFullName: "ecs create-thing",
		ProductCode: "ECS", Version: "2026-01-01", MultiVersion: true,
		Style: meta.StyleROA, Method: "POST", URL: "/things/*", HasWildcardPath: true,
		Description: meta.Description{EN: "Create one thing", ZH: "创建资源"},
	}
	api.Parameters = append(api.Parameters, meta.Parameter{
		Name: "required_id", RawName: "RequiredId", Required: true, Type: meta.TypeString,
		Options: []string{"--required-id"}, Description: meta.Description{EN: "identity"},
		Example: "id-1", MinLength: "1",
	})
	for i := 0; i < 100; i++ {
		api.Parameters = append(api.Parameters, meta.Parameter{
			Name: fmt.Sprintf("optional_%03d", i), RawName: fmt.Sprintf("Optional%03d", i),
			Type: meta.TypeString, Options: []string{fmt.Sprintf("--optional-%03d", i)},
		})
	}
	document := BuildActionDocument(&meta.Product{Code: "ecs"}, api, nil, HelpOptions{AIMode: true})
	if !document.Result.Truncated {
		t.Fatal("expected compact Action document")
	}
	if len(document.Parameters) == 0 || document.Parameters[0].Name != "required-id" || !document.Parameters[0].Required {
		t.Fatalf("required parameter was not preserved: %#v", document.Parameters)
	}
	ordinary := BuildActionDocument(&meta.Product{Code: "ecs"}, api, nil, HelpOptions{})
	if ordinary.Result.Truncated || len(ordinary.Parameters) != len(api.Parameters) {
		t.Fatalf("non-AI Action Help must be complete: %+v", ordinary.Result)
	}
	if len(ordinary.GlobalParameters) == 0 {
		t.Fatal("Action Help omitted runtime global parameters")
	}
	if ordinary.CmdFullName != "ecs create-thing" || !ordinary.MultiVersion ||
		!ordinary.Operation.HasWildcardPath {
		t.Fatalf("Action identity metadata = %#v", ordinary)
	}
	var actionOutput bytes.Buffer
	if err := Render(&actionOutput, ordinary, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(actionOutput.String(), "\n")
	typeColumn, exampleColumn, constraintsColumn := -1, -1, -1
	for _, line := range lines {
		if strings.Contains(line, "identity") {
			typeColumn = strings.Index(line, "string (required)")
		}
		if strings.Contains(line, "example: id-1") {
			exampleColumn = strings.Index(line, "example:")
		}
		if strings.Contains(line, "constraints:") && strings.Contains(line, "minLength=1") {
			constraintsColumn = strings.Index(line, "constraints:")
		}
	}
	if typeColumn < 0 || typeColumn != exampleColumn || typeColumn != constraintsColumn {
		t.Fatalf("parameter detail columns = type:%d example:%d constraints:%d\n%s",
			typeColumn, exampleColumn, constraintsColumn, actionOutput.String())
	}
	complete := BuildRequestDocument(&meta.Product{Code: "ecs"}, api, nil, HelpOptions{AIMode: true})
	if complete.Kind != "api" || len(complete.Parameters) != 101 || complete.Result.Truncated {
		t.Fatalf("explicit request Help was truncated: %+v", complete.Result)
	}
	if len(complete.GlobalParameters) == 0 {
		t.Fatal("request Help omitted runtime global parameters")
	}
	var output bytes.Buffer
	if err := Render(&output, complete, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Global Parameters:", "--cli-dry-run"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("request Help missing %q: %s", want, output.String())
		}
	}
}

func TestParameterDescriptionContinuationAlignsWithTypeColumn(t *testing.T) {
	parameter := Parameter{
		Name: "version-id", Type: meta.TypeString, Required: true,
		Help: LocalizedText{EN: "The version to which the alias points"},
	}
	var output bytes.Buffer
	renderParameterText(&output, parameter, "en", "  ", false)
	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("parameter output lines = %d\n%s", len(lines), output.String())
	}
	typeColumn := strings.Index(lines[0], "string (required)")
	continuationColumn := strings.Index(lines[1], "alias points")
	if typeColumn < 0 || continuationColumn != typeColumn {
		t.Fatalf("description continuation columns = type:%d continuation:%d\n%s",
			typeColumn, continuationColumn, output.String())
	}
}

func TestMapParameterUsesCLIRuntimeHelpShape(t *testing.T) {
	parameter := Parameter{
		Name: "additional-version-weight", Type: meta.TypeMap,
		Help:  LocalizedText{EN: "The weight of the canary release version"},
		Value: &Parameter{Type: meta.TypeFloat},
	}
	var output bytes.Buffer
	renderParameterText(&output, parameter, "en", "  ", false)
	text := output.String()
	for _, want := range []string{
		"--additional-version-weight  string,",
		"structure:  {<key>: float, ...}",
		"format:  --additional-version-weight key1=value1",
		"key2=value2 ...",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("map parameter Help missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Value:") {
		t.Fatalf("map parameter summary expanded its value shape:\n%s", text)
	}
}

func TestBuildAPIParameterDocumentPreservesKebabCompositeShape(t *testing.T) {
	api := &meta.API{
		Name: "CreateThing", CmdName: "create-thing", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{{
			Name: "tags", RawName: "Tags", Type: meta.TypeArray, Options: []string{"--tags"},
			ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{
				Name: "key", RawName: "Key", Type: meta.TypeString, Required: true,
				Options: []string{"--key"}, Description: meta.Description{EN: "Tag key"},
			}}},
		}},
	}
	document, err := BuildAPIParameterDocument(
		&meta.Product{Code: "demo"}, api, "--tags", HelpOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if document.Kind != "parameter" || document.Parameter.Name != "tags" ||
		document.Parameter.Element == nil || len(document.Parameter.Element.Fields) != 1 ||
		document.Parameter.Element.Fields[0].Name != "key" {
		t.Fatalf("parameter document = %#v", document)
	}
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Parameter: --tags") ||
		!strings.Contains(output.String(), "--key") {
		t.Fatalf("parameter Help text = %q", output.String())
	}
}

func TestProductDeprecatedAndNestedParameterCandidates(t *testing.T) {
	product := &meta.Product{Code: "demo", Versions: []string{"v1"}}
	index := &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: map[string]meta.APIIndexEntry{
		"Active": {APIName: "Active", CmdName: "active"},
		"Old":    {APIName: "Old", CmdName: "old", Deprecated: true},
	}}
	ordinary := BuildProductDocument(product, index, HelpOptions{})
	if len(ordinary.APIs) != 1 || ordinary.Result.OmittedDeprecated != 1 || ordinary.Next == nil {
		t.Fatalf("default product projection = %#v", ordinary)
	}
	all := BuildProductDocument(product, index, HelpOptions{All: true})
	if len(all.APIs) != 2 || !all.APIs[1].Deprecated {
		t.Fatalf("all product projection = %#v", all.APIs)
	}

	fields := []meta.Parameter{{
		Name: "retry_count", RawName: "RetryCount", Type: meta.TypeInteger, Options: []string{"--retry-count"},
	}}
	for i := 0; i < 25; i++ {
		fields = append(fields, meta.Parameter{
			Name: fmt.Sprintf("retry_value_%02d", i), RawName: fmt.Sprintf("RetryValue%02d", i),
			Type: meta.TypeString, Options: []string{fmt.Sprintf("--retry-value-%02d", i)},
		})
	}
	api := &meta.API{
		Name: "CreateThing", CmdName: "create-thing", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{{
			Name: "config", RawName: "Config", Type: meta.TypeObject, Options: []string{"--config"},
			Fields: fields,
		}},
	}
	nested, err := BuildAPIParameterDocument(product, api, "config", HelpOptions{Search: "retry"})
	if err != nil || len(nested.Matches) != 20 || nested.Result.Total != 26 ||
		nested.Matches[0].Path != "config.retry-count" || nested.Next == nil {
		t.Fatalf("nested parameter search = %#v, %v", nested, err)
	}
	allMatches, err := BuildAPIParameterDocument(product, api, "config", HelpOptions{Search: "retry", All: true})
	if err != nil || len(allMatches.Matches) != 26 || allMatches.Result.Truncated {
		t.Fatalf("nested parameter all search = %#v, %v", allMatches, err)
	}
	rootMatch, err := BuildAPIParameterDocument(product, api, "config", HelpOptions{Search: "config"})
	if err != nil || len(rootMatch.Matches) != 1 || rootMatch.Matches[0].Path != "config" {
		t.Fatalf("root parameter search = %#v, %v", rootMatch, err)
	}
	_, err = BuildAPIParameterDocument(product, api, "config.retry-count", HelpOptions{})
	var unknown *UnknownParameterError
	if !errors.As(err, &unknown) || len(unknown.Candidates) == 0 {
		t.Fatalf("unknown parameter error = %#v", err)
	}

	var text bytes.Buffer
	if err := Render(&text, nested, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "Matched parameter paths:") ||
		!strings.Contains(text.String(), "config.retry-count") ||
		!strings.Contains(text.String(), "Show all matches:") {
		t.Fatalf("parameter search text = %q", text.String())
	}
	text.Reset()
	if err := Render(&text, nested, HelpOptions{Format: FormatJSON}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), `"path": "config.retry-count"`) ||
		!strings.Contains(text.String(), `"searchAll":`) {
		t.Fatalf("parameter search JSON = %q", text.String())
	}
}

func TestParameterSearchWalksFieldsElementsAndValues(t *testing.T) {
	api := &meta.API{
		Name: "CreateThing", CmdName: "create-thing", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{{
			Name: "config", RawName: "Config", Type: meta.TypeObject, Options: []string{"--config"},
			Fields: []meta.Parameter{
				{Name: "needle_field", RawName: "NeedleField", Type: meta.TypeString, Options: []string{"--needle-field"}},
				{
					Name: "items", RawName: "Items", Type: meta.TypeArray, Options: []string{"--items"},
					ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{
						Name: "needle_element", RawName: "NeedleElement", Type: meta.TypeString, Options: []string{"--needle-element"},
					}}},
				},
				{
					Name: "labels", RawName: "Labels", Type: meta.TypeMap, Options: []string{"--labels"},
					ValueType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{
						Name: "needle_value", RawName: "NeedleValue", Type: meta.TypeString, Options: []string{"--needle-value"},
					}}},
				},
			},
		}},
	}
	document, err := BuildAPIParameterDocument(
		&meta.Product{Code: "demo"}, api, "config", HelpOptions{Search: "needle", All: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, match := range document.Matches {
		paths = append(paths, match.Path)
	}
	want := []string{"config.items[].needle-element", "config.labels.*.needle-value", "config.needle-field"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("nested paths = %#v, want %#v", paths, want)
	}
}

func TestSearchResponseSchemaPrunesTreeAndComponents(t *testing.T) {
	input := ResponseSchema{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"RequestId":{"type":"string"},
				"Data":{"$ref":"#/components/schemas/Data"}
			}
		}`),
		Components: map[string]json.RawMessage{
			"Data": json.RawMessage(`{
				"type":"object",
				"properties":{
					"InstanceId":{"type":"string","description_en":"instance identity","description_zh":"实例标识"},
					"Secret":{"type":"string"}
				}
			}`),
			"Unused": json.RawMessage(`{"type":"object","properties":{"Other":{"type":"string"}}}`),
		},
	}
	result, err := SearchResponseSchema(input, "instance", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Paths) != 1 || result.Paths[0] != "Data.InstanceId" {
		t.Fatalf("unexpected paths: %#v", result.Paths)
	}
	if _, ok := result.Components["Data"]; !ok {
		t.Fatal("reachable projected component missing")
	}
	if _, ok := result.Components["Unused"]; ok {
		t.Fatal("unreachable component leaked")
	}
	if strings.Contains(string(result.Components["Data"]), "Secret") {
		t.Fatalf("component was not pruned: %s", result.Components["Data"])
	}
}

func TestLocalizeRawJSONAndRenderers(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"object",
		"description":"fallback",
		"description_en":"English",
		"description_zh":"中文",
		"properties":{"Name":{"title_en":"Name","title_zh":"名称"}}
	}`)
	localized, err := LocalizeRawJSON(raw, "zh")
	if err != nil {
		t.Fatal(err)
	}
	text := string(localized)
	if !strings.Contains(text, `"description":"中文"`) || !strings.Contains(text, `"title":"名称"`) ||
		strings.Contains(text, "description_zh") {
		t.Fatalf("unexpected localization: %s", text)
	}

	document := &APIResponseDocument{
		SchemaVersion: SchemaVersion, Kind: "api", Section: SectionResponse,
		OutputSchema: &OutputSchema{StatusCode: "200", Schema: raw},
		Result:       Result{Shown: 1, Total: 1},
	}
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{Format: FormatJSON, Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"description": "中文"`) {
		t.Fatalf("JSON renderer did not localize raw schema: %s", output.String())
	}
	if !strings.Contains(output.String(), "\n  \"schemaVersion\"") {
		t.Fatalf("non-AI JSON Help must remain pretty: %q", output.String())
	}
	output.Reset()
	if err := Render(&output, document, HelpOptions{Format: FormatJSON, Language: "zh", AIMode: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "\n") != 1 ||
		!strings.Contains(output.String(), `"description":"中文"`) {
		t.Fatalf("AI JSON Help must be one compact line: %q", output.String())
	}
	output.Reset()
	if err := Render(&output, document, HelpOptions{Format: FormatText, Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "响应结构 (HTTP 200)") || !strings.Contains(output.String(), `"中文"`) {
		t.Fatalf("unexpected text output: %s", output.String())
	}
}

func TestJSONRendererEmitsOnlySelectedDocumentLanguage(t *testing.T) {
	document := &ActionDocument{
		SchemaVersion: SchemaVersion,
		Kind:          "api",
		Description:   LocalizedText{EN: "English description", ZH: "中文描述"},
		Parameters: []Parameter{{
			Name: "name",
			Help: LocalizedText{EN: "English parameter", ZH: "中文参数"},
		}},
	}
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{Format: FormatJSON, Language: "en"}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, `"description": "English description"`) ||
		!strings.Contains(text, `"help": "English parameter"`) {
		t.Fatalf("JSON renderer did not select English strings: %s", text)
	}
	if strings.Contains(text, "中文") || strings.Contains(text, `"en"`) || strings.Contains(text, `"zh"`) {
		t.Fatalf("JSON renderer retained bilingual fields: %s", text)
	}
}

func TestBuildResponseQueryExample(t *testing.T) {
	document := ResponseDocumentation{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"TotalCount":{"type":"integer"},
				"Instances":{"type":"array","items":{"type":"object","properties":{
					"InstanceId":{"type":"string"},
					"Status":{"type":"string"},
					"Description":{"type":"string"}
				}}}
			}
		}`),
		PaginationCollectionPath: "Instances",
	}
	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: document, Product: "ecs", API: "DescribeInstances", APIVersion: "2026-01-01",
		RequiredFlags: []string{"--region-id"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if example == nil {
		t.Fatal("expected query example")
	}
	if example.Path != "Instances[*].{InstanceId:InstanceId,Status:Status,Description:Description}" {
		t.Fatalf("unexpected projection: %s", example.Path)
	}
	if !strings.Contains(example.SchemaCommand, "aliyun help ecs describe-instances") ||
		!strings.Contains(example.QueryCommand, "--region-id <value> --cli-query") {
		t.Fatalf("unexpected commands: %#v", example)
	}
}

func TestBuildResponseQueryExampleUsesRPCItemNameWrapper(t *testing.T) {
	document := ResponseDocumentation{
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"Instances":{
					"type":"array",
					"itemName":"Instance",
					"items":{"type":"object","properties":{
						"InstanceId":{"type":"string"},
						"Status":{"type":"string"}
					}}
				}
			}
		}`),
		PaginationCollectionPath: "Instances",
	}

	example, err := BuildResponseQueryExample(ResponseQueryContext{
		Document: document, Product: "ecs", API: "DescribeInstances",
	})
	if err != nil {
		t.Fatal(err)
	}
	if example == nil {
		t.Fatal("expected query example")
	}
	if example.Path != "Instances.Instance[*].{InstanceId:InstanceId,Status:Status}" {
		t.Fatalf("unexpected itemName projection: %s", example.Path)
	}
}

func TestResponseProjectionIsLosslessNormallyAndCompactForAI(t *testing.T) {
	api := &meta.API{Name: "GetThing", CmdName: "get-thing", ProductCode: "demo", Version: "v1"}
	response := &ResponseDocumentation{
		Responses: json.RawMessage(`{"200":{"description":"ok","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Used"}}}},"400":{"description":"bad"}}`),
		Schema:    json.RawMessage(`{"$ref":"#/components/schemas/Used"}`),
		Components: map[string]json.RawMessage{
			"Used":   json.RawMessage(`{"type":"object","properties":{"Id":{"type":"string"}}}`),
			"Unused": json.RawMessage(`{"type":"object","properties":{"Secret":{"type":"string"}}}`),
		},
	}
	ordinary, err := BuildAPIResponseDocument(api, response, HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ordinary.Responses) == 0 || len(ordinary.Components) != 2 || ordinary.OutputSchema != nil {
		t.Fatalf("ordinary response projection is not lossless: %#v", ordinary)
	}
	ai, err := BuildAPIResponseDocument(api, response, HelpOptions{AIMode: true})
	if err != nil {
		t.Fatal(err)
	}
	if ai.OutputSchema == nil || len(ai.Responses) != 0 || len(ai.OutputSchema.Components) != 1 {
		t.Fatalf("AI response projection is not compact: %#v", ai)
	}
	if _, exists := ai.OutputSchema.Components["Unused"]; exists {
		t.Fatal("AI response projection retained an unreachable component")
	}
}

func TestResponseProjectionReportsMissingSchemaAndSearchMiss(t *testing.T) {
	api := &meta.API{Name: "DeleteThing", CmdName: "delete-thing", ProductCode: "demo", Version: "v1"}
	response := &ResponseDocumentation{
		Responses: json.RawMessage(`{"204":{"description":"deleted"}}`),
	}
	document, err := BuildAPIResponseDocument(api, response, HelpOptions{Language: "zh"})
	if err != nil {
		t.Fatal(err)
	}
	if document.OutputSchema != nil || document.Notice.ZH == "" {
		t.Fatalf("missing schema notice = %#v", document)
	}
	var text bytes.Buffer
	if err := Render(&text, document, HelpOptions{Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "没有可用的响应正文结构") {
		t.Fatalf("missing schema text = %q", text.String())
	}

	response.Schema = json.RawMessage(`{"type":"object","properties":{"RequestId":{"type":"string"}}}`)
	searched, err := BuildAPIResponseDocument(api, response, HelpOptions{Search: "does-not-exist"})
	if err != nil {
		t.Fatal(err)
	}
	if searched.OutputSchema != nil || searched.Notice.EN == "" {
		t.Fatalf("search miss notice = %#v", searched)
	}
}

type fakeSource struct {
	product *meta.Product
	index   *meta.APIIndex
	api     *meta.API
}

func (f fakeSource) EnsureProduct(string) error                         { return nil }
func (f fakeSource) LookupProduct(string) *meta.Product                 { return f.product }
func (f fakeSource) ResolveVersion(string, string) (string, error)      { return f.index.Version, nil }
func (f fakeSource) GetAPIIndex(string, string) (*meta.APIIndex, error) { return f.index, nil }
func (f fakeSource) GetAPI(string, string, string) (*meta.API, error)   { return f.api, nil }
func (f fakeSource) Provenance(string) *source.Provenance {
	return &source.Provenance{Kind: source.KindUser, PluginName: "aliyun-cli-demo", PluginVersion: "1.2.3"}
}

type fakeResponses struct{ value *ResponseDocumentation }

func (f fakeResponses) GetResponseDocumentation(string, string, string) (*ResponseDocumentation, error) {
	return f.value, nil
}

func TestServiceUsesSmallDataInterfaces(t *testing.T) {
	api := &meta.API{Name: "ListThings", CmdName: "list-things", ProductCode: "demo", Version: "v1"}
	source := fakeSource{
		product: &meta.Product{Code: "demo", Versions: []string{"v1"}, DefaultVersion: "v1"},
		index: &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: map[string]meta.APIIndexEntry{
			"ListThings": {APIName: "ListThings", CmdName: "list-things"},
		}},
		api: api,
	}
	service := Service{Data: source, Responses: fakeResponses{value: &ResponseDocumentation{
		Schema: json.RawMessage(`{"type":"object","properties":{"Items":{"type":"array","items":{"type":"string"}}}}`),
	}}}
	productDocument, err := service.BuildProduct("demo", "", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if productDocument.Provenance == nil || productDocument.Provenance.PluginName != "aliyun-cli-demo" {
		t.Fatalf("product provenance = %#v", productDocument.Provenance)
	}
	response, err := service.BuildResponse("demo", "", "ListThings", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if response.Target.API != "list-things" {
		t.Fatalf("unexpected target: %+v", response.Target)
	}
}
