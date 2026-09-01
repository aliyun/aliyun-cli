package canonicalmeta

import (
	"reflect"
	"testing"
)

func TestLegacyParameterViewAccessorsForEverySource(t *testing.T) {
	canonical := &Parameter{
		RawName: "Canonical", Type: "string", Required: true, DocRequired: true,
		Location: "query", IsWildcard: true, Example: "canonical-example",
		HelpZh: "规范中文", HelpEn: "canonical English",
		Enum: []string{"a"}, Minimum: "1", Maximum: "9", MinLength: "2", MaxLength: "8", Pattern: "a+",
	}
	field := &Field{
		RawName: "Field", Type: "int", Required: true, DocRequired: true,
		Example: "field-example", HelpZh: "字段中文", HelpEn: "field English",
		Enum: []string{"2"}, Minimum: "2", Maximum: "4", MinLength: "1", MaxLength: "3", Pattern: "[0-9]+",
	}
	body := &V1Parameter{
		Name: "Body", Position: "body", Type: "string", Required: true,
		Example: "body-example", DescriptionZh: "正文中文", DescriptionEn: "body English",
	}
	v1 := &V1Parameter{
		Name: "Legacy", Position: "header", Type: "long", Required: true,
		Example: "legacy-example", HelpZh: "旧版中文", HelpEn: "legacy English",
	}

	tests := []struct {
		name         string
		view         *LegacyParameterView
		legacyName   string
		position     string
		typeName     string
		description  string
		example      string
		required     bool
		docRequired  bool
		wildcard     bool
		constraintTy string
	}{
		{"canonical", NewCanonicalView(canonical), "Canonical", "Query", "string", "canonical English", "canonical-example", true, true, true, "string"},
		{"field", NewFieldView(field, "form"), "Field", "FormData", "int", "field English", "field-example", true, true, false, "int"},
		{"body", NewBodyView(body), "Body", "Body", "string", "body English", "body-example", true, false, false, "string"},
		{"v1", NewV1View(v1), "Legacy", "Header", "long", "legacy English", "legacy-example", true, false, false, "long"},
		{"unknown", &LegacyParameterView{source: LegacyParameterSource(99)}, "", "", "", "", "", false, false, false, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.view.LegacyName(); got != test.legacyName {
				t.Fatalf("LegacyName() = %q, want %q", got, test.legacyName)
			}
			if got := test.view.LegacyPosition(); got != test.position {
				t.Fatalf("LegacyPosition() = %q, want %q", got, test.position)
			}
			if got := test.view.LegacyType(); got != test.typeName {
				t.Fatalf("LegacyType() = %q, want %q", got, test.typeName)
			}
			if got := test.view.LegacyRequired(); got != test.required {
				t.Fatalf("LegacyRequired() = %v, want %v", got, test.required)
			}
			if got := test.view.DocRequired(); got != test.docRequired {
				t.Fatalf("DocRequired() = %v, want %v", got, test.docRequired)
			}
			if got := test.view.IsWildcard(); got != test.wildcard {
				t.Fatalf("IsWildcard() = %v, want %v", got, test.wildcard)
			}
			if got := test.view.LegacyDescription("en"); got != test.description {
				t.Fatalf("LegacyDescription() = %q, want %q", got, test.description)
			}
			if got := test.view.LegacyExample(); got != test.example {
				t.Fatalf("LegacyExample() = %q, want %q", got, test.example)
			}
			if got := test.view.ConstraintType(); got != test.constraintTy {
				t.Fatalf("ConstraintType() = %q, want %q", got, test.constraintTy)
			}
		})
	}

	if got := NewCanonicalView(canonical).Constraints(); !reflect.DeepEqual(got.Enum, []string{"a"}) || got.MinLength != "2" || got.MaxLength != "8" {
		t.Fatalf("canonical Constraints() = %#v", got)
	}
	if got := NewFieldView(field, "query").Constraints(); !reflect.DeepEqual(got.Enum, []string{"2"}) || got.Minimum != "2" || got.Maximum != "4" {
		t.Fatalf("field Constraints() = %#v", got)
	}
	if got := NewBodyView(body).Constraints(); !reflect.DeepEqual(got, Constraints{}) {
		t.Fatalf("body Constraints() = %#v, want empty", got)
	}
}

func TestLegacyParameterViewScalarArrayConstraints(t *testing.T) {
	shape := &TypeShape{
		Type: "integer", Enum: []string{"1", "2"}, Minimum: "1", Maximum: "2",
		MinLength: "1", MaxLength: "1", Pattern: "[12]",
	}
	canonical := NewCanonicalView(&Parameter{Type: "array", Element: shape})
	field := NewFieldView(&Field{Type: "array", Element: shape}, "query")
	for _, view := range []*LegacyParameterView{canonical, field} {
		if got := view.ConstraintType(); got != "integer" {
			t.Fatalf("ConstraintType() = %q, want integer", got)
		}
		constraints := view.Constraints()
		if !reflect.DeepEqual(constraints.Enum, []string{"1", "2"}) || constraints.Minimum != "1" || constraints.Maximum != "2" || constraints.Pattern != "[12]" {
			t.Fatalf("Constraints() = %#v", constraints)
		}
	}

	for _, typ := range []string{"string", "int", "integer", "int32", "int64", "long", "float", "double", "number", "bool", "boolean"} {
		if !isScalarTypeShape(&TypeShape{Type: typ}) {
			t.Errorf("isScalarTypeShape(%q) = false", typ)
		}
	}
	for _, shape := range []*TypeShape{nil, {Type: "object"}, {Type: "array"}, {Type: "map"}} {
		if isScalarTypeShape(shape) {
			t.Errorf("isScalarTypeShape(%#v) = true", shape)
		}
	}
	if got := NewCanonicalView(&Parameter{Type: "array", Element: &TypeShape{Type: "object"}}).ConstraintType(); got != "array" {
		t.Fatalf("non-scalar array ConstraintType() = %q", got)
	}
}

