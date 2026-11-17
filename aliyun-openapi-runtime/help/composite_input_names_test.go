package help

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestCompositeHelpUsesParserInputNames(t *testing.T) {
	scalar := meta.Parameter{Name: "managed_rule_identifier", RawName: "ManagedRuleIdentifier", Options: []string{"--managed-rule-identifier"}, Type: meta.TypeString}
	params := []meta.Parameter{{Name: "config_rules", RawName: "ConfigRules", Options: []string{"--config-rules"}, Type: meta.TypeArray,
		ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{scalar,
			{Name: "config_rule_parameters", RawName: "ConfigRuleParameters", Type: meta.TypeArray, ItemType: &meta.Parameter{Type: meta.TypeObject, Fields: []meta.Parameter{{Name: "parameter_name", RawName: "ParameterName", Type: meta.TypeString}}}},
		}}}}
	structure, format := parameterFormatHints(projectParameter(params[0]))
	document := BuildAPIRequestDocument(&meta.Product{Code: "config"}, &meta.API{
		Name: "CreateCompliancePack", CmdName: "create-compliance-pack", ProductCode: "config", Parameters: params,
	}, nil, HelpOptions{})
	var output bytes.Buffer
	if err := Render(&output, document, HelpOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "ManagedRuleIdentifier: string") || strings.Contains(output.String(), "managed-rule-identifier:") {
		t.Fatalf("ordinary help uses wrong JSON field names: %s", output.String())
	}
	for _, name := range []string{"ManagedRuleIdentifier: string", "ConfigRuleParameters: [{ParameterName: string}"} {
		if !strings.Contains(structure, name) {
			t.Fatalf("missing %q in %s", name, structure)
		}
	}
	if format != "--config-rules 'value'" {
		t.Fatal(format)
	}
	if _, err := argparser.Parse(params, []string{"--config-rules", `[{"ManagedRuleIdentifier":"oss-bucket-public-read-prohibited","ConfigRuleParameters":[{"ParameterName":"test"}]}]`}); err != nil {
		t.Fatal(err)
	}
	if _, err := argparser.Parse(params, []string{"--config-rules", `[{"managed-rule-identifier":"test"}]`}); err == nil || !strings.Contains(err.Error(), "unknown field: managed-rule-identifier") {
		t.Fatalf("error = %v, want unknown field: managed-rule-identifier", err)
	}
}

func TestCompositeHelpExamplesAndFieldNameFallback(t *testing.T) {
	scalar := meta.Parameter{Name: "managed_rule_identifier", RawName: "ManagedRuleIdentifier", Options: []string{"--managed-rule-identifier"}, Type: meta.TypeString}
	flat := meta.Parameter{Name: "rule", RawName: "Rule", Options: []string{"--rule"}, Type: meta.TypeObject, Fields: []meta.Parameter{scalar}}
	for _, param := range []meta.Parameter{flat, {Name: "rules", Options: []string{"--rules"}, Type: meta.TypeArray, ItemType: &flat}} {
		_, format := parameterFormatHints(projectParameter(param))
		if !strings.Contains(format, "ManagedRuleIdentifier=") || strings.Contains(format, "managed-rule-identifier=") {
			t.Fatal(format)
		}
	}
	mapped := Parameter{Type: meta.TypeMap, Value: &Parameter{Type: meta.TypeObject, Fields: []Parameter{{Name: "snake-field", RawName: "snake_field", Type: meta.TypeString}, {Name: "fallback", Type: meta.TypeString}}}}
	if got := parameterShape(mapped); got != "{<key>: {snake_field: string, fallback: string}, ...}" {
		t.Fatal(got)
	}
}
