package canonicalmeta

import (
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func testFS() *Repository {
	return NewRepository(os.DirFS("testdata"))
}

// ── Reader tests ──

func TestReadAPI_DescribeRegions(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	if api.Name != "DescribeRegions" {
		t.Errorf("expected Name=DescribeRegions, got %s", api.Name)
	}
	if api.Protocol != "HTTP|HTTPS" {
		t.Errorf("expected Protocol=HTTP|HTTPS, got %s", api.Protocol)
	}
	if api.Method != "GET|POST" {
		t.Errorf("expected Method=GET|POST, got %s", api.Method)
	}
	if api.PathPattern != "" {
		t.Errorf("expected empty PathPattern, got %s", api.PathPattern)
	}
	if api.DescriptionZh != "查询地域列表。" {
		t.Errorf("expected DescriptionZh=查询地域列表。, got %s", api.DescriptionZh)
	}
	if api.DescriptionEn != "Queries regions." {
		t.Errorf("expected DescriptionEn=Queries regions., got %s", api.DescriptionEn)
	}
	if len(api.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(api.Parameters))
	}
	if api.V1BodyParameters != nil {
		t.Error("expected V1BodyParameters to be nil")
	}
}

func TestReadAPI_CreateReport(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "CreateReport")
	if err != nil {
		t.Fatal(err)
	}

	if api.Name != "CreateReport" {
		t.Errorf("expected Name=CreateReport, got %s", api.Name)
	}
	if len(api.Parameters) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(api.Parameters))
	}
	if api.V1BodyParameters == nil {
		t.Fatal("expected V1BodyParameters to be non-nil")
	}
	if len(*api.V1BodyParameters) != 1 {
		t.Errorf("expected 1 v1_body_parameter, got %d", len(*api.V1BodyParameters))
	}
	bodyParam := (*api.V1BodyParameters)[0]
	if bodyParam.Name != "body" {
		t.Errorf("expected body param name=body, got %s", bodyParam.Name)
	}
}

func TestCanonicalAPITestdataOmitsEndpointMaps(t *testing.T) {
	data, err := os.ReadFile("testdata/canonical/demo/2026-01-01/CreateReport.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"endpoint_map", "vpc_endpoint_map", "global_endpoint"} {
		if _, ok := raw[key]; ok {
			t.Fatalf("API JSON testdata must not include %q; endpoint data belongs in products.json", key)
		}
	}
}

func TestReadAPI_UsesSpecPathBeforeNestedCompatibilityPath(t *testing.T) {
	repo := NewRepository(fstest.MapFS{
		"canonical/demo/2026-01-01/Describe.json": &fstest.MapFile{
			Data: []byte(`{"name":"SpecPath","protocol":"HTTP","method":"GET","pathPattern":"","parameters":[]}`),
			Mode: fs.ModePerm,
		},
		"canonical/demo/canonical/demo/2026-01-01/Describe.json": &fstest.MapFile{
			Data: []byte(`{"name":"NestedPath","protocol":"HTTP","method":"GET","pathPattern":"","parameters":[]}`),
			Mode: fs.ModePerm,
		},
	})

	api, err := repo.GetAPI("Demo", "2026-01-01", "Describe")
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "SpecPath" {
		t.Fatalf("expected spec path to win, got %s", api.Name)
	}
}

func TestReadAPI_CurrentNestedCompatibilityPath(t *testing.T) {
	repo := NewRepository(fstest.MapFS{
		"canonical/demo/canonical/demo/2026-01-01/Describe.json": &fstest.MapFile{
			Data: []byte(`{"name":"NestedPath","protocol":"HTTP","method":"GET","pathPattern":"","parameters":[]}`),
			Mode: fs.ModePerm,
		},
	})

	api, err := repo.GetAPI("Demo", "2026-01-01", "Describe")
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "NestedPath" {
		t.Fatalf("expected nested compatibility path, got %s", api.Name)
	}
}