func TestLegacyLocationsRepeatListsAndChildren(t *testing.T) {
	for _, test := range []struct{ input, top, sub string }{
		{"query", "Query", "Query"}, {"body", "Body", "Body"}, {"host", "Host", "Host"},
		{"domain", "Domain", "Domain"}, {"path", "Path", "Path"}, {"header", "Header", "Header"},
		{"form", "Body", "FormData"}, {"formData", "Body", "FormData"}, {"unknown", "", ""},
	} {
		if got := legacyPosition(test.input); got != test.top {
			t.Errorf("legacyPosition(%q) = %q", test.input, got)
		}
		if got := legacySubPosition(test.input); got != test.sub {
			t.Errorf("legacySubPosition(%q) = %q", test.input, got)
		}
	}

	fields := []Field{{RawName: "Name", Type: "string"}}
	canonical := NewCanonicalView(&Parameter{Type: "array", ParamStyle: "repeatList", Location: "query", Element: &TypeShape{Type: "object", Fields: fields}})
	if !canonical.LegacyHasChildren() || !canonical.IsLegacyRepeatList() {
		t.Fatal("canonical repeat list should expose children")
	}
	children := canonical.LegacyChildren()
	if len(children) != 1 || children[0].LegacyName() != "Name" || children[0].LegacyChildren() != nil {
		t.Fatalf("canonical children = %#v", children)
	}

	for _, style := range []string{"flat", "json"} {
		view := NewCanonicalView(&Parameter{Type: "array", ParamStyle: style, Element: &TypeShape{Type: "object", Fields: fields}})
		if view.IsLegacyRepeatList() {
			t.Fatalf("style %q unexpectedly behaves as RepeatList", style)
		}
		if style == "flat" && view.LegacyHasChildren() {
			t.Fatal("flat array unexpectedly exposes legacy children")
		}
		if style == "json" && !view.LegacyHasChildren() {
			t.Fatal("json object array should retain its schema children for help")
		}
	}
	if NewCanonicalView(&Parameter{Type: "string"}).LegacyHasChildren() {
		t.Fatal("scalar canonical parameter should not have children")
	}
	if NewFieldView(&Field{Type: "array"}, "query").LegacyHasChildren() || !NewFieldView(&Field{Type: "array"}, "query").IsLegacyRepeatList() {
		t.Fatal("field array should be a leaf repeat list")
	}
	unknown := &LegacyParameterView{source: LegacyParameterSource(99)}
	if unknown.LegacyHasChildren() || unknown.IsLegacyRepeatList() || unknown.LegacyChildren() != nil {
		t.Fatal("unknown source should have no legacy structure")
	}

	body := &V1Parameter{Name: "body", Type: "array", ParamStyle: "repeatList", SubParameters: []V1Parameter{{Name: "Value", Position: "form", Type: "string"}}}
	bodyView := NewBodyView(body)
	bodyView.isTopBody = true
	if !bodyView.IsTopLevelBody() || !bodyView.LegacyHasChildren() || !bodyView.IsLegacyRepeatList() {
		t.Fatal("top-level body repeat list state was not preserved")
	}
	bodyChildren := bodyView.LegacyChildren()
	if len(bodyChildren) != 1 || bodyChildren[0].LegacyPosition() != "FormData" || bodyChildren[0].IsTopLevelBody() {
		t.Fatalf("body children = %#v", bodyChildren)
	}
	for _, style := range []string{"flat", "json"} {
		if NewBodyView(&V1Parameter{Type: "array", ParamStyle: style}).IsLegacyRepeatList() {
			t.Fatalf("V1 style %q unexpectedly behaves as RepeatList", style)
		}
	}
}

func TestLegacyBodyFieldsAndProtocolDefaults(t *testing.T) {
	api := &API{
		Method: "GET|POST", Protocol: "HTTP|HTTPS",
		Parameters: []Parameter{
			{RawName: "Query", Location: "query"},
			{RawName: "Body", Location: "body"},
			{RawName: "Form", Location: "form"},
			{RawName: "FormData", Location: "formData"},
			{RawName: "Action", Location: "body"},
		},
	}
	fields := api.LegacyBodyFields()
	if got := []string{fields[0].RawName, fields[1].RawName, fields[2].RawName}; !reflect.DeepEqual(got, []string{"Body", "Form", "FormData"}) {
		t.Fatalf("LegacyBodyFields() = %v", got)
	}
	if api.GetMethod() != "POST" || api.GetProtocol() != "https" {
		t.Fatalf("method/protocol = %q/%q", api.GetMethod(), api.GetProtocol())
	}
	if (&API{Method: "PATCH", Protocol: "HTTP"}).GetMethod() != "GET" || (&API{Protocol: "HTTP"}).GetProtocol() != "http" {
		t.Fatal("legacy method/protocol defaults changed")
	}
	v1 := []V1Parameter{}
	api.V1Parameters = &v1
	if api.LegacyBodyFields() != nil {
		t.Fatal("V1 projection must suppress canonical body fields")
	}
}
