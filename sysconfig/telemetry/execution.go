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
	"time"
)

type activeCommandExecution struct {
	configDir    string
	scope        Scope
	rootArgs     []string
	start        time.Time
	aiMode       bool
	cfg          *Config
	pluginAction *PluginManagementEvent
}

var activeExecution *activeCommandExecution
var executionEnded bool

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

func commandExecutionDurationMs(execution *activeCommandExecution) int64 {
	if execution == nil {
		return 0
	}
	return time.Since(execution.start).Milliseconds()
}