func TestReadAPI_DoesNotFallbackFromInvalidSpecPath(t *testing.T) {
	repo := NewRepository(fstest.MapFS{
		"canonical/demo/2026-01-01/Describe.json": &fstest.MapFile{
			Data: []byte(`{"name":"SpecPath","protocol":"HTTP","method":"GET","pathPattern":"","parameters":[{"name":"bad","raw_name":"Bad","type":"string","location":"bad"}]}`),
			Mode: fs.ModePerm,
		},
		"canonical/demo/canonical/demo/2026-01-01/Describe.json": &fstest.MapFile{
			Data: []byte(`{"name":"NestedPath","protocol":"HTTP","method":"GET","pathPattern":"","parameters":[]}`),
			Mode: fs.ModePerm,
		},
	})

	_, err := repo.GetAPI("Demo", "2026-01-01", "Describe")
	if err == nil {
		t.Fatal("expected invalid spec path to fail")
	}
	if got := err.Error(); !strings.Contains(got, "unknown canonical location") {
		t.Fatalf("expected invalid spec path error, got %s", got)
	}
}

func TestReadAPI_RejectsUnknownLocation(t *testing.T) {
	repo := NewRepository(fstest.MapFS{
		"canonical/demo/2026-01-01/BadLocation.json": &fstest.MapFile{
			Data: []byte(`{
				"name": "BadLocation",
				"protocol": "HTTP",
				"method": "GET",
				"pathPattern": "",
				"parameters": [
					{"name": "foo", "raw_name": "Foo", "type": "string", "location": "somewhere"}
				]
			}`),
			Mode: fs.ModePerm,
		},
	})

	_, err := repo.GetAPI("demo", "2026-01-01", "BadLocation")
	if err == nil {
		t.Fatal("expected unknown location error")
	}
	if got := err.Error(); !strings.Contains(got, "unknown canonical location") {
		t.Fatalf("expected unknown location error, got %s", got)
	}
}

func TestGetAPIByPathUsesProvidedAPINames(t *testing.T) {
	repo := NewRepository(fstest.MapFS{
		"canonical/demo/2026-01-01/Listed.json": &fstest.MapFile{
			Data: []byte(`{"name":"Listed","protocol":"HTTP","method":"GET","pathPattern":"/items/[id]","parameters":[]}`),
			Mode: fs.ModePerm,
		},
		"canonical/demo/2026-01-01/Unlisted.json": &fstest.MapFile{
			Data: []byte(`{"name":"Unlisted","protocol":"HTTP","method":"GET","pathPattern":"/items/[id]","parameters":[]}`),
			Mode: fs.ModePerm,
		},
	})

	api, err := repo.GetAPIByPath("demo", "2026-01-01", "GET", "/items/123", []string{"Listed"})
	if err != nil {
		t.Fatal(err)
	}
	if api.Name != "Listed" {
		t.Fatalf("expected Listed, got %s", api.Name)
	}
}

// ── Legacy View: API-level methods ──

func TestGetMethod(t *testing.T) {
	api := &API{Method: "GET|POST"}
	if api.GetMethod() != "POST" {
		t.Errorf("expected POST, got %s", api.GetMethod())
	}

	api = &API{Method: "GET"}
	if api.GetMethod() != "GET" {
		t.Errorf("expected GET, got %s", api.GetMethod())
	}
}

func TestGetProtocol(t *testing.T) {
	api := &API{Protocol: "HTTP|HTTPS"}
	if api.GetProtocol() != "https" {
		t.Errorf("expected https, got %s", api.GetProtocol())
	}

	api = &API{Protocol: "HTTP"}
	if api.GetProtocol() != "http" {
		t.Errorf("expected http, got %s", api.GetProtocol())
	}
}

// ── Legacy View: parameter methods ──

func TestLegacyParameterView_Name(t *testing.T) {
	p := &Parameter{RawName: "AcceptLanguage"}
	v := NewCanonicalView(p)
	if v.LegacyName() != "AcceptLanguage" {
		t.Errorf("expected AcceptLanguage, got %s", v.LegacyName())
	}
}

