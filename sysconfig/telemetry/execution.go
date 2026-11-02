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
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

type BeginInput struct {
	ConfigDir    string
	CommandArgs  []string
	CommandToken string
	AuthMode     string
}

type activeCommandExecution struct {
	configDir      string
	scope          Scope
	rootArgs       []string
	start          time.Time
	aiMode         bool
	authMode       string
	authModeSource string
	pluginAction   *PluginManagementEvent
}

var activeExecution *activeCommandExecution
var executionEnded bool

const (
	AuthModeSourceExplicit = "explicit"
	AuthModeSourceDefault  = "default"
	AuthModeSourceExisting = "existing"
)

func resetExecutionState() {
	activeExecution = nil
	executionEnded = false
}

// Defer only within automatic telemetry entry points, never around command
// execution or explicit telemetry management. A recoverable panic discards this
// invocation's collection silently, without retrying it from later callbacks.
func recoverCollectionPanic() {
	if recover() != nil {
		activeExecution = nil
		executionEnded = true
	}
}

func beginCommandExecution(
	configDir string,
	scope Scope,
	rootArgs []string,
	aiMode bool,
) {
	activeExecution = &activeCommandExecution{
		configDir:    configDir,
		scope:        scope,
		rootArgs:     append([]string(nil), rootArgs...),
		start:        time.Now(),
		aiMode:       aiMode,
		pluginAction: DetectPluginManagement(rootArgs),
	}
	executionEnded = false
}

func takeCommandExecutionForEnd() (*activeCommandExecution, bool) {
	if activeExecution == nil || executionEnded {
		return nil, false
	}
	executionEnded = true
	return activeExecution, true
}

// SetEffectiveAuthenticationMode records only a known authentication-mode enum
// for the profile resolved by the active command.
func SetEffectiveAuthenticationMode(mode string) {
	defer recoverCollectionPanic()
	if activeExecution == nil {
		return
	}
	activeExecution.authMode = safeAuthenticationMode(mode)
}

// SetAuthenticationMode records the mode selected by an active configure
// command together with its selection source.
func SetAuthenticationMode(mode, source string) {
	defer recoverCollectionPanic()
	if activeExecution == nil {
		return
	}
	canonicalMode := safeAuthenticationMode(mode)
	if canonicalMode == "" || !safeAuthModeSource(source) {
		return
	}
	activeExecution.authMode = canonicalMode
	activeExecution.authModeSource = source
}

func safeAuthenticationMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "ak":
		return "AK"
	case "ststoken":
		return "StsToken"
	case "ramrolearn":
		return "RamRoleArn"
	case "ecsramrole":
		return "EcsRamRole"
	case "rsakeypair":
		return "RsaKeyPair"
	case "ramrolearnwithrolename":
		return "RamRoleArnWithRoleName"
	case "chainableramrolearn":
		return "ChainableRamRoleArn"
	case "external":
		return "External"
	case "credentialsuri":
		return "CredentialsURI"
	case "oidc":
		return "OIDC"
	case "cloudsso":
		return "CloudSSO"
	case "oauth":
		return "OAuth"
	case "bearertoken":
		return "BearerToken"
	case "anonymous":
		return "Anonymous"
	default:
		return ""
	}
}

func safeAuthModeSource(source string) bool {
	return source == AuthModeSourceExplicit ||
		source == AuthModeSourceDefault ||
		source == AuthModeSourceExisting
}

func commandExecutionDurationMs(execution *activeCommandExecution) int64 {
	if execution == nil {
		return 0
	}
	return time.Since(execution.start).Milliseconds()
}

func Begin(in BeginInput) {
	defer recoverCollectionPanic()
	resetExecutionState()
	// Managing telemetry must not create an ID or emit a new event, especially
	// immediately after disable or purge.
	if isTelemetryManagementCommand(in.CommandArgs) {
		return
	}
	cfg, err := Load(in.ConfigDir)
	if err != nil {
		return
	}
	ok, scope := SelectScope(in.CommandToken, cfg)
	if !ok {
		return
	}
	aiCfg, _ := aimode.Load(in.ConfigDir)
	aiOn := aiCfg != nil && aiCfg.Enabled
	beginCommandExecution(in.ConfigDir, scope, in.CommandArgs, aiOn)
	SetEffectiveAuthenticationMode(in.AuthMode)
}

func isTelemetryManagementCommand(args []string) bool {
	for _, command := range []string{"configure", "telemetry"} {
		args = sysmock.StripLeadingGlobalFlags(args)
		// The command parser accepts "help" before any subcommand.
		for len(args) > 0 && args[0] == "help" {
			args = sysmock.StripLeadingGlobalFlags(args[1:])
		}
		if len(args) == 0 || args[0] != command {
			return false
		}
		args = args[1:]
	}
	return true
}

func End(err error) {
	flush(nil, err)
}

func EndWithExitCode(exitCode int) {
	flush(&exitCode, nil)
}

func flush(exitCode *int, err error) {
	defer recoverCollectionPanic()
	execution, ok := takeCommandExecutionForEnd()
	if !ok {
		return
	}
	// A command may have disabled telemetry while this invocation was running.
	cfg, configErr := Load(execution.configDir)
	if configErr != nil || Disabled(cfg) {
		return
	}
	instanceID, idErr := EnsureInstanceID(execution.configDir)
	if idErr != nil {
		return
	}
	if classifyResult(err, exitCode) == "Success" {
		CompletePluginManagement(execution.pluginAction)
	} else {
		RemoveUnvalidatedPluginNames(execution.pluginAction)
	}
	ev, buildErr := BuildEvent(BuildInput{
		Scope:          execution.scope,
		RootArgs:       execution.rootArgs,
		InstanceID:     instanceID,
		DurationMs:     commandExecutionDurationMs(execution),
		Err:            err,
		ExitCode:       exitCode,
		AIMode:         execution.aiMode,
		AuthMode:       execution.authMode,
		AuthModeSource: execution.authModeSource,
		PluginAction:   execution.pluginAction,
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

// HookEnd is suitable for the root command's AfterExecute callback.
func HookEnd(_ *cli.Context, _ []string, err error) {
	End(err)
}
