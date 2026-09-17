package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathReplacementLifecycle(t *testing.T) {
	for _, existing := range []bool{false, true} {
		for _, rollback := range []bool{false, true} {
			t.Run(fmtReplacementName(existing, rollback), func(t *testing.T) {
				dir := t.TempDir()
				final := filepath.Join(dir, "final")
				staged := filepath.Join(dir, "staged")
				if existing {
					require.NoError(t, os.WriteFile(final, []byte("old"), 0600))
				}
				require.NoError(t, os.WriteFile(staged, []byte("new"), 0600))
				r, err := replacePathWithRollback(staged, final, "backup-*")
				require.NoError(t, err)
				data, err := os.ReadFile(final)
				require.NoError(t, err)
				require.Equal(t, "new", string(data))
				if rollback {
					require.NoError(t, r.rollback())
					if existing {
						data, err = os.ReadFile(final)
						require.NoError(t, err)
						require.Equal(t, "old", string(data))
					} else {
						require.NoFileExists(t, final)
					}
				} else {
					r.commit()
					data, err = os.ReadFile(final)
					require.NoError(t, err)
					require.Equal(t, "new", string(data))
				}
				if r.backupParent != "" {
					require.NoDirExists(t, r.backupParent)
				}
			})
		}
	}
	var r *pathReplacement
	require.NoError(t, r.rollback())
	r.commit()
}
func fmtReplacementName(existing, rollback bool) string {
	if existing {
		if rollback {
			return "replace and rollback"
		}
		return "replace and commit"
	}
	if rollback {
		return "create and rollback"
	}
	return "create and commit"
}

func TestPathReplacementFailedStageRestoresOriginal(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "final")
	require.NoError(t, os.WriteFile(final, []byte("original"), 0600))
	r, err := replacePathWithRollback(filepath.Join(dir, "missing"), final, "backup-*")
	require.Nil(t, r)
	require.ErrorContains(t, err, "replace final path")
	data, err := os.ReadFile(final)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	_, err = replacePathWithRollback(filepath.Join(dir, "missing"), final, "invalid/pattern")
	require.ErrorContains(t, err, "create rollback directory")
}

func TestPathReplacementRollbackRetainsBackupOnRestoreFailure(t *testing.T) {
	dir := t.TempDir()
	backup := filepath.Join(dir, "backup")
	require.NoError(t, os.Mkdir(backup, 0700))
	r := &pathReplacement{finalPath: filepath.Join(dir, "final"), backupParent: backup, backupPath: filepath.Join(backup, "missing"), hadExisting: true}
	require.Error(t, r.rollback())
	require.DirExists(t, backup)
}