func TestLegacyParameterView_Position(t *testing.T) {
	tests := []struct {
		location string
		expected string
	}{
		{"query", "Query"},
		{"body", "Body"},
		{"host", "Host"},
		{"path", "Path"},
		{"header", "Header"},
		{"form", "Body"},
		{"formdata", "Body"},
	}
	for _, tt := range tests {
		p := &Parameter{Location: tt.location}
		v := NewCanonicalView(p)
		if v.LegacyPosition() != tt.expected {
			t.Errorf("location=%s: expected %s, got %s", tt.location, tt.expected, v.LegacyPosition())
		}
	}
}

func TestLegacyParameterView_V1DescriptionKeepsDescription(t *testing.T) {
	p := &LegacyBodyParameter{
		Name:          "LegacyParam",
		DescriptionZh: "中文说明",
		DescriptionEn: "English description",
	}
	v := NewV1View(p)

	if v.LegacyDescription("zh") != "中文说明" {
		t.Errorf("expected 中文说明, got %s", v.LegacyDescription("zh"))
	}
	if v.LegacyDescription("en") != "English description" {
		t.Errorf("expected English description, got %s", v.LegacyDescription("en"))
	}

	zhOnly := NewV1View(&LegacyBodyParameter{DescriptionZh: "仅中文"})
	if zhOnly.LegacyDescription("en") != "" {
		t.Errorf("expected no cross-language fallback for English, got %q", zhOnly.LegacyDescription("en"))
	}

	enOnly := NewV1View(&LegacyBodyParameter{DescriptionEn: "English only"})
	if enOnly.LegacyDescription("zh") != "" {
		t.Errorf("expected no cross-language fallback for Chinese, got %q", enOnly.LegacyDescription("zh"))
	}
}

func TestLegacyParameterView_V1HelpPreferredOverDescription(t *testing.T) {
	p := &LegacyBodyParameter{
		Name:          "LegacyParam",
		DescriptionZh: "中文说明",
		DescriptionEn: "English description",
		HelpZh:        "中文帮助",
		HelpEn:        "English help",
	}
	v := NewV1View(p)

	if v.LegacyDescription("zh") != "中文帮助" {
		t.Errorf("expected 中文帮助, got %s", v.LegacyDescription("zh"))
	}
	if v.LegacyDescription("en") != "English help" {
		t.Errorf("expected English help, got %s", v.LegacyDescription("en"))
	}

	bodyOnlyHelp := NewBodyView(&LegacyBodyParameter{HelpZh: "body 帮助"})
	if bodyOnlyHelp.LegacyDescription("zh") != "body 帮助" {
		t.Errorf("expected body 帮助, got %s", bodyOnlyHelp.LegacyDescription("zh"))
	}
	if bodyOnlyHelp.LegacyDescription("en") != "" {
		t.Errorf("expected no cross-language fallback for English, got %q", bodyOnlyHelp.LegacyDescription("en"))
	}
}

