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
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/posener/complete"
)

func NewTestCommand() *cli.Command {
	return &cli.Command{
		Name:   "test",
		Usage:  "Test",
		Hidden: true,
		Run: func(ctx *cli.Context, args []string) error {
			// create a Command object, that represents the command we want
			// to complete.
			run := complete.Command{

				// Sub defines a list of sub commands of the program,
				// this is recursive, since every command is of type command also.
				Sub: complete.Commands{

					// add a build sub command
					"build": complete.Command{

						// define flags of the build sub command
						Flags: complete.Flags{
							// build sub command has a flag '-cpus', which
							// expects number of cpus after it. in that case
							// anything could complete this flag.
							"-cpus": complete.PredictAnything,
						},
					},
				},

				// define flags of the 'run' main command
				Flags: complete.Flags{
					// a flag -o, which expects a file ending with .out after
					// it, the tab completion will auto complete for files matching
					// the given pattern.
					"-o": complete.PredictFiles("*.out"),
				},

				// define global flags of the 'run' main command
				// those will show up also when a sub command was entered in the
				// command line
				GlobalFlags: complete.Flags{

					// a flag '-h' which does not expects anything after it
					"-h": complete.PredictNothing,
				},
			}

			// run the command completion, as part of the main() function.
			// this triggers the autocompletion when needed.
			// name must be exactly as the binary that we want to complete.
			complete.New("run", run).Run()
			return nil
		},
	}
}
