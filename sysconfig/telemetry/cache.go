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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func WriteEvent(cacheDir string, ev Event) (string, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(cacheDir, "event-"+ev.EventID+".ndjson")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()

	line, err := json.Marshal(ev)
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

type CacheStats struct {
	Files      int
	TotalBytes int64
	Lines      int
}

func InnerCacheStats(cacheDir string) CacheStats {
	var stats CacheStats
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return stats
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".ndjson") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		stats.Files++
		stats.TotalBytes += info.Size()
	}
	return stats
}