func TestLegacyParameterView_HelpOnlyForCanonicalParametersAndFields(t *testing.T) {
	// help_* is the CLI-facing parameter text for canonical parameters/fields.
	p := &Parameter{
		DescriptionZh: "中文说明",
		DescriptionEn: "English description",
		HelpZh:        "中文帮助",
		HelpEn:        "English help",
	}
	v := NewCanonicalView(p)
	if v.LegacyDescription("zh") != "中文帮助" {
		t.Errorf("expected 中文帮助, got %s", v.LegacyDescription("zh"))
	}
	if v.LegacyDescription("en") != "English help" {
		t.Errorf("expected English help, got %s", v.LegacyDescription("en"))
	}

	noHelp := NewCanonicalView(&Parameter{DescriptionZh: "中文说明", DescriptionEn: "English description"})
	if noHelp.LegacyDescription("zh") != "" {
		t.Errorf("expected empty zh help without help_zh, got %q", noHelp.LegacyDescription("zh"))
	}
	if noHelp.LegacyDescription("en") != "" {
		t.Errorf("expected empty en help without help_en, got %q", noHelp.LegacyDescription("en"))
	}

	// No cross-language mixing and no description fallback.
	zhHelpOnly := NewCanonicalView(&Parameter{HelpZh: "仅中文帮助", DescriptionEn: "English description"})
	if zhHelpOnly.LegacyDescription("en") != "" {
		t.Errorf("expected empty en help without help_en, got %q", zhHelpOnly.LegacyDescription("en"))
	}

	// Field views follow the same preference.
	f := &Field{DescriptionZh: "字段说明", DescriptionEn: "Field description", HelpZh: "字段帮助"}
	fv := NewFieldView(f, "body")
	if fv.LegacyDescription("zh") != "字段帮助" {
		t.Errorf("expected 字段帮助, got %s", fv.LegacyDescription("zh"))
	}
	if fv.LegacyDescription("en") != "" {
		t.Errorf("expected empty en help without help_en, got %q", fv.LegacyDescription("en"))
	}
}

func TestAPI_DescriptionKeepsTopLevelDescription(t *testing.T) {
	api := &API{DescriptionZh: "API 中文说明", DescriptionEn: "API English description"}
	if api.Description("zh") != "API 中文说明" {
		t.Errorf("expected API 中文说明, got %q", api.Description("zh"))
	}
	if api.Description("en") != "API English description" {
		t.Errorf("expected API English description, got %q", api.Description("en"))
	}
}

func TestLegacyParameterView_RepeatList(t *testing.T) {
	// Array with repeatList style
	p := &Parameter{Type: "array", ParamStyle: "repeatList"}
	v := NewCanonicalView(p)
	if !v.IsLegacyRepeatList() {
		t.Error("expected IsLegacyRepeatList=true for array+repeatList")
	}

	// Array with flat style
	p = &Parameter{Type: "array", ParamStyle: "flat"}
	v = NewCanonicalView(p)
	if v.IsLegacyRepeatList() {
		t.Error("expected IsLegacyRepeatList=false for array+flat")
	}

	// Array with json style
	p = &Parameter{Type: "array", ParamStyle: "json"}
	v = NewCanonicalView(p)
	if v.IsLegacyRepeatList() {
		t.Error("expected IsLegacyRepeatList=false for array+json")
	}

	// Array with no style
	p = &Parameter{Type: "array"}
	v = NewCanonicalView(p)
	if !v.IsLegacyRepeatList() {
		t.Error("expected IsLegacyRepeatList=true for array+no style")
	}
}

func TestLegacyParameterView_HasChildren(t *testing.T) {
	// Array with element_fields
	p := &Parameter{
		Type:       "array",
		ParamStyle: "repeatList",
		ElementFields: []Field{
			{RawName: "Key", Type: "string"},
		},
	}
	v := NewCanonicalView(p)
	if !v.LegacyHasChildren() {
		t.Error("expected LegacyHasChildren=true")
	}

	children := v.LegacyChildren()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].LegacyName() != "Key" {
		t.Errorf("expected child name=Key, got %s", children[0].LegacyName())
	}

	arrayChild := children[0]
	arrayChild.field.Type = "array"
	if !arrayChild.IsLegacyRepeatList() {
		t.Error("expected array field child to behave as legacy RepeatList")
	}
}

// ── Legacy View: API-level parameter methods ──

func TestLegacyTopLevelParameters_NoV1Body(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	params := api.LegacyTopLevelParameters()
	if len(params) != 2 {
		t.Errorf("expected 2 top-level params, got %d", len(params))
	}

	// Should be sorted by name
	if params[0].LegacyName() != "AcceptLanguage" {
		t.Errorf("expected first param=AcceptLanguage, got %s", params[0].LegacyName())
	}
	if params[1].LegacyName() != "Tags" {
		t.Errorf("expected second param=Tags, got %s", params[1].LegacyName())
	}
}

