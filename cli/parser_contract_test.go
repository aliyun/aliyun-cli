//go:build parser_contract
// +build parser_contract

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

type parserContractFixture struct {
	Source       string               `json:"source"`
	Model        string               `json:"model"`
	ExpectedFrom string               `json:"expectedFrom"`
	Cases        []parserContractCase `json:"cases"`
}

type parserContractCase struct {
	Name                        string                   `json:"name"`
	StateName                   string                   `json:"stateName"`
	Argv                        []string                 `json:"argv"`
	TokenRaw                    string                   `json:"tokenRaw"`
	TokenClass                  parserContractTokenClass `json:"tokenClass"`
	TokenFlag                   string                   `json:"tokenFlag"`
	TokenValue                  string                   `json:"tokenValue"`
	TokenParseError             parserContractError      `json:"tokenParseError"`
	AllowedDashValueEnhancement bool                     `json:"allowedDashValueEnhancement"`
	ExpectedSource              string                   `json:"expectedSource"`
	Expected                    parserContractObservable `json:"expected"`
	OldObservable               parserContractObservable `json:"oldObservable"`
}

type parserContractTokenClass struct {
	SplitterClass  string `json:"splitterClass"`
	PrefixShape    string `json:"prefixShape"`
	DetectorResult string `json:"detectorResult"`
}

type parserContractError struct {
	Class string `json:"class"`
	Text  string `json:"text"`
}

type parserContractFlagObservable struct {
	Assigned bool     `json:"assigned"`
	Value    string   `json:"value"`
	Values   []string `json:"values"`
}

type parserContractObservable struct {
	Current     int                                     `json:"current"`
	OutArg      string                                  `json:"outArg"`
	OutFlag     string                                  `json:"outFlag"`
	OutMore     bool                                    `json:"outMore"`
	Err         parserContractError                     `json:"err"`
	Flags       map[string]parserContractFlagObservable `json:"flags"`
	PendingFlag string                                  `json:"pendingFlag"`
	PendingFrom string                                  `json:"pendingFrom"`
}

func TestParserContractCases(t *testing.T) {
	fixture := loadParserContractFixture(t)
	if fixture.Model != "formal/parser/Parser.tla" {
		t.Fatalf("model = %q, want formal/parser/Parser.tla", fixture.Model)
	}
	if fixture.ExpectedFrom != "selectedCase.newObservable" {
		t.Fatalf("expectedFrom = %q, want selectedCase.newObservable", fixture.ExpectedFrom)
	}
	if len(fixture.Cases) <= 65 {
		t.Fatalf("case count = %d, want regenerated complete token-class space, not old 65-case pilot", len(fixture.Cases))
	}

	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			parser, ctx := newParserContractParser(tc)
			consumeParserContractPrefix(t, parser, tc)

			arg, flag, more, err := parser.readNext()
			actual := captureParserContractObservable(parser, ctx, tc, arg, flag, more, err)
			if !reflect.DeepEqual(actual, tc.Expected) {
				t.Fatalf("observable mismatch for argv %#v\nexpected: %s\nactual:   %s", tc.Argv, prettyJSON(tc.Expected), prettyJSON(actual))
			}
		})
	}
}

func loadParserContractFixture(t *testing.T) parserContractFixture {
	t.Helper()
	content, err := os.ReadFile("testdata/parser_contract_cases.json")
	if err != nil {
		t.Fatalf("read parser contract cases: %v", err)
	}
	var fixture parserContractFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("unmarshal parser contract cases: %v", err)
	}
	return fixture
}

func newParserContractParser(tc parserContractCase) (*Parser, *Context) {
	ctx := NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	flags := ctx.Flags()
	flags.Add(&Flag{Name: "PortRange", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "Count", Shorthand: 'c', AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "Name", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "RegionId", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "help", Shorthand: 'h', AssignedMode: AssignedNone})

	if needsUnknownFlags(tc) {
		ctx.SetUnknownFlags(NewFlagSet())
		ctx.command = &Command{EnableUnknownFlag: true}
	} else {
		ctx.SetUnknownFlags(nil)
	}

	parser := NewParser(tc.Argv, ctx)
	parser.SetAllowUnknown(false)
	return parser, ctx
}

func needsUnknownFlags(tc parserContractCase) bool {
	return strings.Contains(tc.StateName, "dynamic") ||
		tc.TokenClass.DetectorResult == "dynamic_unknown_flag"
}

