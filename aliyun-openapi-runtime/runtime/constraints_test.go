package runtime

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func constraintAPI(parameter meta.Parameter) *meta.API {
	return &meta.API{Parameters: []meta.Parameter{parameter}}
}

func TestValidateConstraintsEnumIsCaseSensitive(t *testing.T) {
	parameter := meta.Parameter{
		Name: "mode", RawName: "Mode", Type: meta.TypeString,
		Options: []string{"--mode"}, Enum: []string{"ReadOnly", "ReadWrite"},
	}
	err := ValidateConstraints(constraintAPI(parameter), map[string]any{"Mode": "readonly"}, false)

	var violation *ConstraintViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("error = %v, want ConstraintViolationError", err)
	}
	if violation.Parameter != "mode" || violation.Flag != "--mode" ||
		violation.Path != "Mode" || violation.Constraint != "enum" ||
		violation.Actual != "readonly" ||
		!reflect.DeepEqual(violation.Allowed, parameter.Enum) {
		t.Fatalf("violation = %#v", violation)
	}
	if got, want := err.Error(), `constraint violation: parameter="mode" flag="--mode" path="Mode" constraint="enum" actual="readonly" allowed=["ReadOnly","ReadWrite"]`; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestValidateConstraintsNumericBoundsAreInclusiveAndExact(t *testing.T) {
	parameter := meta.Parameter{
		Name: "id", RawName: "Id", Type: meta.TypeLong,
		Minimum: "9007199254740993", Maximum: "9007199254740995",
	}
	api := constraintAPI(parameter)
	for _, value := range []json.Number{"9007199254740993", "9007199254740995"} {
		if err := ValidateConstraints(api, map[string]any{"Id": value}, false); err != nil {
			t.Fatalf("inclusive bound rejected %s: %v", value, err)
		}
	}

	err := ValidateConstraints(api, map[string]any{"Id": json.Number("9007199254740992")}, false)
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) || violation.Constraint != "minimum" ||
		violation.Expected != "9007199254740993" {
		t.Fatalf("minimum violation = %#v, error = %v", violation, err)
	}
}

func TestValidateConstraintsStringLengthUsesUnicodeCodePoints(t *testing.T) {
	parameter := meta.Parameter{
		Name: "name", RawName: "Name", Type: meta.TypeString,
		MinLength: "2", MaxLength: "3",
	}
	api := constraintAPI(parameter)
	for _, value := range []string{"阿里", "a云c"} {
		if err := ValidateConstraints(api, map[string]any{"Name": value}, false); err != nil {
			t.Fatalf("inclusive length rejected %q: %v", value, err)
		}
	}

	err := ValidateConstraints(api, map[string]any{"Name": "云"}, false)
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) || violation.Constraint != "minLength" ||
		violation.Expected != "2" {
		t.Fatalf("minLength violation = %#v, error = %v", violation, err)
	}

	err = ValidateConstraints(api, map[string]any{"Name": "阿里云CLI"}, false)
	if !errors.As(err, &violation) || violation.Constraint != "maxLength" ||
		violation.Expected != "3" {
		t.Fatalf("maxLength violation = %#v, error = %v", violation, err)
	}
}

func TestValidateConstraintsRecursesObjectArrayAndMap(t *testing.T) {
	parameter := meta.Parameter{
		Name: "config", RawName: "Config", Type: meta.TypeObject, Options: []string{"--config"},
		Fields: []meta.Parameter{{
			Name: "groups", RawName: "Groups", Type: meta.TypeArray,
			ItemType: &meta.Parameter{
				Type: meta.TypeMap,
				ValueType: &meta.Parameter{
					Type: meta.TypeObject,
					Fields: []meta.Parameter{{
						Name: "name", RawName: "Name", Type: meta.TypeString, Pattern: "^[a-z]+$",
					}},
				},
			},
		}},
	}
	args := map[string]any{"Config": map[string]any{
		"Groups": []any{map[string]any{
			"secondary": map[string]any{"Name": "valid"},
			"primary":   map[string]any{"Name": "INVALID"},
		}},
	}}

	err := ValidateConstraints(constraintAPI(parameter), args, false)
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) {
		t.Fatalf("error = %v, want ConstraintViolationError", err)
	}
	if violation.Parameter != "name" || violation.Flag != "--config" ||
		violation.Path != `Config.Groups[0]["primary"].Name` ||
		violation.Constraint != "pattern" {
		t.Fatalf("recursive violation = %#v", violation)
	}
}

func TestValidateConstraintsFailsOpenForInvalidMetadata(t *testing.T) {
	parameters := []meta.Parameter{
		{Name: "count", RawName: "Count", Type: meta.TypeInteger, Minimum: "not-a-number", Maximum: "also-invalid"},
		{Name: "name", RawName: "Name", Type: meta.TypeString, MinLength: "bad", MaxLength: "-1", Pattern: "["},
	}
	api := &meta.API{Parameters: parameters}
	args := map[string]any{"Count": json.Number("10"), "Name": "anything"}
	if err := ValidateConstraints(api, args, false); err != nil {
		t.Fatalf("invalid producer metadata must fail open: %v", err)
	}
}