func TestLegacyTopLevelParameters_WithV1Body(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "CreateReport")
	if err != nil {
		t.Fatal(err)
	}

	params := api.LegacyTopLevelParameters()

	// Should have ReportId (path) + body (from v1_body_parameters)
	// WorkspaceId and ReportName should be removed (body location, v1_body_parameters present)
	found := make(map[string]bool)
	for _, p := range params {
		found[p.LegacyName()] = true
	}

	if !found["ReportId"] {
		t.Error("expected ReportId in top-level params")
	}
	if !found["body"] {
		t.Error("expected body (from v1_body_parameters) in top-level params")
	}
	if found["WorkspaceId"] {
		t.Error("WorkspaceId should be excluded when v1_body_parameters is present")
	}
	if found["ReportName"] {
		t.Error("ReportName should be excluded when v1_body_parameters is present")
	}
}

func TestLegacyTopLevelParameters_V1ParametersOverrideAllOtherSources(t *testing.T) {
	var api API
	err := json.Unmarshal([]byte(`{
		"name": "CreateThing",
		"protocol": "HTTP|HTTPS",
		"method": "POST",
		"pathPattern": "",
		"parameters": [
			{"name": "canonical_only", "raw_name": "CanonicalOnly", "type": "string", "required": true, "location": "query"},
			{"name": "body_field", "raw_name": "BodyField", "type": "string", "required": true, "location": "body"}
		],
		"v1_body_parameters": [
			{"name": "body", "position": "body", "type": "string", "required": true}
		],
		"v1_parameters": [
			{"name": "Action", "position": "query", "type": "string", "required": true},
			{"name": "ServiceHost", "position": "domain", "type": "string", "required": false},
			{"name": "OnlyV1", "position": "query", "type": "string", "required": true},
			{"name": "Payload", "position": "body", "type": "string", "required": false},
			{
				"name": "Tag",
				"position": "query",
				"type": "array",
				"param_style": "repeatList",
				"required": false,
				"sub_parameters": [
					{"name": "Key", "position": "query", "type": "string", "required": false}
				]
			}
		]
	}`), &api)
	if err != nil {
		t.Fatal(err)
	}
	if api.V1Parameters == nil {
		t.Fatal("expected v1_parameters to be deserialized")
	}
	if err := validateAPI(&api); err != nil {
		t.Fatal(err)
	}

	params := api.LegacyTopLevelParameters()
	names := map[string]*LegacyParameterView{}
	for _, p := range params {
		names[p.LegacyName()] = p
	}

	if len(names) != 4 {
		t.Fatalf("expected only v1_parameters to be exposed, got %d: %#v", len(names), names)
	}
	if _, ok := names["OnlyV1"]; !ok {
		t.Fatal("expected OnlyV1 from v1_parameters")
	}
	if names["OnlyV1"].LegacyPosition() != "Query" {
		t.Fatalf("expected OnlyV1 position Query, got %s", names["OnlyV1"].LegacyPosition())
	}
	if names["Payload"].LegacyPosition() != "Body" {
		t.Fatalf("expected Payload position Body, got %s", names["Payload"].LegacyPosition())
	}
	if names["ServiceHost"].LegacyPosition() != "Domain" {
		t.Fatalf("expected ServiceHost position Domain, got %s", names["ServiceHost"].LegacyPosition())
	}
	if _, ok := names["CanonicalOnly"]; ok {
		t.Fatal("CanonicalOnly from parameters must not be merged when v1_parameters exists")
	}
	if _, ok := names["body"]; ok {
		t.Fatal("body from v1_body_parameters must not be merged when v1_parameters exists")
	}
	if _, ok := names["Action"]; ok {
		t.Fatal("protocol-level Action must not be exposed when v1_parameters exists")
	}
	if len(api.LegacyBodyFields()) != 0 {
		t.Fatal("LegacyBodyFields must not expose canonical body fields when v1_parameters exists")
	}

	if v := api.FindLegacyParameter("Tag.1.Key"); v == nil || v.LegacyName() != "Key" {
		t.Fatalf("expected v1_parameters repeatList child lookup to find Key, got %#v", v)
	}

	err = api.CheckLegacyRequiredParameters(func(name string) bool {
		return name != "OnlyV1"
	})
	if err == nil || !strings.Contains(err.Error(), "--OnlyV1") {
		t.Fatalf("expected OnlyV1 to be the only missing required parameter, got %v", err)
	}
	if strings.Contains(err.Error(), "--CanonicalOnly") || strings.Contains(err.Error(), "--body") || strings.Contains(err.Error(), "--Action") {
		t.Fatalf("required check must not include parameters or v1_body_parameters when v1_parameters exists, got %v", err)
	}
}

