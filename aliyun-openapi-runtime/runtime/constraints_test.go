package runtime

import (
	"encoding/json"
	"errors"
	"reflect"
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
