//go:build parser_contract
// +build parser_contract

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type parserContractSequenceFixture struct {
	Source       string                       `json:"source"`
	Model        string                       `json:"model"`
	ExpectedFrom string                       `json:"expectedFrom"`
	Cases        []parserContractSequenceCase `json:"cases"`
}

type parserContractSequenceCase struct {
	Name                             string                   `json:"name"`
	Argv                             []string                 `json:"argv"`
	Token1Class                      parserContractTokenClass `json:"token1Class"`
	Token2Class                      parserContractTokenClass `json:"token2Class"`
	Step2AllowedDashValueEnhancement bool                     `json:"step2AllowedDashValueEnhancement"`
	FinalExpected                    parserContractObservable `json:"finalExpected"`
}

func TestParserContractSequenceCases(t *testing.T) {
	fixture := loadParserContractSequenceFixture(t)
	step1ByToken := indexInitParserContractCases(t, loadParserContractFixture(t))
	if fixture.Model != "formal/parser/Parser.tla" {
		t.Fatalf("model = %q, want formal/parser/Parser.tla", fixture.Model)
	}
	if fixture.ExpectedFrom != "selectedSeq2Case.step1NewObservable/newFinalObservable" {
		t.Fatalf("expectedFrom = %q, want selectedSeq2Case.step1NewObservable/newFinalObservable", fixture.ExpectedFrom)
	}
	if len(fixture.Cases) < 4000 {
		t.Fatalf("sequence case count = %d, want complete two-step token space", len(fixture.Cases))
	}

	for _, tc := range fixture.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			parser, ctx := newParserContractSequenceParser(tc)
			step1Expected, ok := step1ByToken[tc.Argv[0]]
			if !ok {
				t.Fatalf("missing init single-step oracle for first token %q", tc.Argv[0])
			}

			arg, flag, more, err := parser.readNext()
			step1Actual := captureParserContractObservable(parser, ctx, parserContractCase{Expected: step1Expected}, arg, flag, more, err)
			if !reflect.DeepEqual(step1Actual, step1Expected) {
				t.Fatalf("step1 observable mismatch for argv %#v\nexpected: %s\nactual:   %s", tc.Argv, prettyJSON(step1Expected), prettyJSON(step1Actual))
			}

			arg, flag, more, err = parser.readNext()
			finalActual := captureParserContractObservable(parser, ctx, parserContractCase{Expected: tc.FinalExpected}, arg, flag, more, err)
			if !reflect.DeepEqual(finalActual, tc.FinalExpected) {
				t.Fatalf("final observable mismatch for argv %#v\nexpected: %s\nactual:   %s", tc.Argv, prettyJSON(tc.FinalExpected), prettyJSON(finalActual))
			}
		})
	}
}

func loadParserContractSequenceFixture(t *testing.T) parserContractSequenceFixture {
	t.Helper()
	content, err := os.ReadFile("testdata/parser_contract_sequence_cases.json")
	if err != nil {
		t.Fatalf("read parser contract sequence cases: %v", err)
	}
	var fixture parserContractSequenceFixture
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatalf("unmarshal parser contract sequence cases: %v", err)
	}
	return fixture
}

func indexInitParserContractCases(t *testing.T, fixture parserContractFixture) map[string]parserContractObservable {
	t.Helper()
	out := map[string]parserContractObservable{}
	for _, c := range fixture.Cases {
		if c.StateName != "init" {
			continue
		}
		if _, ok := out[c.TokenRaw]; ok {
			t.Fatalf("duplicate init single-step oracle for token %q", c.TokenRaw)
		}
		out[c.TokenRaw] = c.Expected
	}
	return out
}

func newParserContractSequenceParser(tc parserContractSequenceCase) (*Parser, *Context) {
	ctx := NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	flags := ctx.Flags()
	flags.Add(&Flag{Name: "PortRange", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "Count", Shorthand: 'c', AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "Name", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "RegionId", AssignedMode: AssignedOnce})
	flags.Add(&Flag{Name: "help", Shorthand: 'h', AssignedMode: AssignedNone})

	if parserContractSequenceNeedsUnknownFlags(tc) {
		ctx.SetUnknownFlags(NewFlagSet())
		ctx.command = &Command{EnableUnknownFlag: true}
	} else {
		ctx.SetUnknownFlags(nil)
	}

	parser := NewParser(tc.Argv, ctx)
	parser.SetAllowUnknown(false)
	return parser, ctx
}

func parserContractSequenceNeedsUnknownFlags(tc parserContractSequenceCase) bool {
	return tc.Token1Class.DetectorResult == "dynamic_unknown_flag" ||
		tc.Token2Class.DetectorResult == "dynamic_unknown_flag"
}