func TestFindLegacyParameter_Exact(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	v := api.FindLegacyParameter("AcceptLanguage")
	if v == nil {
		t.Fatal("expected to find AcceptLanguage")
	}
	if v.LegacyName() != "AcceptLanguage" {
		t.Errorf("expected AcceptLanguage, got %s", v.LegacyName())
	}
}

func TestFindLegacyParameter_RepeatListWithIndex(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	// Tags has children; old FindParameter only accepts Tags.1.Key-style child
	// access, not the bare Tags.1 parent.
	v := api.FindLegacyParameter("Tags.1")
	if v != nil {
		t.Fatalf("expected Tags.1 to be invalid for array with children, got %s", v.LegacyName())
	}
}

func TestFindLegacyParameter_SubField(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	// Tags.1.Key should match Key sub-field
	v := api.FindLegacyParameter("Tags.1.Key")
	if v == nil {
		t.Fatal("expected to find Tags.1.Key")
	}
	if v.LegacyName() != "Key" {
		t.Errorf("expected Key, got %s", v.LegacyName())
	}
}

func TestFindLegacyParameter_SubFieldLooseIndex(t *testing.T) {
	api := &API{
		Parameters: []Parameter{
			{
				RawName:    "Tag",
				Type:       "array",
				ParamStyle: "repeatList",
				ElementFields: []Field{
					{RawName: "Key", Type: "string"},
				},
			},
		},
	}

	if v := api.FindLegacyParameter("Tag.foo.Key"); v == nil || v.LegacyName() != "Key" {
		t.Fatalf("expected Tag.foo.Key to match Key, got %#v", v)
	}
	if v := api.FindLegacyParameter("Tag..Key"); v == nil || v.LegacyName() != "Key" {
		t.Fatalf("expected Tag..Key to match Key, got %#v", v)
	}
	if v := api.FindLegacyParameter("Tag.Key"); v != nil {
		t.Fatalf("expected Tag.Key not to match, got %s", v.LegacyName())
	}
}

func TestFindLegacyParameter_SubFieldRepeatListLeaf(t *testing.T) {
	api := &API{
		Parameters: []Parameter{
			{
				RawName:    "Filter",
				Type:       "array",
				ParamStyle: "repeatList",
				ElementFields: []Field{
					{RawName: "Key", Type: "string"},
					{RawName: "Value", Type: "array", ElementType: "string"},
				},
			},
		},
	}

	if v := api.FindLegacyParameter("Filter.1.Value.1"); v == nil || v.LegacyName() != "Value" {
		t.Fatalf("expected Filter.1.Value.1 to match Value, got %#v", v)
	}

	var names []string
	api.ForeachLegacyParameter(func(name string, v *LegacyParameterView) {
		names = append(names, name)
	})
	expected := []string{"Filter.1.Key", "Filter.1.Value.1"}
	if len(names) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, names)
		}
	}
}

func TestForeachLegacyParameter(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	api.ForeachLegacyParameter(func(name string, v *LegacyParameterView) {
		names = append(names, name)
	})

	// AcceptLanguage + Tags.1.Key + Tags.1.Value
	expected := []string{"AcceptLanguage", "Tags.1.Key", "Tags.1.Value"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected name[%d]=%s, got %s", i, expected[i], name)
		}
	}
}

