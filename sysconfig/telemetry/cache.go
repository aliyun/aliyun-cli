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
	"strconv"
	"strings"
	"time"
)

const (
	cacheRetentionDaysEnv     = "ALIYUN_CLI_TELEMETRY_CACHE_RETENTION_DAYS"
	defaultCacheRetentionDays = 7
	maxCacheRetentionDays     = 3650
)

func WriteEvent(cacheDir string, ev Event) (string, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	now := time.Now()
	cleanupExpiredEventFiles(cacheDir, now, cacheRetention())
	path := filepath.Join(cacheDir, eventCacheFileName(now, ev.EventID))
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

func eventCacheFileName(createdAt time.Time, eventID string) string {
	timestamp := createdAt.UTC().Format("20060102T150405.000Z")
	return "event-" + timestamp + "-" + eventID + ".ndjson"
}

func cacheRetention() time.Duration {
	days := defaultCacheRetentionDays
	if value := strings.TrimSpace(os.Getenv(cacheRetentionDaysEnv)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil &&
			parsed >= 1 &&
			parsed <= maxCacheRetentionDays {
			days = parsed
		}
	}
	return time.Duration(days) * 24 * time.Hour
}

func cleanupExpiredEventFiles(
	cacheDir string,
	now time.Time,
	retention time.Duration,
) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	cutoff := now.Add(-retention)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() ||
			!strings.HasPrefix(name, "event-") ||
			!strings.HasSuffix(name, ".ndjson") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(cacheDir, name))
	}
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
