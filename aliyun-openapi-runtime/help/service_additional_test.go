package help

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
	"github.com/aliyun/aliyun-openapi-runtime/source"
)

type serviceTestSource struct {
	product *meta.Product
	index   *meta.APIIndex
	api     *meta.API

	ensureErr  error
	resolveErr error
	indexErr   error
	apiErr     error

	ensuredProduct   string
	resolvedProduct  string
	requestedVersion string
	apiProduct       string
	apiVersion       string
	apiName          string
}

func (s *serviceTestSource) EnsureProduct(code string) error {
	s.ensuredProduct = code
	return s.ensureErr
}

func (s *serviceTestSource) LookupProduct(string) *meta.Product { return s.product }

func (s *serviceTestSource) ResolveVersion(product, requested string) (string, error) {
	s.resolvedProduct = product
	s.requestedVersion = requested
	if s.resolveErr != nil {
		return "", s.resolveErr
	}
	return s.index.Version, nil
}

func (s *serviceTestSource) GetAPIIndex(string, string) (*meta.APIIndex, error) {
	if s.indexErr != nil {
		return nil, s.indexErr
	}
	return s.index, nil
}

func (s *serviceTestSource) GetAPI(product, version, name string) (*meta.API, error) {
	s.apiProduct, s.apiVersion, s.apiName = product, version, name
	if s.apiErr != nil {
		return nil, s.apiErr
	}
	return s.api, nil
}

func (*serviceTestSource) Provenance(string) *source.Provenance {
	return &source.Provenance{
		Kind: source.KindUser, PluginName: "aliyun-cli-demo", PluginVersion: "2.0.0",
		APIVersion: "v2", Origin: "unit-test",
	}
}

type serviceTestResponses struct {
	document *ResponseDocumentation
	err      error
	product  string
	version  string
	api      string
}

func (s *serviceTestResponses) GetResponseDocumentation(product, version, api string) (*ResponseDocumentation, error) {
	s.product, s.version, s.api = product, version, api
	if s.err != nil {
		return nil, s.err
	}
	return s.document, nil
}

func newServiceTestFixtures() (*serviceTestSource, *serviceTestResponses) {
	api := &meta.API{
		Name: "ListThings", CmdName: "list-things", ProductCode: "Demo", Version: "v2",
		Style: meta.StyleROA, Method: "GET", Protocol: "HTTPS", URL: "/things",
		Title: meta.Description{EN: "List things", ZH: "查询资源"},
		Parameters: []meta.Parameter{{
			Name: "resource_id", RawName: "ResourceId", Options: []string{"--resource-id"},
			Type: meta.TypeString, Required: true,
		}},
	}
	data := &serviceTestSource{
		product: &meta.Product{
			Code: "demo", Name: meta.Description{EN: "Demo", ZH: "演示"},
			Versions: []string{"v1", "v2"}, DefaultVersion: "v2",
		},
		index: &meta.APIIndex{
			ProductCode: "demo", Version: "v2",
			Entries: map[string]meta.APIIndexEntry{
				"ListThings": {APIName: "ListThings", CmdName: "list-things"},
			},
		},
		api: api,
	}
	responses := &serviceTestResponses{document: &ResponseDocumentation{
		Responses:  json.RawMessage(`{"200":{"description":"OK"}}`),
		Schema:     json.RawMessage(`{"type":"object","properties":{"Items":{"type":"array","items":{"type":"string"}}}}`),
		StatusCode: "200", ContentType: "application/json",
	}}
	return data, responses
}

