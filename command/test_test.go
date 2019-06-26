// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