func TestValidateConstraintsSkipsOptionalAnyAndRawBody(t *testing.T) {
	api := &meta.API{Parameters: []meta.Parameter{
		{Name: "optional", RawName: "Optional", Type: meta.TypeString, Enum: []string{"ok"}},
		{Name: "payload", RawName: "Payload", Type: meta.TypeAny, Enum: []string{"never"}},
		{Name: "body", RawName: "Body", Type: meta.TypeString, Position: meta.PosBody, Enum: []string{"ok"}},
		{Name: "form", RawName: "Form", Type: meta.TypeString, Position: meta.PosFormData, Pattern: "^ok$"},
	}}
	args := map[string]any{"Payload": "anything", "Body": "bad", "Form": "bad"}
	if err := ValidateConstraints(api, args, true); err != nil {
		t.Fatalf("optional, TypeAny, and raw body values should be skipped: %v", err)
	}
}

func TestConstraintNumericMaximumAndPatterns(t *testing.T) {
	maximum := meta.Parameter{Name: "ratio", RawName: "Ratio", Type: meta.TypeFloat, Maximum: "1.5"}
	err := ValidateConstraints(constraintAPI(maximum), map[string]any{"Ratio": float64(2)}, false)
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) || violation.Constraint != "maximum" || violation.Expected != "1.5" {
		t.Fatalf("maximum violation = %#v, %v", violation, err)
	}
	pattern := meta.Parameter{Name: "name", RawName: "Name", Type: meta.TypeString, Pattern: "^[a-z]+$"}
	err = ValidateConstraints(constraintAPI(pattern), map[string]any{"Name": "BAD"}, false)
	if !errors.As(err, &violation) || violation.Constraint != "pattern" {
		t.Fatalf("pattern violation = %#v, %v", violation, err)
	}
	if err := ValidateConstraints(nil, nil, false); err != nil {
		t.Fatalf("nil API error = %v", err)
	}
}

func TestConstraintHelperConversions(t *testing.T) {
	ratTests := []struct {
		value any
		want  string
		ok    bool
	}{
		{json.Number("1.25"), "5/4", true}, {float64(1.5), "3/2", true}, {float32(2.5), "5/2", true},
		{int(3), "3", true}, {int32(4), "4", true}, {int64(5), "5", true}, {"6", "", false},
	}
	for _, tc := range ratTests {
		got, ok := constraintRat(tc.value)
		if ok != tc.ok {
			t.Errorf("constraintRat(%#v) ok = %v", tc.value, ok)
			continue
		}
		if ok && got.RatString() != tc.want {
			t.Errorf("constraintRat(%#v) = %s", tc.value, got.RatString())
		}
	}
	actualTests := []struct {
		value any
		want  string
	}{
		{"text", "text"}, {json.Number("9.1"), "9.1"}, {float64(2.5), "2.5"}, {float32(3.5), "3.5"}, {true, "true"}, {int(7), "7"},
	}
	for _, tc := range actualTests {
		if got := constraintActual(tc.value); got != tc.want {
			t.Errorf("constraintActual(%#v) = %q, want %q", tc.value, got, tc.want)
		}
	}
	if pathName(&meta.Parameter{Name: "fallback"}) != "fallback" || pathName(&meta.Parameter{Name: "name", RawName: "Raw"}) != "Raw" {
		t.Fatal("pathName fallback failed")
	}
	for _, tc := range []struct{ parent, child, want string }{
		{"", "child", "child"}, {"parent", "", "parent"}, {"parent", "child", "parent.child"},
	} {
		if got := appendObjectPath(tc.parent, tc.child); got != tc.want {
			t.Errorf("appendObjectPath(%q,%q) = %q", tc.parent, tc.child, got)
		}
	}
}

func TestValidateConstraintValueIgnoresShapeMismatches(t *testing.T) {
	context := constraintContext{parameter: "value", flag: "--value", path: "Value"}
	tests := []struct {
		parameter *meta.Parameter
		value     any
	}{
		{nil, "x"},
		{&meta.Parameter{Type: meta.TypeAny}, "x"},
		{&meta.Parameter{Type: meta.TypeString, MinLength: "2"}, nil},
		{&meta.Parameter{Type: meta.TypeInteger, Minimum: "2"}, "not-number"},
		{&meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{RawName: "Name", Type: meta.TypeString}}}, "not-object"},
		{&meta.Parameter{Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeString}}, "not-array"},
		{&meta.Parameter{Type: meta.TypeArray}, []any{"x"}},
		{&meta.Parameter{Type: meta.TypeMap, ValueType: &meta.Parameter{Type: meta.TypeString}}, "not-map"},
		{&meta.Parameter{Type: meta.TypeMap}, map[string]any{"a": "b"}},
	}
	for i, tc := range tests {
		if err := validateConstraintValue(tc.parameter, tc.value, context); err != nil {
			t.Errorf("case %d returned %v", i, err)
		}
	}
}

func TestConstraintErrorCopiesAllowedValues(t *testing.T) {
	allowed := []string{"a", "b"}
	err := constraintError(constraintContext{parameter: "p", flag: "--p", path: "P"}, &meta.Parameter{}, "enum", "c", "", allowed)
	allowed[0] = "changed"
	var violation *ConstraintViolationError
	if !errors.As(err, &violation) || violation.Allowed[0] != "a" {
		t.Fatalf("constraint error = %#v", violation)
	}
	withExpected := &ConstraintViolationError{Parameter: "p", Constraint: "minimum", Actual: "1", Expected: "2"}
	if !strings.Contains(withExpected.Error(), `expected="2"`) {
		t.Fatalf("expected formatting = %q", withExpected.Error())
	}
}