func consumeParserContractPrefix(t *testing.T, parser *Parser, tc parserContractCase) {
	t.Helper()
	prefixLen := len(tc.Argv) - 1
	for i := 0; i < prefixLen; i++ {
		_, _, _, err := parser.readNext()
		if err != nil {
			t.Fatalf("prefix read %d for argv %#v returned error: %v", i, tc.Argv, err)
		}
	}
	if parser.GetCurrent() != prefixLen {
		t.Fatalf("after prefix current = %d, want %d", parser.GetCurrent(), prefixLen)
	}
	if prefixLen == 0 && parser.currentFlag != nil {
		t.Fatalf("init prefix produced pending flag %s", parser.currentFlag.Name)
	}
	if want, wantOrigin, ok := parserContractPendingState(tc.StateName); ok {
		if parser.currentFlag == nil || parser.currentFlag.Name != want {
			got := "NoFlag"
			if parser.currentFlag != nil {
				got = parser.currentFlag.Name
			}
			t.Fatalf("after prefix pending flag = %s, want %s", got, want)
		}
		if parser.currentFlagOrigin != wantOrigin {
			t.Fatalf("after prefix pending origin = %s, want %s", parser.currentFlagOrigin, wantOrigin)
		}
	}
}

func parserContractPendingState(stateName string) (flagName, origin string, ok bool) {
	switch stateName {
	case "pending_PortRange":
		return "PortRange", "long_flag_candidate", true
	case "pending_Count":
		return "Count", "long_flag_candidate", true
	case "pending_Name":
		return "Name", "long_flag_candidate", true
	case "pending_RegionId":
		return "RegionId", "long_flag_candidate", true
	case "pending_short_Count":
		return "Count", "short_flag_candidate", true
	case "pending_dynamic_long_dynamic":
		return "dynamic", "dynamic_unknown_flag", true
	case "pending_dynamic_short_u":
		return "u", "dynamic_unknown_flag", true
	default:
		return "", "", false
	}
}

func captureParserContractObservable(parser *Parser, ctx *Context, tc parserContractCase, arg string, flag *Flag, more bool, err error) parserContractObservable {
	outFlag := "NoFlag"
	if flag != nil {
		outFlag = flag.Name
	}

	pendingFlag := "NoFlag"
	if parser.currentFlag != nil {
		pendingFlag = parser.currentFlag.Name
	}

	return parserContractObservable{
		Current:     parser.GetCurrent(),
		OutArg:      arg,
		OutFlag:     outFlag,
		OutMore:     more,
		Err:         classifyParserContractError(err),
		Flags:       captureParserContractFlags(ctx, tc.Expected.Flags),
		PendingFlag: pendingFlag,
		PendingFrom: parser.currentFlagOrigin,
	}
}

func captureParserContractFlags(ctx *Context, expected map[string]parserContractFlagObservable) map[string]parserContractFlagObservable {
	out := make(map[string]parserContractFlagObservable)
	for name := range expected {
		f := ctx.Flags().Get(name)
		if f == nil && ctx.UnknownFlags() != nil {
			f = ctx.UnknownFlags().Get(name)
		}
		out[name] = captureParserContractFlag(f)
	}
	return out
}

func captureParserContractFlag(f *Flag) parserContractFlagObservable {
	if f == nil {
		return parserContractFlagObservable{Values: []string{}}
	}
	values := append([]string{}, f.values...)
	if values == nil {
		values = []string{}
	}
	return parserContractFlagObservable{
		Assigned: f.assigned,
		Value:    f.value,
		Values:   values,
	}
}

func classifyParserContractError(err error) parserContractError {
	if err == nil {
		return parserContractError{Class: "NoError", Text: ""}
	}
	text := err.Error()
	switch {
	case strings.HasPrefix(text, "--") && strings.HasSuffix(text, " must be assigned with value"):
		return parserContractError{Class: "MissingValue", Text: text}
	case strings.HasPrefix(text, "--") && strings.HasSuffix(text, " duplicated"):
		return parserContractError{Class: "DuplicateFlag", Text: text}
	case strings.HasPrefix(text, "flag duplicated --"):
		return parserContractError{Class: "FlagSetDuplicate", Text: text}
	case text == "not support '--' in command line":
		return parserContractError{Class: "InvalidDoubleDash", Text: text}
	case strings.HasPrefix(text, "not support flag form "):
		return parserContractError{Class: "InvalidFlagForm", Text: text}
	case strings.HasPrefix(text, "invalid flag "):
		return parserContractError{Class: "InvalidFlag", Text: text}
	case strings.HasPrefix(text, "unknown flag -"):
		return parserContractError{Class: "UnknownShortFlag", Text: text}
	case strings.HasPrefix(text, "flag --") && strings.HasSuffix(text, " can't be assiged"):
		return parserContractError{Class: "AssignToNoValueFlag", Text: text}
	default:
		return parserContractError{Class: "UnknownError", Text: text}
	}
}

func prettyJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", v)
	}
	return string(data)
}
