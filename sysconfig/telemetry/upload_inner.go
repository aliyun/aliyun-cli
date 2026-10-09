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
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultInnerEndpoint = "https://telemetry-inner.aliyun-inc.com/v1/telemetry/events"

type innerUploadEvent struct {
	SchemaVersion string                `json:"schemaVersion"`
	EventID       string                `json:"eventId"`
	EventType     string                `json:"eventType"`
	Timestamp     string                `json:"timestamp"`
	ClientName    string                `json:"clientName"`
	ClientVersion string                `json:"clientVersion"`
	InstanceID    string                `json:"instanceId"`
	Status        string                `json:"status"`
	DurationMs    int64                 `json:"durationMs"`
	Attributes    innerUploadAttributes `json:"attributes"`
}

type innerUploadAttributes struct {
	Pipeline       string   `json:"pipeline"`
	InnerTrigger   string   `json:"innerTrigger,omitempty"`
	Command        string   `json:"command"`
	ParameterNames []string `json:"parameterNames,omitempty"`
	ErrorType      string   `json:"errorType,omitempty"`
	Plugin         string   `json:"plugin,omitempty"`
	OS             string   `json:"os"`
	Arch           string   `json:"arch"`
	GoVersion      string   `json:"goVersion"`
	RegionID       string   `json:"regionId,omitempty"`
	Mode           string   `json:"mode"`
	AgentName      string   `json:"agentName,omitempty"`
	Vendor         string   `json:"vendor,omitempty"`
}

func innerUploadEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("ALIYUN_CLI_TELEMETRY_INNER_ENDPOINT")); v != "" {
		return v
	}
	return defaultInnerEndpoint
}

func UploadInnerFile(configDir, cacheFile string) error {
	cacheDir := GetInnerCacheDir(configDir)
	absCacheDir, err := filepath.Abs(cacheDir)
	if err != nil {
		return nil
	}
	absCacheFile, err := filepath.Abs(cacheFile)
	if err != nil ||
		filepath.Dir(absCacheFile) != absCacheDir ||
		!strings.HasSuffix(absCacheFile, ".ndjson") {
		return nil
	}
	lines, readErr := readNDJSON(absCacheFile)
	_ = os.Remove(absCacheFile)
	if readErr != nil || len(lines) == 0 {
		return nil
	}

	var event Event
	if err := json.Unmarshal(lines[0], &event); err != nil {
		return nil
	}
	if !postInnerBatch([]innerUploadEvent{newInnerUploadEvent(event)}) {
		return nil
	}
	cfg, _ := Load(configDir)
	return MarkInnerUploaded(configDir, cfg)
}

func postInnerBatch(batch []innerUploadEvent) bool {
	body, err := json.Marshal(batch)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, innerUploadEndpoint(), bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}
	return true
}

func newInnerUploadEvent(event Event) innerUploadEvent {
	return innerUploadEvent{
		SchemaVersion: event.Schema,
		EventID:       event.EventID,
		EventType:     "cli.command.completed",
		Timestamp:     event.Timestamp,
		ClientName:    "aliyun-cli",
		ClientVersion: event.CLIVersion,
		InstanceID:    event.InstanceID,
		Status:        uploadStatus(event.Result),
		DurationMs:    event.DurationMs,
		Attributes: innerUploadAttributes{
			Pipeline:       event.Pipeline,
			InnerTrigger:   event.InnerTrigger,
			Command:        event.Command,
			ParameterNames: event.Params,
			ErrorType:      event.ErrorType,
			Plugin:         event.Plugin,
			OS:             event.OS,
			Arch:           event.Arch,
			GoVersion:      event.GoVersion,
			RegionID:       event.RegionID,
			Mode:           event.Mode,
			AgentName:      event.AgentName,
			Vendor:         event.Vendor,
		},
	}
}

func uploadStatus(result string) string {
	switch result {
	case "Success":
		return "success"
	case "UserFault":
		return "user_fault"
	default:
		return "failure"
	}
}

func readNDJSON(path string) ([]json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []json.RawMessage
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		out = append(out, append(json.RawMessage(nil), line...))
	}
	return out, sc.Err()
}
