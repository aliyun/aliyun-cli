package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserReturnsFlagErrorsFromAllReadMethods(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "missing value before next flag", args: []string{"--test", "--prev"}, want: "must be assigned with value"},
		{name: "duplicate flag", args: []string{"--test=value", "--test=again"}, want: "duplicated"},
		{name: "value on no-value flag", args: []string{"--prev=value"}, want: "can't be assiged"},
		{name: "unknown shorthand after short flag", args: []string{"-t", "-x"}, want: "unknown flag -x"},
		{name: "unknown long flag", args: []string{"--missing"}, want: "unknown flag --missing"},
	} {
		for _, method := range []string{"all", "next"} {
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				parser, _ := newTestParser(tc.args...)
				if method == "all" {
					_, err := parser.ReadAll()
					require.ErrorContains(t, err, tc.want)
					return
				}
				for {
					_, more, err := parser.ReadNextArg()
					if err != nil {
						require.ErrorContains(t, err, tc.want)
						return
					}
					if !more {
						t.Fatal("expected parse error")
					}
				}
			})
		}
	}
}

func TestParserConsumesNegativeValueOnlyAfterDeclaredLongFlag(t *testing.T) {
	parser, flags := newTestParser("--test", "-42", "positional")
	args, err := parser.ReadAll()
	require.NoError(t, err)
	require.Equal(t, []string{"positional"}, args)
	value, assigned := flags.Get("test").GetValue()
	require.True(t, assigned)
	require.Equal(t, "-42", value)
	parser, flags = newTestParser("--test", "-42")
	flags.Get("test").dynamicUnknown = true
	_, err = parser.ReadAll()
	require.ErrorContains(t, err, "not support flag form")
	parser, _ = newTestParser("-t", "-42")
	_, err = parser.ReadAll()
	require.ErrorContains(t, err, "not support flag form")
	for _, value := range []string{"-a=value", "--long", "plain"} {
		require.False(t, isDashLeadingValueCandidate(value), value)
	}
}
