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
	resetSessionState()
	cfg, err := Load(in.ConfigDir)
	if err != nil {
		return
	}
	ok, scope := ShouldCollectInner(in.CommandToken, in.Profile, cfg)
	if !ok {
		return
	}
	aiCfg, _ := aimode.Load(in.ConfigDir)
	aiOn := aiCfg != nil && aiCfg.Enabled
	beginSession(in.ConfigDir, scope, in.Profile, in.CommandArgs, aiOn, cfg)
}

func End(err error) {
	flush(nil, err)
}

func EndWithExitCode(exitCode int) {
	flush(&exitCode, nil)
}

func flush(exitCode *int, err error) {
	s, ok := takeSessionForEnd()
	if !ok {
		return
	}
	cfg := s.cfg
	if cfg == nil {
		cfg, _ = Load(s.configDir)
	}
	instanceID, idErr := EnsureInstanceID(s.configDir, cfg)
	if idErr != nil {
		return
	}
	ev, buildErr := BuildEvent(BuildInput{
		Scope:      s.scope,
		RootArgs:   s.rootArgs,
		Profile:    s.profile,
		InstanceID: instanceID,
		DurationMs: sessionDurationMs(s),
		Err:        err,
		ExitCode:   exitCode,
		AIMode:     s.aiMode,
	})
	if buildErr != nil {
		return
	}
	cacheDir := GetInnerCacheDir(s.configDir)
	cacheFile, writeErr := WriteEvent(cacheDir, ev)
	if writeErr != nil {
		return
	}
	spawnInnerUpload(s.configDir, cacheFile)
}

func spawnInnerUpload(configDir, cacheFile string) {
	exe, err := os.Executable()
	if err != nil {
		_ = os.Remove(cacheFile)
		return
	}
	cmd := exec.Command(
		exe,
		"__telemetry-upload",
		"--pipeline=inner",
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
		switch args[i] {
		case "--config-dir":
			if i+1 < len(args) {
				configDir = args[i+1]
				i++
			}
		case "--pipeline":
			if i+1 < len(args) {
				pipeline = args[i+1]
				i++
			}
		case "--cache-file":
			if i+1 < len(args) {
				cacheFile = args[i+1]
				i++
			}
		}
	}
	if pipeline != "inner" || configDir == "" || cacheFile == "" {
		return 0
	}
	_ = UploadInnerFile(configDir, cacheFile)
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
