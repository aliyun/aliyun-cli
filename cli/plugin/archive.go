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

package plugin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxPluginArchiveBytes   int64 = 64 << 20
	maxPluginFileBytes      int64 = 128 << 20
	maxPluginExtractedBytes int64 = 256 << 20
	maxPluginArchiveEntries       = 10_000
)

type pluginArchiveLimits struct {
	archiveBytes   int64
	fileBytes      int64
	extractedBytes int64
	entries        int
}

var defaultPluginArchiveLimits = pluginArchiveLimits{
	archiveBytes:   maxPluginArchiveBytes,
	fileBytes:      maxPluginFileBytes,
	extractedBytes: maxPluginExtractedBytes,
	entries:        maxPluginArchiveEntries,
}

func downloadFile(url, dest string) error {
	return downloadFileWithLimit(url, dest, defaultPluginArchiveLimits.archiveBytes)
}

func downloadFileWithLimit(url, dest string, limit int64) error {
	resp, err := httpGet(url, pluginArchiveDLTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}
	return writePluginArchiveResponse(resp, dest, limit)
}

func writePluginArchiveResponse(resp *http.Response, dest string, limit int64) error {
	if resp.ContentLength > limit {
		return fmt.Errorf("plugin archive download size %d exceeds limit of %d bytes", resp.ContentLength, limit)
	}
	return writeReaderWithLimit(resp.Body, dest, limit)
}

func writeReaderWithLimit(reader io.Reader, dest string, limit int64) (err error) {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
			keep = false
		}
		if !keep {
			_ = os.Remove(dest)
		}
	}()

	written, err := io.Copy(out, io.LimitReader(reader, limit+1))
	if err != nil {
		return err
	}
	if written > limit {
		return fmt.Errorf("plugin archive download exceeds limit of %d bytes", limit)
	}
	keep = true
	return nil
}

func validatePluginArchiveSize(src string, limit int64) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("plugin archive is a directory: %s", src)
	}
	if info.Size() > limit {
		return fmt.Errorf("plugin archive size %d exceeds limit of %d bytes", info.Size(), limit)
	}
	return nil
}

func untar(src, dest string) error {
	return untarWithLimits(src, dest, defaultPluginArchiveLimits)
}

func untarWithLimits(src, dest string, limits pluginArchiveLimits) error {
	if err := validatePluginArchiveSize(src, limits.archiveBytes); err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var extracted int64
	entries := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		entries++
		if entries > limits.entries {
			return fmt.Errorf("plugin archive contains more than %d entries", limits.entries)
		}

		target, err := safeArchiveTarget(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := validateExtractedFileSize(header.Name, header.Size, extracted, limits); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			written, err := writeExtractedFile(target, os.FileMode(header.Mode), tr, header.Size)
			if err != nil {
				return err
			}
			if written != header.Size {
				return fmt.Errorf("plugin archive file %q size changed during extraction", header.Name)
			}
			extracted += written
		}
	}
}

func unzip(src, dest string) error {
	return unzipWithLimits(src, dest, defaultPluginArchiveLimits)
}

func unzipWithLimits(src, dest string, limits pluginArchiveLimits) error {
	if err := validatePluginArchiveSize(src, limits.archiveBytes); err != nil {
		return err
	}
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) > limits.entries {
		return fmt.Errorf("plugin archive contains more than %d entries", limits.entries)
	}
	var extracted int64
	for _, file := range r.File {
		target, err := safeArchiveTarget(dest, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if file.UncompressedSize64 > uint64(^uint64(0)>>1) {
			return fmt.Errorf("plugin archive file %q has unsupported size", file.Name)
		}
		size := int64(file.UncompressedSize64)
		if err := validateExtractedFileSize(file.Name, size, extracted, limits); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		written, writeErr := writeExtractedFile(target, file.Mode(), rc, minInt64(limits.fileBytes, limits.extractedBytes-extracted))
		closeErr := rc.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written != size {
			return fmt.Errorf("plugin archive file %q size changed during extraction", file.Name)
		}
		extracted += written
	}
	return nil
}

func validateExtractedFileSize(name string, size, extracted int64, limits pluginArchiveLimits) error {
	if size < 0 {
		return fmt.Errorf("plugin archive file %q has invalid size %d", name, size)
	}
	if size > limits.fileBytes {
		return fmt.Errorf("plugin archive file %q size %d exceeds per-file limit of %d bytes", name, size, limits.fileBytes)
	}
	if size > limits.extractedBytes-extracted {
		return fmt.Errorf("plugin archive extracted size exceeds limit of %d bytes", limits.extractedBytes)
	}
	return nil
}

func writeExtractedFile(target string, mode os.FileMode, reader io.Reader, limit int64) (written int64, err error) {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return 0, err
	}
	keep := false
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
			keep = false
		}
		if !keep {
			_ = os.Remove(target)
		}
	}()

	written, err = io.Copy(out, io.LimitReader(reader, limit+1))
	if err != nil {
		return written, err
	}
	if written > limit {
		return written, fmt.Errorf("plugin archive file %q exceeds extraction limit of %d bytes", target, limit)
	}
	keep = true
	return written, nil
}

func safeArchiveTarget(dest, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("illegal absolute path in archive: %s", name)
	}
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("illegal path with '..' in archive: %s", name)
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return "", fmt.Errorf("illegal path starting with separator in archive: %s", name)
	}

	target := filepath.Clean(filepath.Join(dest, name))
	destPath := filepath.Clean(dest) + string(os.PathSeparator)
	if !strings.HasPrefix(target, destPath) {
		return "", fmt.Errorf("illegal file path in archive: %s", name)
	}
	return target, nil
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func (m *Manager) effectiveArchiveLimits() pluginArchiveLimits {
	if m != nil && m.archiveLimits != nil {
		return *m.archiveLimits
	}
	return defaultPluginArchiveLimits
}
