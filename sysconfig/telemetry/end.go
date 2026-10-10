// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"os"
	"os/exec"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

type BeginInput struct {
	ConfigDir    string
	CommandArgs  []string
	CommandToken string
	Profile      ProfileSnapshot
}

func Begin(in BeginInput) {
	resetExecutionState()
	cfg, err := Load(in.ConfigDir)
	if err != nil {
		return
	}
	ok, scope := SelectScope(in.CommandToken, in.Profile, cfg)
	if !ok {
		return
	}
	aiCfg, _ := aimode.Load(in.ConfigDir)
	aiOn := aiCfg != nil && aiCfg.Enabled
	beginCommandExecution(in.ConfigDir, scope, in.Profile, in.CommandArgs, aiOn, cfg)
}

func End(err error) {
	flush(nil, err)
}

func EndWithExitCode(exitCode int) {
	flush(&exitCode, nil)
}

func flush(exitCode *int, err error) {
	execution, ok := takeCommandExecutionForEnd()
	if !ok {
		return
	}
	cfg := execution.cfg
	if cfg == nil {
		cfg, _ = Load(execution.configDir)
	}
	instanceID, idErr := EnsureInstanceID(execution.configDir, cfg)
	if idErr != nil {
		return
	}
	if classifyResult(err, exitCode) == "Success" {
		CompletePluginManagement(execution.pluginAction)
	}
	ev, buildErr := BuildEvent(BuildInput{
		Scope:        execution.scope,
		RootArgs:     execution.rootArgs,
		Profile:      execution.profile,
		InstanceID:   instanceID,
		DurationMs:   commandExecutionDurationMs(execution),
		Err:          err,
		ExitCode:     exitCode,
		AIMode:       execution.aiMode,
		PluginAction: execution.pluginAction,
	})
	if buildErr != nil {
		return
	}
	cacheDir := GetCacheDir(execution.configDir, execution.scope.Pipeline)
	cacheFile, writeErr := WriteEvent(cacheDir, ev)
	if writeErr != nil {
		return
	}
	spawnUpload(execution.configDir, execution.scope.Pipeline, cacheFile)
}

func spawnUpload(configDir, pipeline, cacheFile string) {
	exe, err := os.Executable()
	if err != nil {
		_ = os.Remove(cacheFile)
		return
	}
	cmd := exec.Command(
		exe,
		"__telemetry-upload",
		"--pipeline",
		pipeline,
		"--config-dir",
		configDir,
		"--cache-file",
		cacheFile,
	)
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		_ = os.Remove(cacheFile)
	}
}

// RunUploadCommand handles the hidden __telemetry-upload entrypoint.
func RunUploadCommand(args []string) int {
	configDir := ""
	cacheFile := ""
	pipeline := ""
	for i := 0; i < len(args); i++ {
		switch {
		case strings.HasPrefix(args[i], "--config-dir="):
			configDir = strings.TrimPrefix(args[i], "--config-dir=")
		case args[i] == "--config-dir":
			if i+1 < len(args) {
				configDir = args[i+1]
				i++
			}
		case strings.HasPrefix(args[i], "--pipeline="):
			pipeline = strings.TrimPrefix(args[i], "--pipeline=")
		case args[i] == "--pipeline":
			if i+1 < len(args) {
				pipeline = args[i+1]
				i++
			}
		case strings.HasPrefix(args[i], "--cache-file="):
			cacheFile = strings.TrimPrefix(args[i], "--cache-file=")
		case args[i] == "--cache-file":
			if i+1 < len(args) {
				cacheFile = args[i+1]
				i++
			}
		}
	}
	if (pipeline != PipelineInner && pipeline != PipelinePublic) ||
		configDir == "" ||
		cacheFile == "" {
		return 0
	}
	_ = UploadFile(configDir, pipeline, cacheFile)
	return 0
}

func ProfileFromRegionEndpoint(regionID, endpoint string) ProfileSnapshot {
	return ProfileSnapshot{
		RegionID: regionID,
		Endpoint: endpoint,
	}
}

// HookEnd is suitable for the root command's AfterExecute callback.
func HookEnd(_ *cli.Context, _ []string, err error) {
	End(err)
}
