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
			want: []string{"--help", "--cli-output", "json"},
		},
		{
			name: "api equals",
			in:   []string{"ecs", "describe-instances", "--help=json"},
			want: []string{"ecs", "describe-instances", "--help", "--cli-output", "json"},
		},
		{
			name: "help command separated format",
			in:   []string{"help", "ecs", "DescribeInstances", "--format", "json"},
			want: []string{"ecs", "DescribeInstances", "--help", "--cli-output", "json"},
		},
		{
			name: "help response section and format stay orthogonal",
			in:   []string{"help", "ecs", "DescribeInstances", "--cli-section", "response", "--format", "json"},
			want: []string{"ecs", "DescribeInstances", "--cli-section", "response", "--help", "--cli-output", "json"},
		},
		{
			name: "help command equals format",
			in:   []string{"help", "ecs", "--format=json"},
			want: []string{"ecs", "--help", "--cli-output", "json"},
		},
		{
			name: "invalid format still reaches machine error renderer",
			in:   []string{"help", "ecs", "--format", "yaml"},
			want: []string{"ecs", "--help", "--cli-output", "yaml"},
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
			want: []string{"ecs", "Call", "--help", "--cli-output", "json"},
		},
		{
			name: "compatibility response help json keeps section",
			in:   []string{"ecs", "DescribeInstances", "--cli-section", "response", "--help=json"},
			want: []string{"ecs", "DescribeInstances", "--cli-section", "response", "--help", "--cli-output", "json"},
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
