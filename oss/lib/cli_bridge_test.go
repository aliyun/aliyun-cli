package lib

import (
	"bytes"
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCliBridge(t *testing.T) {
	NewCommandBridge(configCommand.command)
}

func TestParseAndGetEndpoint(t *testing.T) {
	type args struct {
		ctx  *cli.Context
		args []string
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	context := cli.NewCommandContext(w, stderr)
	flag := cli.Flag{
		Name: "endpoint",
	}
	flag.SetValue("oss-cn-hangzhou.aliyuncs.com")
	context.Flags().Add(&flag)

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Valid endpoint from args",
			args: args{
				ctx:  new(cli.Context),
				args: []string{"--endpoint", "oss-cn-shenzhen.aliyuncs.com"},
			},
			want:    "oss-cn-shenzhen.aliyuncs.com",
			wantErr: assert.NoError,
		},
		{
			name: "Valid region from args",
			args: args{
				ctx:  new(cli.Context),
				args: []string{"--region", "cn-shenzhen"},
			},
			want:    "oss-cn-shenzhen.aliyuncs.com",
			wantErr: assert.NoError,
		},
		{
			name: "Fetch endpoint flag from context",
			args: args{
				ctx: context,
			},
			want:    "oss-cn-hangzhou.aliyuncs.com",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAndGetEndpoint(tt.args.ctx, tt.args.args)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseAndGetEndpoint(%v, %v)", tt.args.ctx, tt.args.args)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ParseAndGetEndpoint(%v, %v)", tt.args.ctx, tt.args.args)
		})
	}
}
