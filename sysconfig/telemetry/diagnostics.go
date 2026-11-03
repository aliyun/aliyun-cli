// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	TelemetryLogDisableEnv = "ALIBABA_CLOUD_CLI_TELEMETRY_LOG_DISABLE"
	diagnosticMaxBytes     = 10 * 1024 * 1024
	diagnosticBackups      = 5
	diagnosticLogName      = "telemetry.log"
)

var diagnosticMu sync.Mutex

// diagnosticRecord deliberately accepts no free-form messages, errors, command
// arguments, or paths. Every string field is validated again before writing.
type diagnosticRecord struct {
	Stage      string `json:"stage"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	ErrorKind  string `json:"errorKind,omitempty"`
	EventID    string `json:"eventId,omitempty"`
	Pipeline   string `json:"pipeline,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

// writeDiagnostic is independent of collection state. Logging must never affect
// a command's output, exit status, or the event being collected.
func writeDiagnostic(configDir string, record diagnosticRecord) {
	writeDiagnosticWithLimits(configDir, record, diagnosticMaxBytes, diagnosticBackups)
}

func writeDiagnosticWithLimits(configDir string, record diagnosticRecord, maxBytes int64, backups int) {
	defer func() { _ = recover() }()
	if envTruthy(TelemetryLogDisableEnv) {
		return
	}
	if configDir == "" || maxBytes <= 0 || backups < 1 || backups > diagnosticBackups {
		return
	}
	if !sanitizeDiagnostic(&record) {
		return
	}
	entry := struct {
		Timestamp string `json:"timestamp"`
		PID       int    `json:"pid"`
		diagnosticRecord
	}{time.Now().UTC().Format(time.RFC3339Nano), os.Getpid(), record}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')
	if int64(len(data)) > maxBytes {
		return
	}

	diagnosticMu.Lock()
	defer diagnosticMu.Unlock()
	logDir := filepath.Join(configDir, "logs")
	if !prepareDiagnosticDirectory(logDir) {
		return
	}
	path := filepath.Join(logDir, diagnosticLogName)
	lockPath := path + ".lock"
	if _, err := diagnosticRegularFile(lockPath); err != nil {
		return
	}
	unlock, err := acquireFileLock(lockPath, 5*time.Millisecond)
	if err != nil {
		return
	}
	defer unlock()
	if err := os.Chmod(lockPath, 0600); err != nil {
		return
	}
	// Check every rotation target under the same stable lock, even when this
	// append will not rotate. In particular, never follow existing symlinks.
	for i := 1; i <= backups; i++ {
		if _, err := diagnosticRegularFile(path + "." + strconv.Itoa(i)); err != nil {
			return
		}
	}
	info, err := diagnosticRegularFile(path)
	if err != nil {
		return
	}
	if info != nil {
		if err := os.Chmod(path, 0600); err != nil {
			return
		}
		if info.Size() > maxBytes-int64(len(data)) {
			if !rotateDiagnostic(path, backups) {
				return
			}
		}
	}
	// No handle to the log is open during rotation, including on Windows.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return
	}
	_, _ = f.Write(data)
}

func prepareDiagnosticDirectory(path string) bool {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0700); err != nil {
			return false
		}
		info, err = os.Lstat(path)
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return os.Chmod(path, 0700) == nil
}

func diagnosticRegularFile(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("unsafe diagnostic file type")
	}
	return info, nil
}

func rotateDiagnostic(path string, backups int) bool {
	if err := os.Remove(path + "." + strconv.Itoa(backups)); err != nil && !os.IsNotExist(err) {
		return false
	}
	for i := backups - 1; i >= 1; i-- {
		if err := os.Rename(path+"."+strconv.Itoa(i), path+"."+strconv.Itoa(i+1)); err != nil && !os.IsNotExist(err) {
			return false
		}
	}
	return os.Rename(path, path+".1") == nil
}

func sanitizeDiagnostic(record *diagnosticRecord) bool {
	switch record.Stage {
	case "collection", "event_build", "cache_write", "upload_spawn", "cache_read", "cache_remove", "upload":
	default:
		return false
	}
	switch record.Status {
	case "started", "succeeded", "failed", "skipped":
	default:
		return false
	}
	switch record.Reason {
	case "management_command", "invalid_config", "disabled", "scope_inactive", "instance_id_failed", "panic", "marshal_failed", "invalid_cache_path", "cache_empty", "invalid_event", "pipeline_mismatch", "executable_unavailable", "process_start_failed", "request_invalid", "http_error", "missing_arguments", "invalid_pipeline":
	default:
		record.Reason = ""
	}
	switch record.ErrorKind {
	case "permission_denied", "not_found", "lock_busy", "timeout", "network_error", "invalid_json", "io_error", "other":
	default:
		record.ErrorKind = ""
	}
	if !validDiagnosticEventID(record.EventID) {
		record.EventID = ""
	}
	if record.Pipeline != "public" && record.Pipeline != "inner" {
		record.Pipeline = ""
	}
	if record.HTTPStatus < 100 || record.HTTPStatus > 599 {
		record.HTTPStatus = 0
	}
	if record.DurationMs < 0 {
		record.DurationMs = 0
	}
	return true
}

func validDiagnosticEventID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range []byte(id) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// diagnosticErrorKind classifies by type and sentinel identity, never by text.
func diagnosticErrorKind(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, os.ErrPermission):
		return "permission_denied"
	case errors.Is(err, os.ErrNotExist):
		return "not_found"
	case errors.Is(err, ErrLockBusy):
		return "lock_busy"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		if networkError.Timeout() {
			return "timeout"
		}
		return "network_error"
	}
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &syntaxError) || errors.As(err, &typeError) {
		return "invalid_json"
	}
	var pathError *os.PathError
	var linkError *os.LinkError
	if errors.As(err, &pathError) || errors.As(err, &linkError) || errors.Is(err, io.ErrShortWrite) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "io_error"
	}
	return "other"
}
