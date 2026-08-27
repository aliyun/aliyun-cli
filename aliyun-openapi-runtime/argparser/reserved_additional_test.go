package argparser

import (
	"strings"
	"testing"
)

func TestReservedFlagDescriptionFallbacks(t *testing.T) {
	flag := ReservedFlag{DescZH: "中文", DescEN: "English"}
	if flag.Desc("zh") != "中文" || flag.Desc("en") != "English" {
		t.Fatalf("localized descriptions = %q, %q", flag.Desc("zh"), flag.Desc("en"))
	}
	if got := (ReservedFlag{DescZH: "中文"}).Desc("en"); got != "中文" {
		t.Fatalf("Chinese fallback = %q", got)
	}
	if got := (ReservedFlag{DescEN: "English"}).Desc("zh"); got != "English" {
		t.Fatalf("English fallback = %q", got)
	}
}

func TestNewReservedFlagIndexRejectsInvalidRegistrations(t *testing.T) {
	tests := []struct {
		name  string
		specs []reservedFlagSpec
		want  string
	}{
		{name: "empty name", specs: []reservedFlagSpec{{}}, want: "name is empty"},
		{name: "duplicate name", specs: []reservedFlagSpec{{name: "one"}, {name: "one"}}, want: "registered more than once"},
		{name: "duplicate shorthand", specs: []reservedFlagSpec{{name: "one", shorthand: 'x'}, {name: "two", shorthand: 'x'}}, want: "registered by both"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil || !strings.Contains(got.(string), tc.want) {
					t.Fatalf("panic = %v, want %q", got, tc.want)
				}
			}()
			newReservedFlagIndex(tc.specs...)
		})
	}
	index := newReservedFlagIndex(reservedFlagSpec{name: "one"}, reservedFlagSpec{name: "two", shorthand: 't'})
	if len(index.byName) != 2 || index.byShorthand['t'] != "two" {
		t.Fatalf("index = %#v", index)
	}
}

func TestReservedObjectApplicationBranches(t *testing.T) {
	r := &Reserved{}
	if err := applyPager(r, map[string]string{
		"path": "Data.Items[]", "PageNumber": "PN", "PageSize": "PS", "TotalCount": "TC", "NextToken": "NT",
	}); err != nil {
		t.Fatal(err)
	}
	if r.Pager.Path != "Data.Items[]" || r.Pager.PageNumber != "PN" || r.Pager.PageSize != "PS" || r.Pager.TotalCount != "TC" || r.Pager.NextToken != "NT" {
		t.Fatalf("pager = %#v", r.Pager)
	}
	if err := applyPager(r, map[string]string{"bad": "value"}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("pager unknown field error = %v", err)
	}

	r = &Reserved{}
	if err := applyWaiter(r, map[string]string{"expr": "Status", "to": "Ready", "timeout": "60", "interval": "2"}); err != nil {
		t.Fatal(err)
	}
	if r.Waiter.Expr != "Status" || r.Waiter.To != "Ready" || r.Waiter.Timeout != 60 || r.Waiter.Interval != 2 {
		t.Fatalf("waiter = %#v", r.Waiter)
	}
	for _, kv := range []map[string]string{{"bad": "x"}, {"timeout": "bad"}, {"timeout": "0"}, {"interval": "bad"}, {"interval": "-1"}} {
		if err := applyWaiter(&Reserved{}, kv); err == nil {
			t.Fatalf("applyWaiter(%v) succeeded", kv)
		}
	}

	for _, kv := range []map[string]string{{"bad": "x"}, {}, {"cols": ""}} {
		if err := applyOutputTable(&Reserved{}, kv); err == nil {
			t.Fatalf("applyOutputTable(%v) succeeded", kv)
		}
	}
	r = &Reserved{}
	if err := applyOutputTable(r, map[string]string{"cols": " Id, , Name ", "rows": "Items", "num": "1"}); err != nil {
		t.Fatal(err)
	}
	if len(r.OutputTable.Cols) != 2 || r.OutputTable.Cols[0] != "Id" || !r.OutputTable.ShowNum {
		t.Fatalf("output table = %#v", r.OutputTable)
	}
}

func TestReservedParsingErrorsAndValueModes(t *testing.T) {
	if err := parseOutputFlag(&Reserved{}, nil); err == nil || !strings.Contains(err.Error(), "expects cols") {
		t.Fatalf("empty output error = %v", err)
	}
	if err := parseOutputFlag(&Reserved{}, []string{"not-object"}); err == nil || !strings.Contains(err.Error(), "expected key=value") {
		t.Fatalf("invalid output error = %v", err)
	}
	for _, tokens := range [][]string{{"broken"}, {"=value"}} {
		if _, err := parseReservedObject(tokens); err == nil {
			t.Fatalf("parseReservedObject(%v) succeeded", tokens)
		}
	}
	if got, err := parseReservedObject([]string{"", "a=one=two"}); err != nil || got["a"] != "one=two" {
		t.Fatalf("parseReservedObject = %#v, %v", got, err)
	}

	r := &Reserved{}
	apply := func(r *Reserved, values []string) error {
		r.Region = strings.Join(values, ",")
		return nil
	}
	for _, tc := range []struct {
		mode   reservedValueMode
		inline string
		want   string
	}{
		{reservedNoValue, "ignored", ""},
		{reservedSingleValue, "single", "single"},
		{reservedMultiValue, "multi", "multi"},
	} {
		r.Region = ""
		_, err := consumeReservedFlag(nil, 0, nil, nil, reservedFlagSpec{mode: tc.mode, apply: apply}, tc.inline, true, r)
		if err != nil || r.Region != tc.want {
			t.Fatalf("mode %d = %q, %v", tc.mode, r.Region, err)
		}
	}
	if _, err := consumeReservedFlag(nil, 0, nil, nil, reservedFlagSpec{mode: 99, apply: apply}, "", false, r); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported mode error = %v", err)
	}
	wantErr := func(*Reserved, []string) error { return errReservedAdditional }
	if _, err := consumeReservedFlag(nil, 0, nil, nil, reservedFlagSpec{mode: reservedNoValue, apply: wantErr}, "", false, r); err != errReservedAdditional {
		t.Fatalf("apply error = %v", err)
	}
}

var errReservedAdditional = &reservedAdditionalError{}

type reservedAdditionalError struct{}

func (*reservedAdditionalError) Error() string { return "reserved additional error" }