func TestCheckLegacyRequiredParameters(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "DescribeRegions")
	if err != nil {
		t.Fatal(err)
	}

	// All parameters are optional, should pass
	err = api.CheckLegacyRequiredParameters(func(name string) bool { return false })
	if err != nil {
		t.Errorf("expected no error for optional params, got %v", err)
	}
}

func TestCheckLegacyRequiredParameters_Missing(t *testing.T) {
	repo := testFS()
	api, err := repo.GetAPI("demo", "2026-01-01", "CreateReport")
	if err != nil {
		t.Fatal(err)
	}

	// ReportId is required but not a body param, so it should be checked
	// Actually ReportId is path location, not body, so it stays
	err = api.CheckLegacyRequiredParameters(func(name string) bool { return false })
	if err == nil {
		t.Error("expected error for missing required params")
	}
}

func TestCheckLegacyRequiredParameters_V1BodyOverridesSameNamePublicParam(t *testing.T) {
	api := &API{
		Parameters: []Parameter{
			{RawName: "Payload", Location: "query", Type: "string", Required: false},
			{RawName: "Other", Location: "query", Type: "string", Required: true},
		},
	}
	api.V1BodyParameters = &[]LegacyBodyParameter{
		{Name: "Payload", Position: "body", Type: "string", Required: true},
	}

	err := api.CheckLegacyRequiredParameters(func(name string) bool {
		return name == "Other"
	})
	if err == nil {
		t.Fatal("expected v1 body Payload to be required")
	}
	if !strings.Contains(err.Error(), "--Payload") {
		t.Fatalf("expected Payload missing, got %s", err)
	}
}

func TestCheckLegacyRequiredParameters_V1BodyOptionalOverridesRequiredPublicParam(t *testing.T) {
	api := &API{
		Parameters: []Parameter{
			{RawName: "Payload", Location: "query", Type: "string", Required: true},
		},
	}
	api.V1BodyParameters = &[]LegacyBodyParameter{
		{Name: "Payload", Position: "body", Type: "string", Required: false},
	}

	err := api.CheckLegacyRequiredParameters(func(name string) bool { return false })
	if err != nil {
		t.Fatalf("expected optional v1 body Payload to override required public param, got %s", err)
	}
}

// ── Body view tests ──

func TestLegacyBodyView(t *testing.T) {
	body := &LegacyBodyParameter{
		Name:     "body",
		Position: "body",
		Type:     "string",
		Required: false,
	}
	v := NewBodyView(body)

	if v.LegacyName() != "body" {
		t.Errorf("expected body, got %s", v.LegacyName())
	}
	if v.LegacyPosition() != "Body" {
		t.Errorf("expected Body, got %s", v.LegacyPosition())
	}
	if v.LegacyType() != "string" {
		t.Errorf("expected string, got %s", v.LegacyType())
	}
	if v.LegacyRequired() {
		t.Error("expected not required")
	}
}

func TestLegacyBodyView_RepeatList(t *testing.T) {
	params := []LegacyBodyParameter{
		{
			Name:       "Items",
			Position:   "body",
			Type:       "array",
			ParamStyle: "repeatList",
		},
	}
	api := &API{V1BodyParameters: &params}

	if v := api.FindLegacyParameter("Items.1"); v == nil || v.LegacyName() != "Items" {
		t.Fatalf("expected Items.1 to match Items, got %#v", v)
	}

	var names []string
	api.ForeachLegacyParameter(func(name string, v *LegacyParameterView) {
		names = append(names, name)
	})
	if len(names) != 1 || names[0] != "Items.1" {
		t.Fatalf("expected Items.1 completion, got %v", names)
	}

	err := api.CheckLegacyRequiredParameters(func(name string) bool { return false })
	if err != nil {
		t.Fatalf("expected optional body repeatList not to fail required check, got %v", err)
	}
}
