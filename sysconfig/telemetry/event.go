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
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/util"
)

const schemaVersion = "1"

type Event struct {
	Schema       string   `json:"schema"`
	EventID      string   `json:"event_id"`
	Timestamp    string   `json:"timestamp"`
	Pipeline     string   `json:"pipeline"`
	InnerTrigger string   `json:"inner_trigger,omitempty"`
	Command      string   `json:"command"`
	RawCommand   string   `json:"raw_command"`
	Params       []string `json:"params,omitempty"`
	Result       string   `json:"result"`
	ErrorType    string   `json:"error_type,omitempty"`
	ErrorSummary string   `json:"error_summary,omitempty"`
	DurationMs   int64    `json:"duration_ms"`
	Plugin       string   `json:"plugin,omitempty"`
	CLIVersion   string   `json:"cli_version"`
	OS           string   `json:"os"`
	Arch         string   `json:"arch"`
	GoVersion    string   `json:"go_version"`
	RegionID     string   `json:"region_id,omitempty"`
	Mode         string   `json:"mode"`
	AgentName    string   `json:"agent_name,omitempty"`
	Vendor       string   `json:"vendor,omitempty"`
	InstanceID   string   `json:"instance_id"`
}

type BuildInput struct {
	Scope      InnerScope
	RootArgs   []string
	Profile    ProfileSnapshot
	InstanceID string
	DurationMs int64
	Err        error
	ExitCode   *int
	AIMode     bool
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

func errorSummary(err error, exitCode *int) string {
	if err != nil {
		return SanitizeSummary(err.Error())
	}
	if exitCode != nil && *exitCode != 0 {
		return SanitizeSummary(fmt.Sprintf("exit status %d", *exitCode))
	}
	return ""
}

func BuildEvent(in BuildInput) (Event, error) {
	eventID, err := newUUID()
	if err != nil {
		return Event{}, err
	}

	pos := positionalParts(in.RootArgs)
	command := normalizeCommand(pos)
	raw := strings.Join(pos, " ")

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

	return Event{
		Schema:       schemaVersion,
		EventID:      eventID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Pipeline:     "inner",
		InnerTrigger: in.Scope.Trigger,
		Command:      command,
		RawCommand:   raw,
		Params:       ExtractParamNames(in.RootArgs),
		Result:       classifyResult(in.Err, in.ExitCode),
		ErrorType:    errorTypeName(in.Err, in.ExitCode),
		ErrorSummary: errorSummary(in.Err, in.ExitCode),
		DurationMs:   in.DurationMs,
		Plugin:       pluginField,
		CLIVersion:   cli.GetVersion(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		RegionID:     strings.TrimSpace(in.Profile.RegionID),
		Mode:         mode,
		AgentName:    util.DetectAgentName(),
		Vendor:       sanitizeVendor(os.Getenv("ALIBABA_CLOUD_VENDOR")),
		InstanceID:   in.InstanceID,
	}, nil
}

func sanitizeVendor(v string) string {
	v = util.SanitizeUserAgent(strings.TrimSpace(v))
	if len(v) > 64 {
		v = v[:64]
	}
	return v
}
