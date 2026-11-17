package argparser

import (
	"reflect"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestParseObservesOnlyConfirmedParameterNames(t *testing.T) {
	params := append(schema(), meta.Parameter{
		Name: "name", RawName: "Name", Type: meta.TypeString,
		Options: []string{"--name", "--NameAlias"},
	})
	external := []ExternalFlagSpec{
		{Name: "profile", Shorthand: 'p', Mode: ExternalFlagOptional},
		{Name: "read-timeout", Mode: ExternalFlagRequired},
		{Name: "cli-ai-mode", Mode: ExternalFlagNone},
		{Name: "rejected", Mode: ExternalFlagRequired, RejectMessage: "host flag is unavailable"},
	}
	cases := []struct {
		name    string
		args    []string
		want    []string
		wantErr bool
	}{
		{name: "empty"},
		{name: "scalar dash value", args: []string{"--name", "--customer-secret-token"}, want: []string{"--name"}},
		{name: "array dash values", args: []string{"--images", "--customer-secret-token", "another", "--another-private-value"}, want: []string{"--images"}},
		{name: "array ends at registered runtime and host flags", args: []string{"--images", "--customer-secret-token", "--quiet", "--profile", "private-profile"}, want: []string{"--images", "--quiet", "--profile"}},
		{name: "array JSON values", args: []string{"--images", `["--customer-secret-token"]`, "--quiet"}, want: []string{"--images", "--quiet"}},
		{name: "map business key", args: []string{"--labels", "--customer-secret-token=value", "env=prod"}, want: []string{"--labels"}},
		{name: "runtime dash value", args: []string{"--region", "--customer-secret-token"}, want: []string{"--region"}},
		{name: "host dash value", args: []string{"--profile", "--customer-secret-token"}, want: []string{"--profile"}},
		{name: "order aliases case and repeats", args: []string{"--NameAlias", "first", "-q", "--name", "second", "--NameAlias=third", "--name", "fourth", "-q"}, want: []string{"--NameAlias", "-q", "--name", "--NameAlias", "--name", "-q"}},
		{name: "repeated arrays", args: []string{"--images", "first", "--images", "--customer-secret-token"}, want: []string{"--images", "--images"}},
		{name: "inline API and runtime", args: []string{"--name=--customer-secret-token", "--region=--private-region"}, want: []string{"--name", "--region"}},
		{name: "empty API inline", args: []string{"--name="}, want: []string{"--name"}},
		{name: "host inline and short names", args: []string{"--profile:--customer-secret-token", "-p=--private-profile", "-p:other", "--read-timeout=30", "--cli-ai-mode"}, want: []string{"--profile", "-p", "-p", "--read-timeout", "--cli-ai-mode"}},
		{name: "API colon is not inline syntax", args: []string{"--name:private-value"}, wantErr: true},
		{name: "runtime colon is not inline syntax", args: []string{"--region:private-value"}, wantErr: true},
		{name: "unknown first", args: []string{"--customer-secret-token"}, wantErr: true},
		{name: "unknown after scalar value", args: []string{"--name", "known-value", "--customer-secret-token", "--quiet"}, want: []string{"--name"}, wantErr: true},
		{name: "unexpected positional", args: []string{"private-value", "--quiet"}, wantErr: true},
		{name: "API missing value", args: []string{"--name"}, want: []string{"--name"}, wantErr: true},
		{name: "API missing value before registered boundary", args: []string{"--name", "--quiet"}, want: []string{"--name"}, wantErr: true},
		{name: "runtime omitted optional value", args: []string{"--region"}, want: []string{"--region"}},
		{name: "host missing value", args: []string{"--read-timeout"}, want: []string{"--read-timeout"}, wantErr: true},
		{name: "API type failure", args: []string{"--name", "--customer-secret-token", "--count", "--another-private-value", "--quiet"}, want: []string{"--name", "--count"}, wantErr: true},
		{name: "composite type failure", args: []string{"--scores", "--customer-secret-token=not-a-number", "--quiet"}, want: []string{"--scores"}, wantErr: true},
		{name: "invalid composite inline", args: []string{"--images=--customer-secret-token"}, want: []string{"--images"}, wantErr: true},
		{name: "runtime composite failure", args: []string{"--output", "cols=Id", "--customer-secret-token", "--quiet"}, want: []string{"--output"}, wantErr: true},
		{name: "runtime ignored inline value", args: []string{"-q=private-value"}, want: []string{"-q"}},
		{name: "host value rejection", args: []string{"--cli-ai-mode:private-value"}, want: []string{"--cli-ai-mode"}, wantErr: true},
		{name: "host explicit rejection", args: []string{"--rejected", "--customer-secret-token"}, want: []string{"--rejected"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseline, baselineErr := ParseWithOptions(params, tc.args, ParseOptions{ExternalFlags: external})
			var observed []string
			calls := 0
			got, gotErr := ParseWithOptions(params, tc.args, ParseOptions{
				ExternalFlags: external,
				OnParameterNames: func(names []string) {
					calls++
					observed = names
				},
			})
			if calls != 1 {
				t.Fatalf("observer calls = %d, want 1", calls)
			}
			if !reflect.DeepEqual(observed, tc.want) {
				t.Fatalf("observed names = %q, want %q", observed, tc.want)
			}
			if (gotErr != nil) != tc.wantErr {
				t.Fatalf("parse error = %v, want error %v", gotErr, tc.wantErr)
			}
			if !reflect.DeepEqual(got, baseline) || !reflect.DeepEqual(gotErr, baselineErr) {
				t.Fatalf("observation changed parsing: got (%#v, %v), baseline (%#v, %v)", got, gotErr, baseline, baselineErr)
			}
		})
	}
}

func TestParameterNameObserverRunsOnInvalidExternalDefinitions(t *testing.T) {
	calls := 0
	_, err := ParseWithOptions(nil, []string{"--private-token"}, ParseOptions{
		ExternalFlags: []ExternalFlagSpec{{Name: ""}},
		OnParameterNames: func(names []string) {
			calls++
			if len(names) != 0 {
				t.Fatalf("invalid definitions produced names: %q", names)
			}
		},
	})
	if err == nil || calls != 1 {
		t.Fatalf("error = %v, observer calls = %d", err, calls)
	}
}

func TestParameterNameObserverCannotChangeParsing(t *testing.T) {
	for _, args := range [][]string{
		{"--image-cache-name", "--private-token"},
		{"--count", "not-a-number"},
	} {
		original := append([]string(nil), args...)
		baseline, baselineErr := Parse(schema(), args)
		var retained []string
		got, gotErr := ParseWithOptions(schema(), args, ParseOptions{
			OnParameterNames: func(names []string) {
				retained = names
				names[0] = "changed-by-observer"
				panic("observer unavailable")
			},
		})
		if !reflect.DeepEqual(got, baseline) || !reflect.DeepEqual(gotErr, baselineErr) {
			t.Fatalf("observer changed result: got (%#v, %v), baseline (%#v, %v)", got, gotErr, baseline, baselineErr)
		}
		if !reflect.DeepEqual(args, original) {
			t.Fatalf("observer changed argv: %q", args)
		}
		var next []string
		_, _ = ParseWithOptions(schema(), args, ParseOptions{OnParameterNames: func(names []string) { next = names }})
		if next[0] != original[0] || retained[0] != "changed-by-observer" {
			t.Fatalf("name snapshots share state: previous=%q next=%q", retained, next)
		}
	}
}
