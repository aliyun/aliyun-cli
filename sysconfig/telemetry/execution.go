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
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

type BeginInput struct {
	ConfigDir    string
	CommandArgs  []string
	CommandToken string
	AuthMode     string
	// AIMode is the effective per-command state resolved by the host, including
	// flag, environment, agent detection, and saved configuration overrides.
	AIMode bool
}

type activeCommandExecution struct {
	configDir             string
	scope                 Scope
	rootArgs              []string
	start                 time.Time
	aiMode                bool
	authMode              string
	authModeSource        string
	pluginAction          *PluginManagementEvent
	openAPIRuntimeVersion string
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
func recoverCollectionPanic(configDirs ...string) {
	if recover() != nil {
		configDir := ""
		if activeExecution != nil {
			configDir = activeExecution.configDir
		}
		if len(configDirs) > 0 {
			configDir = configDirs[0]
		}
		writeDiagnostic(configDir, diagnosticRecord{Stage: "collection", Status: "failed", Reason: "panic"})
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

// SetOpenAPIRuntimeVersion records the version reported by the runtime host
// after it has selected the in-process OpenAPI runtime for this invocation.
func SetOpenAPIRuntimeVersion(version string) {
	defer recoverCollectionPanic()
	if activeExecution == nil || executionEnded {
		return
	}
	activeExecution.openAPIRuntimeVersion = version
}

// SetOpenAPILegacyRoute replaces a pre-routing plugin candidate when the host
// selects its built-in OpenAPI path. Installed plugins do not own this call.
func SetOpenAPILegacyRoute() {
	defer recoverCollectionPanic()
	if activeExecution == nil || executionEnded {
		return
	}
	activeExecution.scope = scopeFromLocalPlugin(activeExecution.scope.CommandToken, nil)
	activeExecution.scope.CommandType = CommandTypeOpenAPILegacy
	activeExecution.openAPIRuntimeVersion = ""
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
	defer recoverCollectionPanic(in.ConfigDir)
	resetExecutionState()
	writeDiagnostic(in.ConfigDir, diagnosticRecord{Stage: "collection", Status: "started"})
	// Managing telemetry must not create an ID or emit a new event, especially
	// immediately after disable or purge.
	if isTelemetryManagementCommand(in.CommandArgs) {
		writeDiagnostic(in.ConfigDir, diagnosticRecord{Stage: "collection", Status: "skipped", Reason: "management_command"})
		return
	}
	cfg, err := Load(in.ConfigDir)
	if err != nil {
		writeDiagnostic(in.ConfigDir, diagnosticRecord{
			Stage: "collection", Status: "failed", Reason: "invalid_config", ErrorKind: diagnosticErrorKind(err),
		})
		return
	}
	ok, scope := SelectScope(in.CommandToken, cfg)
	if !ok {
		reason := "scope_inactive"
		if Disabled(cfg) {
			reason = "disabled"
		}
		writeDiagnostic(in.ConfigDir, diagnosticRecord{Stage: "collection", Status: "skipped", Reason: reason})
		return
	}
	beginCommandExecution(in.ConfigDir, scope, in.CommandArgs, in.AIMode)
	SetEffectiveAuthenticationMode(in.AuthMode)
	writeDiagnostic(in.ConfigDir, diagnosticRecord{Stage: "collection", Status: "succeeded", Pipeline: scope.Pipeline})
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
	if configErr != nil {
		writeDiagnostic(execution.configDir, diagnosticRecord{
			Stage: "collection", Status: "failed", Reason: "invalid_config", ErrorKind: diagnosticErrorKind(configErr),
		})
		return
	}
	if Disabled(cfg) {
		writeDiagnostic(execution.configDir, diagnosticRecord{Stage: "collection", Status: "skipped", Reason: "disabled"})
		return
	}
	instanceID, idErr := EnsureInstanceID(execution.configDir)
	if idErr != nil {
		writeDiagnostic(execution.configDir, diagnosticRecord{
			Stage: "collection", Status: "failed", Reason: "instance_id_failed", ErrorKind: diagnosticErrorKind(idErr),
		})
		return
	}
	if classifyResult(err, exitCode) == "Success" {
		CompletePluginManagement(execution.pluginAction)
	} else {
		RemoveUnvalidatedPluginNames(execution.pluginAction)
	}
	ev, buildErr := BuildEvent(BuildInput{
		Scope:                 execution.scope,
		RootArgs:              execution.rootArgs,
		InstanceID:            instanceID,
		DurationMs:            commandExecutionDurationMs(execution),
		Err:                   err,
		ExitCode:              exitCode,
		AIMode:                execution.aiMode,
		AuthMode:              execution.authMode,
		AuthModeSource:        execution.authModeSource,
		PluginAction:          execution.pluginAction,
		OpenAPIRuntimeVersion: execution.openAPIRuntimeVersion,
	})
	if buildErr != nil {
		writeDiagnostic(execution.configDir, diagnosticRecord{
			Stage: "event_build", Status: "failed", ErrorKind: diagnosticErrorKind(buildErr), Pipeline: execution.scope.Pipeline,
		})
		return
	}
	writeDiagnostic(execution.configDir, diagnosticRecord{
		Stage: "event_build", Status: "succeeded", EventID: ev.EventID, Pipeline: execution.scope.Pipeline,
	})
	cacheDir := GetCacheDir(execution.configDir, execution.scope.Pipeline)
	cacheFile, writeErr := WriteEvent(cacheDir, ev)
	if writeErr != nil {
		writeDiagnostic(execution.configDir, diagnosticRecord{
			Stage: "cache_write", Status: "failed", ErrorKind: diagnosticErrorKind(writeErr),
			EventID: ev.EventID, Pipeline: execution.scope.Pipeline,
		})
		return
	}
	writeDiagnostic(execution.configDir, diagnosticRecord{
		Stage: "cache_write", Status: "succeeded", EventID: ev.EventID, Pipeline: execution.scope.Pipeline,
	})
	spawnUpload(execution.configDir, execution.scope.Pipeline, cacheFile, ev.EventID)
}

// HookEnd is suitable for the root command's AfterExecute callback.
func HookEnd(_ *cli.Context, _ []string, err error) {
	End(err)
}
