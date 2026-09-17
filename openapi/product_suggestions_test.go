package openapi

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/require"
)

func TestProductSuggestionsMatchPriority(t *testing.T) {
	for _, tc := range []struct {
		name, input      string
		candidates, want []string
	}{
		{"prefix beats substring and closer typo", "bss", []string{"dbs", "ess", "abss", "BssOpenApi"}, []string{"bssopenapi"}},
		{"substring beats closer typo", "openapi", []string{"openapx", "BssOpenApi", "OtherOpenApiService"}, []string{"bssopenapi", "otheropenapiservice"}},
		{"distance fallback keeps nearest matches", "ecx", []string{"Ecs", "Ecsa", "Eci", "unrelated"}, []string{"eci", "ecs"}},
		{"case whitespace and duplicates", " BSS ", []string{" BssOpenApi ", "bssopenapi", "", "dbs"}, []string{"bssopenapi"}},
		{"prefix limit is deterministic", "svc", []string{"svcf", "svce", "svcd", "svcc", "svcb", "svca"}, []string{"svca", "svcb", "svcc", "svcd", "svce"}},
		{"substring limit is deterministic", "svc", []string{"fsvc", "esvc", "dsvc", "csvc", "bsvc", "asvc"}, []string{"asvc", "bsvc", "csvc", "dsvc", "esvc"}},
		{"no match", "unrelated", []string{"ecs", "bssopenapi"}, []string{}},
		{"empty input", " ", []string{"ecs"}, nil},
		{"empty catalog", "bss", nil, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, productSuggestions(tc.input, tc.candidates))
			reversed := append([]string(nil), tc.candidates...)
			for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
				reversed[i], reversed[j] = reversed[j], reversed[i]
			}
			require.Equal(t, tc.want, productSuggestions(tc.input, reversed))
		})
	}
}

func TestUnknownProductSuggestionsAndRecoveryUseBestMatch(t *testing.T) {
	for _, input := range []string{"bss", "openapi"} {
		t.Run(input, func(t *testing.T) {
			products := []meta.Product{{Code: "Dbs"}, {Code: "Ess"}, {Code: "BssOpenApi"}}
			repo, err := meta.MockLoadRepository(products)
			require.NoError(t, err)
			builtin := &InvalidProductError{Code: input, library: &Library{builtinRepo: repo}}
			extension := &InvalidProductOrPluginError{Code: input, plugins: []plugin.PluginInfo{{ProductCode: "dbs"}, {ProductCode: "ess"}, {ProductCode: "BssOpenApi"}}}
			for _, cause := range []interface {
				error
				GetSuggestions() []string
				AgentSuggestions() []string
			}{builtin, extension} {
				require.Equal(t, []string{"bssopenapi"}, cause.GetSuggestions())
				require.Equal(t, cause.GetSuggestions(), cause.AgentSuggestions())
				var searches []string
				envelope := requireAgentEnvelope(t, cause, []string{input, "--help-search", "MonthBill"}, func(req RecoverySearchRequest) bool { searches = append(searches, req.Keyword); return true })
				require.Equal(t, []string{"bssopenapi"}, envelope.DidYouMean)
				require.Equal(t, "aliyun --help-search bssopenapi", envelope.Recovery.Command)
				require.Equal(t, []string{"bssopenapi"}, searches)
			}
		})
	}
	require.Nil(t, (&InvalidProductError{Code: "bss"}).GetSuggestions())
}
