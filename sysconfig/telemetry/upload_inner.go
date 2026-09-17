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
	"strings"
	"time"
)

const defaultInnerEndpoint = "https://telemetry-inner.aliyun-inc.com/v1/events"

func innerUploadEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("ALIYUN_CLI_TELEMETRY_INNER_ENDPOINT")); v != "" {
		return v
	}
	return defaultInnerEndpoint
}

func UploadInner(configDir string) error {
	cacheDir := GetInnerCacheDir(configDir)
	files, err := ListNDJSONFiles(cacheDir)
	if err != nil || len(files) == 0 {
		return nil
	}

	var batch []json.RawMessage
	var uploaded []string
	for _, path := range files {
		lines, err := readNDJSON(path)
		if err != nil {
			continue
		}
		if len(lines) == 0 {
			continue
		}
		batch = append(batch, lines...)
		uploaded = append(uploaded, path)
	}
	if len(batch) == 0 {
		return nil
	}

	body, err := json.Marshal(batch)
	if err != nil {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, innerUploadEndpoint(), bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	RemoveFiles(uploaded)
	cfg, _ := Load(configDir)
	return MarkInnerUploaded(configDir, cfg)
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
