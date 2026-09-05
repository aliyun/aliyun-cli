package canonicalmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDistribution(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty", raw: "", want: nil},
		{name: "whitespace", raw: "  ", want: nil},
		{name: "go only", raw: "go", want: []string{"go"}},
		{name: "uppercase go", raw: "GO", want: []string{"go"}},
		{name: "meta", raw: "meta", want: []string{"meta"}},
		{name: "openapi", raw: "openapi", want: []string{"openapi"}},
		{name: "openapi|go", raw: "openapi|go", want: []string{"openapi", "go"}},
		{name: "go|openapi", raw: "go|openapi", want: []string{"go", "openapi"}},
		{name: "spaces around tokens", raw: " openapi | go ", want: []string{"openapi", "go"}},
		{name: "empty token skipped", raw: "openapi||go|", want: []string{"openapi", "go"}},
		{name: "duplicate tokens", raw: "go|GO|openapi", want: []string{"go", "openapi"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseDistribution(tt.raw))
		})
	}
}

func TestHasDistribution(t *testing.T) {
	assert.False(t, HasDistribution("", DistributionGo))
	assert.True(t, HasDistribution("go", DistributionGo))
	assert.True(t, HasDistribution("GO", DistributionGo))
	assert.False(t, HasDistribution("go", DistributionOpenAPI))
	assert.False(t, HasDistribution("meta", DistributionGo))
	assert.False(t, HasDistribution("openapi", DistributionGo))
	assert.True(t, HasDistribution("openapi|go", DistributionGo))
	assert.True(t, HasDistribution("openapi|go", DistributionOpenAPI))
	assert.True(t, HasDistribution(" go | openapi ", DistributionGo))
	assert.False(t, HasDistribution("openapi|go", DistributionMeta))
	assert.False(t, HasDistribution("openapi|go", ""))

	assert.True(t, ProductEntry{Distribution: "openapi|go"}.HasDistribution(DistributionGo))
	assert.False(t, ProductEntry{Distribution: "openapi"}.HasDistribution(DistributionGo))
}
