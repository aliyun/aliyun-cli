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
	"fmt"
	"os"
	"path/filepath"
)

// pathReplacement swaps a staged file or directory into its final path while retaining the previous path until the caller commits.
// It guarantees rollback for errors returned within the current process;
// it does not provide crash recovery or cross-process serialization for now.
type pathReplacement struct {
	finalPath    string
	backupParent string
	backupPath   string
	hadExisting  bool
}

func replacePathWithRollback(stagedPath, finalPath, backupPattern string) (*pathReplacement, error) {
	replacement := &pathReplacement{finalPath: finalPath}
	if _, err := os.Lstat(finalPath); err == nil {
		replacement.hadExisting = true
		replacement.backupParent, err = os.MkdirTemp(filepath.Dir(finalPath), backupPattern)
		if err != nil {
			return nil, fmt.Errorf("create rollback directory: %w", err)
		}
		replacement.backupPath = filepath.Join(replacement.backupParent, "previous")
		if err := os.Rename(finalPath, replacement.backupPath); err != nil {
			_ = os.RemoveAll(replacement.backupParent)
			return nil, fmt.Errorf("backup existing path: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect existing path: %w", err)
	}

	if err := os.Rename(stagedPath, finalPath); err != nil {
		restoreErr := replacement.restorePrevious()
		if restoreErr != nil {
			return nil, fmt.Errorf("replace final path: %w; restore previous path: %v", err, restoreErr)
		}
		replacement.commit()
		return nil, fmt.Errorf("replace final path: %w", err)
	}
	return replacement, nil
}

func (r *pathReplacement) restorePrevious() error {
	if r == nil {
		return nil
	}
	if err := os.RemoveAll(r.finalPath); err != nil {
		return err
	}
	if r.hadExisting {
		if err := os.Rename(r.backupPath, r.finalPath); err != nil {
			return err
		}
	}
	return nil
}

func (r *pathReplacement) rollback() error {
	if err := r.restorePrevious(); err != nil {
		return err
	}
	r.commit()
	return nil
}

func (r *pathReplacement) commit() {
	if r != nil && r.backupParent != "" {
		_ = os.RemoveAll(r.backupParent)
	}
}
