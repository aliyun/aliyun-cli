// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	InstanceIDFileName = "telemetry-instance-id"
	instanceIDLockName = "telemetry-instance-id.lock"
	purgeLockTimeout   = 2 * time.Second
)

func GetInstanceIDFilePath(configDir string) string {
	return filepath.Join(configDir, InstanceIDFileName)
}

// LoadInstanceID only reads the ID; status commands must not create one.
func LoadInstanceID(configDir string) (string, error) {
	data, err := os.ReadFile(GetInstanceIDFilePath(configDir))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(data))
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return "", fmt.Errorf("invalid telemetry instance ID; run 'aliyun configure telemetry purge' to reset it")
	}
	decoded, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	if err != nil || len(decoded) != 16 {
		return "", fmt.Errorf("invalid telemetry instance ID; run 'aliyun configure telemetry purge' to reset it")
	}
	return id, nil
}

func EnsureInstanceID(configDir string) (string, error) {
	// Steady-state commands only read the ID, without writing or taking a lock.
	if id, err := LoadInstanceID(configDir); err != nil || id != "" {
		return id, err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	// Collection must not wait behind another CLI process. flush skips the
	// event on ErrLockBusy; a later invocation will reuse the published ID.
	unlock, err := acquireFileLock(filepath.Join(configDir, instanceIDLockName), 0)
	if err != nil {
		return "", err
	}
	defer unlock()

	// Another process may have initialized the ID between our read and lock.
	if id, err := LoadInstanceID(configDir); err != nil || id != "" {
		return id, err
	}
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	if err := atomicWriteFile(GetInstanceIDFilePath(configDir), []byte(id+"\n"), 0600); err != nil {
		return "", err
	}
	return id, nil
}

// Purge resets local state without changing the enabled preference. It does
// not cancel commands or uploads already in flight.
func Purge(configDir string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	unlock, err := acquireFileLock(filepath.Join(configDir, instanceIDLockName), purgeLockTimeout)
	if err != nil {
		return err
	}
	defer unlock()
	if err := os.RemoveAll(filepath.Join(configDir, CacheDirName)); err != nil {
		return err
	}
	if err := os.Remove(GetInstanceIDFilePath(configDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Keep the lock file: deleting it could let two processes lock different
	// files at the same path. It lives outside the removable event cache.
	return nil
}
