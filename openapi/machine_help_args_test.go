package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeMachineHelpArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "root equals",
			in:   []string{"--help=json"},
			want: []string{"--help", "--help-format", "json"},
		},
		{
			name: "api equals",
			in:   []string{"ecs", "describe-instances", "--help=json"},
			want: []string{"ecs", "describe-instances", "--help", "--help-format", "json"},
		},
		{
			name: "help command separated format",
			in:   []string{"help", "ecs", "DescribeInstances", "--format", "json"},
			want: []string{"ecs", "DescribeInstances", "--help", "--help-format", "json"},
		},
		{
			name: "help command equals format",
			in:   []string{"help", "ecs", "--format=json"},
			want: []string{"ecs", "--help", "--help-format", "json"},
		},
		{
			name: "invalid format still reaches machine error renderer",
			in:   []string{"help", "ecs", "--format", "yaml"},
			want: []string{"ecs", "--help", "--help-format", "yaml"},
		},
		{
			name: "text unchanged",
			in:   []string{"help", "ecs"},
			want: []string{"help", "ecs"},
		},
		{
			name: "ordinary format unchanged",
			in:   []string{"ecs", "Call", "--format", "json"},
			want: []string{"ecs", "Call", "--format", "json"},
		},
		{
			name: "input is not mutated",
			in:   []string{"ecs", "Call", "--help=json"},
			want: []string{"ecs", "Call", "--help", "--help-format", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := append([]string(nil), tt.in...)
			assert.Equal(t, tt.want, NormalizeMachineHelpArgs(tt.in))
			assert.Equal(t, before, tt.in)
		})
	}
}
