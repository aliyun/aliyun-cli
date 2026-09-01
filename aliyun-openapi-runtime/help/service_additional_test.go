package help

import (
	"bytes"
	"encoding/json"
	"errors"
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
	for _, want := range []string{"产品: demo (演示产品)", "描述: 演示服务", "元数据来源: aliyun-cli-demo (2.0.0)", "[已废弃] 旧接口"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("product text missing %q:\n%s", want, output.String())
		}
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
