/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package command

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/config"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/posener/complete"
)

func TestNewTestCommand(t *testing.T) {
	excmd := &cli.Command{
		Name:   "test",
		Usage:  "Test",
		Hidden: true,
		Run: func(ctx *cli.Context, args []string) error {
			run := complete.Command{
				Sub: complete.Commands{
					"build": complete.Command{
						Flags: complete.Flags{
							"-cpus": complete.PredictAnything,
						},
					},
				},
				Flags: complete.Flags{
					"-o": complete.PredictFiles("*.out"),
				},
				GlobalFlags: complete.Flags{
					"-h": complete.PredictNothing,
				},
			}
			complete.New("run", run).Run()
			return nil
		},
	}

	cmd := NewTestCommand()
	assert.ObjectsAreEqualValues(excmd, cmd)
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	config.AddFlags(ctx.Flags())
	cmd.Run(ctx, []string{"Test"})
}
