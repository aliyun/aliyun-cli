package runtime

import (
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestRequiredErrorsKeepReadableFlagAndWirePaths(t *testing.T) {
	for _, tc := range []struct {
		err  *MissingRequiredError
		want string
	}{
		{&MissingRequiredError{Flags: []string{"--id", "--name"}}, "missing required parameter(s): --id, --name"},
		{&MissingRequiredError{Flags: []string{"--config", "--config"}, Paths: []string{"Config.ID", "Config.Name"}}, "missing required parameter(s): --config (Config.ID), --config (Config.Name)"},
		{&MissingRequiredError{Flags: []string{"--id"}, Paths: []string{"ID", "extra"}}, "missing required parameter(s): --id"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Fatalf("error=%q,want %q", got, tc.want)
		}
	}
	err := &EndpointNotResolvedError{Product: "ecs", Region: "cn-test"}
	if got := err.Error(); !strings.Contains(got, `product "ecs" region "cn-test"; pass --endpoint`) {
		t.Fatalf("endpoint error=%q", got)
	}
	if err := ValidateRequired(nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDocRequired(nil, nil, false); err != nil {
		t.Fatal(err)
	}
}

func TestDocRequiredIgnoresMalformedOptionalContainers(t *testing.T) {
	for _, parameter := range []meta.Parameter{
		{Type: meta.TypeObject, Fields: []meta.Parameter{{RawName: "required", DocRequired: true}}},
		{Type: meta.TypeArray, ItemType: &meta.Parameter{DocRequired: true}},
		{Type: meta.TypeMap, ValueType: &meta.Parameter{DocRequired: true}},
		{Type: meta.TypeArray}, {Type: meta.TypeMap},
	} {
		parameter.Name, parameter.RawName = "config", "Config"
		api := &meta.API{Parameters: []meta.Parameter{parameter}}
		if err := ValidateDocRequired(api, map[string]any{"Config": "wrong type"}, false); err != nil {
			t.Fatalf("optional malformed container should be left to type validation: %v", err)
		}
	}
	missing := &missingRequired{}
	validateDocRequiredValue(nil, nil, true, docRequiredContext{}, missing)
	if len(missing.flags) != 0 {
		t.Fatalf("nil metadata produced missing flags: %v", missing.flags)
	}
}
