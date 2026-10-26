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
	"errors"
	"reflect"
	"runtime"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/util"
)

const schemaVersion = "1"

type Event struct {
	Schema       string                 `json:"schema"`
	EventID      string                 `json:"event_id"`
	EventType    string                 `json:"event_type"`
	Timestamp    string                 `json:"timestamp"`
	Pipeline     string                 `json:"pipeline"`
	InnerTrigger string                 `json:"inner_trigger,omitempty"`
	Command      string                 `json:"command"`
	Params       []string               `json:"params,omitempty"`
	Result       string                 `json:"result"`
	ErrorType    string                 `json:"error_type,omitempty"`
	DurationMs   int64                  `json:"duration_ms"`
	Plugin       string                 `json:"plugin,omitempty"`
	CLIVersion   string                 `json:"cli_version"`
	OS           string                 `json:"os"`
	Arch         string                 `json:"arch"`
	GoVersion    string                 `json:"go_version"`
	Mode         string                 `json:"mode"`
	AgentName    string                 `json:"agent_name,omitempty"`
	InstanceID   string                 `json:"instance_id"`
	PluginAction *PluginManagementEvent `json:"plugin_action,omitempty"`
}

type BuildInput struct {
	Scope        Scope
	RootArgs     []string
	InstanceID   string
	DurationMs   int64
	Err          error
	ExitCode     *int
	AIMode       bool
	PluginAction *PluginManagementEvent
}

func classifyResult(err error, exitCode *int) string {
	if err == nil && (exitCode == nil || *exitCode == 0) {
		return "Success"
	}
	if exitCode != nil && *exitCode != 0 {
		return "Failure"
	}
	var suggest cli.SuggestibleError
	if errors.As(err, &suggest) {
		return "UserFault"
	}
	return "Failure"
}

func errorTypeName(err error, exitCode *int) string {
	if err != nil {
		t := reflect.TypeOf(err)
		if t != nil {
			return t.String()
		}
	}
	if exitCode != nil && *exitCode != 0 {
		return "exec.ExitError"
	}
	return ""
}

func BuildEvent(in BuildInput) (Event, error) {
	eventID, err := newUUID()
	if err != nil {
		return Event{}, err
	}

	pos := commandParts(in.RootArgs, in.Scope.CommandToken)
	command := normalizeCommand(pos)

	mode := "default"
	if in.AIMode {
		mode = "ai"
	}

	pluginField := ""
	if in.Scope.PluginName != "" {
		ver := in.Scope.PluginVer
		if ver == "" {
			ver = "unknown"
		}
		pluginField = in.Scope.PluginName + "@" + ver
	}
	eventType := "cli.command.completed"
	if in.PluginAction != nil {
		eventType = "cli.plugin.management.completed"
	}
	innerTrigger := ""
	if in.Scope.Pipeline == PipelineInner {
		innerTrigger = in.Scope.Trigger
	}

	return Event{
		Schema:       schemaVersion,
		EventID:      eventID,
		EventType:    eventType,
		Timestamp:    time.Now().Format(time.RFC3339),
		Pipeline:     in.Scope.Pipeline,
		InnerTrigger: innerTrigger,
		Command:      command,
		Params:       ExtractParamNames(in.RootArgs),
		Result:       classifyResult(in.Err, in.ExitCode),
		ErrorType:    errorTypeName(in.Err, in.ExitCode),
		DurationMs:   in.DurationMs,
		Plugin:       pluginField,
		CLIVersion:   cli.GetVersion(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		Mode:         mode,
		AgentName:    util.DetectAgentName(),
		InstanceID:   in.InstanceID,
		PluginAction: in.PluginAction,
	}, nil
}
