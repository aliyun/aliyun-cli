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
)

type activeCommandExecution struct {
	configDir      string
	scope          Scope
	rootArgs       []string
	start          time.Time
	aiMode         bool
	authMode       string
	authModeSource string
	cfg            *Config
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

func beginCommandExecution(
	configDir string,
	scope Scope,
	rootArgs []string,
	aiMode bool,
	cfg *Config,
) {
	activeExecution = &activeCommandExecution{
		configDir:    configDir,
		scope:        scope,
		rootArgs:     append([]string(nil), rootArgs...),
		start:        time.Now(),
		aiMode:       aiMode,
		cfg:          cfg,
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
	if activeExecution == nil {
		return
	}
	activeExecution.authMode = safeAuthenticationMode(mode)
}

// SetAuthenticationMode records the mode selected by an active configure
// command together with its selection source.
func SetAuthenticationMode(mode, source string) {
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