func TestServiceBuildsEveryHelpLevelThroughSmallInterfaces(t *testing.T) {
	data, responses := newServiceTestFixtures()
	service := Service{Data: data, Responses: responses}

	product, err := service.BuildProduct("  DEMO  ", "v1", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if product.Kind != "product" || product.Target.APIVersion != "v2" ||
		product.Provenance == nil || product.Provenance.PluginVersion != "2.0.0" {
		t.Fatalf("product document = %#v", product)
	}
	if data.ensuredProduct != "demo" || data.resolvedProduct != "demo" || data.requestedVersion != "v1" {
		t.Fatalf("product source calls = ensure:%q resolve:%q requested:%q",
			data.ensuredProduct, data.resolvedProduct, data.requestedVersion)
	}

	request, err := service.BuildAPI(" DEMO ", "v1", "list-things", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if request.Kind != "request" || request.Target.APIVersion != "v2" || request.Provenance == nil {
		t.Fatalf("compatibility request document = %#v", request)
	}
	action, err := service.BuildAction(" DEMO ", "v1", "list-things", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if action.Kind != "api" || action.Target.APIVersion != "v2" || action.Provenance == nil {
		t.Fatalf("action document = %#v", action)
	}
	if data.apiProduct != "demo" || data.apiVersion != "v2" || data.apiName != "list-things" {
		t.Fatalf("API source call = %q/%q/%q", data.apiProduct, data.apiVersion, data.apiName)
	}
	if responses.product != "demo" || responses.version != "v2" || responses.api != "ListThings" {
		t.Fatalf("response source call = %q/%q/%q", responses.product, responses.version, responses.api)
	}

	parameter, err := service.BuildParameter(" DEMO ", "v1", "list-things", "--resource-id", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if parameter.Kind != "parameter" || parameter.Parameter.RawName != "ResourceId" || parameter.Provenance == nil {
		t.Fatalf("parameter document = %#v", parameter)
	}
	response, err := service.BuildResponse(" DEMO ", "v1", "list-things", HelpOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if response.Kind != "response" || len(response.Responses) == 0 || response.Provenance == nil {
		t.Fatalf("response document = %#v", response)
	}
}

func TestServiceRejectsMissingDependenciesAndPropagatesSourceErrors(t *testing.T) {
	if _, err := (Service{}).BuildProduct("demo", "", HelpOptions{}); err == nil || !strings.Contains(err.Error(), "data source") {
		t.Fatalf("BuildProduct nil source error = %v", err)
	}
	if _, err := (Service{}).BuildParameter("demo", "", "ListThings", "id", HelpOptions{}); err == nil || !strings.Contains(err.Error(), "data source") {
		t.Fatalf("BuildParameter nil source error = %v", err)
	}
	data, _ := newServiceTestFixtures()
	if _, err := (Service{Data: data}).BuildResponse("demo", "", "ListThings", HelpOptions{}); err == nil || !strings.Contains(err.Error(), "response sources") {
		t.Fatalf("BuildResponse nil response source error = %v", err)
	}

	testErr := errors.New("source failed")
	tests := []struct {
		name  string
		setup func(*serviceTestSource, *serviceTestResponses)
		call  func(Service) error
	}{
		{
			name:  "ensure product",
			setup: func(data *serviceTestSource, _ *serviceTestResponses) { data.ensureErr = testErr },
			call: func(service Service) error {
				_, err := service.BuildProduct("demo", "", HelpOptions{})
				return err
			},
		},
		{
			name:  "resolve version",
			setup: func(data *serviceTestSource, _ *serviceTestResponses) { data.resolveErr = testErr },
			call: func(service Service) error {
				_, err := service.BuildRequest("demo", "", "ListThings", HelpOptions{})
				return err
			},
		},
		{
			name:  "load index",
			setup: func(data *serviceTestSource, _ *serviceTestResponses) { data.indexErr = testErr },
			call: func(service Service) error {
				_, err := service.BuildProduct("demo", "", HelpOptions{})
				return err
			},
		},
		{
			name:  "load API",
			setup: func(data *serviceTestSource, _ *serviceTestResponses) { data.apiErr = testErr },
			call: func(service Service) error {
				_, err := service.BuildAction("demo", "", "ListThings", HelpOptions{})
				return err
			},
		},
		{
			name:  "load response",
			setup: func(_ *serviceTestSource, responses *serviceTestResponses) { responses.err = testErr },
			call: func(service Service) error {
				_, err := service.BuildResponse("demo", "", "ListThings", HelpOptions{})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, responses := newServiceTestFixtures()
			test.setup(data, responses)
			err := test.call(Service{Data: data, Responses: responses})
			if !errors.Is(err, testErr) {
				t.Fatalf("error = %v, want %v", err, testErr)
			}
		})
	}

	missingProduct, responses := newServiceTestFixtures()
	missingProduct.product = nil
	if _, err := (Service{Data: missingProduct, Responses: responses}).BuildProduct("demo", "", HelpOptions{}); err == nil || !strings.Contains(err.Error(), "unavailable after loading") {
		t.Fatalf("missing product error = %v", err)
	}
}

func TestRequestSearchCoversParametersNestedFieldsAndRuntimeGlobals(t *testing.T) {
	api := &meta.API{
		Name: "RunThing", CmdName: "run-thing", ProductCode: "demo", Version: "v1",
		Parameters: []meta.Parameter{
			{
				Name: "config", RawName: "Config", Type: meta.TypeObject,
				Fields: []meta.Parameter{{
					Name: "retry_count", RawName: "RetryCount", Type: meta.TypeInteger,
					Description: meta.Description{EN: "Number of retry attempts"},
				}},
			},
			{
				Name: "state", RawName: "State", Type: meta.TypeString,
				Enum: []string{"Running", "Stopped"},
			},
		},
	}
	product := &meta.Product{Code: "demo", Versions: []string{"v1"}}

	nested := BuildRequestDocument(product, api, nil, HelpOptions{Search: "retry count"})
	if len(nested.Parameters) != 1 || nested.Parameters[0].Name != "config" || len(nested.GlobalParameters) != 0 {
		t.Fatalf("nested parameter search = %#v", nested)
	}
	enum := BuildRequestDocument(product, api, nil, HelpOptions{Search: "stopped"})
	if len(enum.Parameters) != 1 || enum.Parameters[0].Name != "state" {
		t.Fatalf("enum parameter search = %#v", enum.Parameters)
	}
	global := BuildRequestDocument(product, api, nil, HelpOptions{Search: "region"})
	if len(global.Parameters) != 0 || len(global.GlobalParameters) == 0 {
		t.Fatalf("global parameter search = params:%#v globals:%#v", global.Parameters, global.GlobalParameters)
	}
	foundRegion := false
	for _, parameter := range global.GlobalParameters {
		if parameter.Name == "--region" {
			foundRegion = true
		}
	}
	if !foundRegion {
		t.Fatalf("global parameter search omitted --region: %#v", global.GlobalParameters)
	}
}

func TestProductTextRendererLocalizesAndMarksDeprecatedAPIs(t *testing.T) {
	product := &meta.Product{
		Code: "demo", Name: meta.Description{EN: "Demo", ZH: "演示产品"},
		Description: meta.Description{EN: "Demo service", ZH: "演示服务"}, Versions: []string{"v1"},
	}
	index := &meta.APIIndex{ProductCode: "demo", Version: "v1", Entries: map[string]meta.APIIndexEntry{
		"OldAPI": {
			APIName: "OldAPI", CmdName: "old-api", Deprecated: true,
			Title: meta.Description{EN: "Old API", ZH: "旧接口"},
		},
	}}
	document := BuildProductDocument(product, index, HelpOptions{All: true})
	document.Provenance = &MetadataProvenance{Kind: "user", PluginName: "aliyun-cli-demo", PluginVersion: "2.0.0"}

	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"产品: demo (演示产品)", "描述: 演示服务", "[已废弃] 旧接口"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("product text missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "元数据来源") || strings.Contains(output.String(), "aliyun-cli-demo") {
		t.Fatalf("product Text Help leaked provenance:\n%s", output.String())
	}
}

func TestRenderRejectsInvalidOutputsAndDocuments(t *testing.T) {
	documentTests := []struct {
		name     string
		document any
		want     string
	}{
		{name: "nil product", document: (*ProductDocument)(nil), want: "product Help document is nil"},
		{name: "nil action", document: (*ActionDocument)(nil), want: "action Help document is nil"},
		{name: "nil request", document: (*RequestDocument)(nil), want: "request Help document is nil"},
		{name: "nil parameter", document: (*APIParameterDocument)(nil), want: "parameter Help document is nil"},
		{name: "nil response", document: (*APIResponseDocument)(nil), want: "response Help document is nil"},
		{name: "unsupported", document: struct{}{}, want: "unsupported Runtime Help"},
	}
	for _, test := range documentTests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := Render(&output, test.document, HelpOptions{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}

	if err := Render(nil, &ProductDocument{}, HelpOptions{}); err == nil || !strings.Contains(err.Error(), "output is nil") {
		t.Fatalf("text nil output error = %v", err)
	}
	if err := Render(nil, &ProductDocument{}, HelpOptions{Format: FormatJSON}); err == nil || !strings.Contains(err.Error(), "output is nil") {
		t.Fatalf("JSON nil output error = %v", err)
	}
	var output bytes.Buffer
	if err := Render(&output, make(chan int), HelpOptions{Format: FormatJSON}); err == nil {
		t.Fatal("JSON renderer accepted a non-serializable document")
	}
}

func TestUnknownParameterErrorIncludesCandidates(t *testing.T) {
	err := (&UnknownParameterError{Parameter: "instnace-id", Candidates: []string{"instance-id"}}).Error()
	for _, want := range []string{"instnace-id", "instance-id"} {
		if !strings.Contains(err, want) {
			t.Fatalf("UnknownParameterError %q does not include %q", err, want)
		}
	}
	if got := (&UnknownParameterError{API: "ListThings", Parameter: "unknown"}).Error(); got != "unknown parameter --unknown for ListThings" {
		t.Fatalf("UnknownParameterError without candidates = %q", got)
	}
}

func TestDocumentBuildersCoverFallbackIdentityAndRPCProjection(t *testing.T) {
	index := &meta.APIIndex{ProductCode: "dns", Version: "v1"}
	if BuildProductDocument(nil, index, HelpOptions{}) != nil {
		t.Fatal("BuildProductDocument accepted a nil product")
	}
	if BuildProductDocument(&meta.Product{Code: "dns"}, nil, HelpOptions{}) != nil {
		t.Fatal("BuildProductDocument accepted a nil API index")
	}
	if BuildRequestDocument(nil, nil, nil, HelpOptions{}) != nil {
		t.Fatal("BuildRequestDocument accepted a nil API")
	}
	if BuildActionDocument(nil, nil, nil, HelpOptions{}) != nil {
		t.Fatal("BuildActionDocument accepted a nil API")
	}

	api := &meta.API{
		Name: "DescribeDNSRecords", ProductCode: "DNS", Version: "v1",
		Style: meta.StyleRPC, Method: "POST", Protocol: "HTTPS", URL: "/rpc",
		ReqBodyType: "json", ContentType: "application/json", IsSSE: true, HasWildcardPath: true,
		Description: meta.Description{EN: "Describe DNS records"},
		Examples:    []string{"aliyun dns describe-dns-records"},
	}
	request := BuildAPIRequestDocument(nil, api, nil, HelpOptions{})
	if request.Command != "describe-dns-records" || request.CmdFullName != "dns describe-dns-records" ||
		request.Target.Product != "dns" || request.Examples.Kebab == "" {
		t.Fatalf("request fallback identity = %#v", request)
	}
	if request.Operation.Method != "POST" || request.Operation.Protocol != "HTTPS" || !request.Operation.IsSSE {
		t.Fatalf("request RPC wire metadata = %#v", request.Operation)
	}

	action := BuildActionDocument(nil, api, nil, HelpOptions{})
	if action.Description.EN != "Describe DNS records" {
		t.Fatalf("action description = %#v", action.Description)
	}
	if action.Operation.Method != "" || action.Operation.Protocol != "" || action.Operation.URL != "" ||
		action.Operation.RequestBodyType != "" || action.Operation.ContentType != "" ||
		action.Operation.IsSSE || action.Operation.HasWildcardPath {
		t.Fatalf("RPC action leaked ROA-only wire metadata: %#v", action.Operation)
	}
}

func TestActionSearchReportsTheFullMatchCount(t *testing.T) {
	api := &meta.API{Name: "RunThing", CmdName: "run-thing", ProductCode: "demo", Version: "v1"}
	for i := 0; i < 25; i++ {
		api.Parameters = append(api.Parameters, meta.Parameter{
			Name:    fmt.Sprintf("coverage_probe_%02d", i),
			Options: []string{fmt.Sprintf("--coverage-probe-%02d", i)},
			Type:    meta.TypeString,
		})
	}
	product := &meta.Product{Code: "demo", Versions: []string{"v1"}}
	limited := BuildActionDocument(product, api, nil, HelpOptions{Search: "coverage probe"})
	if len(limited.Parameters) != 20 || limited.Result.Total != 25 || !limited.Result.Truncated ||
		limited.Next == nil || limited.Next.SearchAll == "" {
		t.Fatalf("limited action search = %#v", limited)
	}
	unlimited := BuildActionDocument(product, api, nil, HelpOptions{Search: "coverage probe", All: true})
	if len(unlimited.Parameters) != 25 || unlimited.Result.Total != 25 || unlimited.Result.Truncated || unlimited.Next != nil {
		t.Fatalf("unlimited action search = %#v", unlimited)
	}
}

func TestResponseTextRendererCoversSchemaAndResponsesComponents(t *testing.T) {
	schemaDocument := &APIResponseDocument{
		Warnings: []string{"schema warning"},
		Matches:  []string{"Items.Name"},
		OutputSchema: &OutputSchema{
			StatusCode: "200",
			Schema: json.RawMessage(`{
				"type":"object",
				"description_en":"English schema",
				"description_zh":"中文结构"
			}`),
			Components: map[string]json.RawMessage{
				"Thing": json.RawMessage(`{
					"type":"object",
					"description_en":"English component",
					"description_zh":"中文组件"
				}`),
			},
		},
		ResponseQuery: &QueryExample{QueryCommand: "aliyun demo list-things --cli-query Items"},
		Result:        Result{Shown: 1, Total: 2, Truncated: true},
		Next:          &Next{ShowAll: "aliyun demo list-things --help-all"},
	}
	var output bytes.Buffer
	if err := Render(&output, schemaDocument, HelpOptions{Language: "zh"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"警告:", "schema warning", "匹配的响应路径:", "Items.Name",
		"响应结构 (HTTP 200):", "中文结构", "组件:", "中文组件",
		"使用 JMESPath 查询:", "显示全部: aliyun demo list-things --help-all",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("response schema text missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "English schema") || strings.Contains(output.String(), "English component") {
		t.Fatalf("response schema text mixed languages:\n%s", output.String())
	}

	responsesDocument := &APIResponseDocument{
		Responses: json.RawMessage(`{
			"200":{"description_en":"Success","description_zh":"成功"}
		}`),
		Components: map[string]json.RawMessage{
			"Error": json.RawMessage(`{
				"type":"object",
				"title_en":"Error response",
				"title_zh":"错误响应"
			}`),
		},
	}
	output.Reset()
	if err := Render(&output, responsesDocument, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Responses:", "Success", "Components:", "Error response"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("responses text missing %q:\n%s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "成功") || strings.Contains(output.String(), "错误响应") {
		t.Fatalf("responses text mixed languages:\n%s", output.String())
	}
}

func TestResponseTextRendererPropagatesMalformedJSON(t *testing.T) {
	tests := []struct {
		name     string
		document *APIResponseDocument
	}{
		{
			name: "output schema",
			document: &APIResponseDocument{OutputSchema: &OutputSchema{
				Schema: json.RawMessage(`{`),
			}},
		},
		{
			name: "output component",
			document: &APIResponseDocument{OutputSchema: &OutputSchema{
				Schema:     json.RawMessage(`{}`),
				Components: map[string]json.RawMessage{"Broken": json.RawMessage(`{`)},
			}},
		},
		{
			name:     "responses",
			document: &APIResponseDocument{Responses: json.RawMessage(`{`)},
		},
		{
			name: "response component",
			document: &APIResponseDocument{
				Responses:  json.RawMessage(`{}`),
				Components: map[string]json.RawMessage{"Broken": json.RawMessage(`{`)},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := Render(&output, test.document, HelpOptions{}); err == nil {
				t.Fatalf("Render accepted malformed JSON: %s", output.String())
			}
		})
	}

	var output bytes.Buffer
	if err := writePrettyJSON(&output, nil); err != nil || output.String() != "{}\n" {
		t.Fatalf("empty pretty JSON = %q, %v", output.String(), err)
	}
	localized, err := LocalizeRawJSONMap(nil, "en")
	if err != nil || localized != nil {
		t.Fatalf("nil localized component map = %#v, %v", localized, err)
	}
}
