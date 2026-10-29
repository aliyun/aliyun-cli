package ossutil

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func ossutilTestArchive(t *testing.T, center, payload string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	name := "ossutil"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	f, err := w.CreateHeader(&zip.FileHeader{Name: center + "/" + name, Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInstallOwnsTemporaryDirectory(t *testing.T) {
	for _, result := range []string{"success", "failure", "already removed"} {
		t.Run(result, func(t *testing.T) {
			installationRoot := t.TempDir()
			for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(key, installationRoot)
			}
			original := downloadAndUnzipFunc
			t.Cleanup(func() { downloadAndUnzipFunc = original })
			var directories []string
			downloadAndUnzipFunc = func(_, archive, _, _ string) error {
				dir := filepath.Dir(archive)
				relative, err := filepath.Rel(installationRoot, dir)
				if err != nil || relative == "." || !filepath.IsLocal(relative) {
					t.Fatalf("installation directory is outside the test-owned root: %s", dir)
				}
				if err := os.WriteFile(archive, []byte("partial download"), 0600); err != nil {
					t.Fatal(err)
				}
				directories = append(directories, dir)
				if result == "failure" {
					return fmt.Errorf("download interrupted")
				}
				if result == "already removed" {
					return os.RemoveAll(dir)
				}
				return nil
			}
			for range 2 {
				c := &Context{versionRemote: "2.4.0", downloadPathSuffix: "linux-amd64.zip"}
				err := c.Install()
				if result == "failure" {
					if err == nil || !strings.Contains(err.Error(), "download interrupted") {
						t.Fatalf("original download error lost: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
			if directories[0] == directories[1] {
				t.Fatalf("installations shared temporary directory %s", directories[0])
			}
			for _, dir := range directories {
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("temporary directory was not removed: %s: %v", dir, err)
				}
			}
		})
	}
}

func TestInstallRejectsUnusableTemporaryRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, file)
	}
	original := downloadAndUnzipFunc
	t.Cleanup(func() { downloadAndUnzipFunc = original })
	downloadAndUnzipFunc = func(_, _, _, _ string) error {
		t.Fatal("download attempted without a usable temporary directory")
		return nil
	}
	err := (&Context{}).Install()
	if err == nil || !strings.Contains(err.Error(), "failed to create ossutil temp dir") {
		t.Fatalf("expected temporary directory error, got %v", err)
	}
	data, readErr := os.ReadFile(file)
	if readErr != nil || string(data) != "keep" {
		t.Fatalf("temporary root file changed: %q, %v", data, readErr)
	}
}

func TestInstallPreservesDownloadErrorWhenCleanupFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix directory permissions without root privileges")
	}
	root := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, root)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0700); err != nil {
			t.Error(err)
		}
	})
	original := downloadAndUnzipFunc
	t.Cleanup(func() { downloadAndUnzipFunc = original })
	downloadErr := errors.New("download interrupted")
	downloadAndUnzipFunc = func(_, _, _, _ string) error {
		if err := os.Chmod(root, 0500); err != nil {
			t.Fatal(err)
		}
		return downloadErr
	}
	err := (&Context{}).Install()
	if !errors.Is(err, downloadErr) || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected both download and cleanup errors, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to remove temp dir") {
		t.Fatalf("cleanup error missing context: %v", err)
	}
}

type ossutilReadFunc func([]byte) (int, error)

func (read ossutilReadFunc) Read(p []byte) (int, error) { return read(p) }

func TestDownloadAndUnzipReportsDirectoryAndCleanupFailures(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix directory permissions without root privileges")
	}
	root := t.TempDir()
	t.Cleanup(func() {
		if err := os.Chmod(root, 0700); err != nil {
			t.Error(err)
		}
	})
	executable := filepath.Join(root, "ossutil")
	if err := os.WriteFile(executable, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	original := httpGetFunc
	t.Cleanup(func() { httpGetFunc = original })
	archive := bytes.NewReader(ossutilTestArchive(t, "center", "new binary"))
	httpGetFunc = func(string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(ossutilReadFunc(func(p []byte) (int, error) {
			n, err := archive.Read(p)
			if err == io.EOF {
				if chmodErr := os.Chmod(root, 0500); chmodErr != nil {
					t.Fatal(chmodErr)
				}
			}
			return n, err
		}))}, nil
	}
	err := DownloadAndUnzip("http://example/ossutil.zip", filepath.Join(root, "download.zip"), executable, "center")
	if err == nil || !strings.Contains(err.Error(), "failed to create unzip dir") || !strings.Contains(err.Error(), "failed to remove temp path") {
		t.Fatalf("expected both directory creation and cleanup errors, got %v", err)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("permission error lost: %v", err)
	}
	data, err := os.ReadFile(executable)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing binary changed: %q, %v", data, err)
	}
}

func TestDownloadAndUnzipPreservesDestinationOnPublishFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wantErr string
	}{
		{"missing target parent", "failed to create staged ossutil"},
		{"target directory", "failed to replace ossutil"},
		{"directory executable", "failed to stage ossutil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			executable := filepath.Join(root, "ossutil")
			archive := ossutilTestArchive(t, "center", "new binary")
			marker := executable
			switch tc.name {
			case "missing target parent":
				executable = filepath.Join(root, "missing", "ossutil")
				marker = ""
			case "target directory":
				if err := os.Mkdir(executable, 0700); err != nil {
					t.Fatal(err)
				}
				marker = filepath.Join(executable, "marker")
			case "directory executable":
				name := "ossutil/"
				if runtime.GOOS == "windows" {
					name = "ossutil.exe/"
				}
				var err error
				archive, err = os.ReadFile(createZipWithEntries(t, map[string]string{"center/" + name: ""}))
				if err != nil {
					t.Fatal(err)
				}
			}
			if marker != "" {
				if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(archive)
			}))
			defer server.Close()
			err := DownloadAndUnzip(server.URL, filepath.Join(root, "download.zip"), executable, "center")
			if runtime.GOOS == "windows" && tc.name == "directory executable" && err != nil && strings.Contains(err.Error(), "failed to open extracted file") {
				tc.wantErr = "failed to open extracted file"
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q, got %v", tc.wantErr, err)
			}
			if marker != "" {
				data, err := os.ReadFile(marker)
				if err != nil || string(data) != "keep" {
					t.Fatalf("existing destination changed: %q, %v", data, err)
				}
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if marker == "" {
				if len(entries) != 0 {
					t.Fatalf("failed download left files: %v", entries)
				}
			} else if len(entries) != 1 || entries[0].Name() != "ossutil" {
				t.Fatalf("failed download left temporary files: %v", entries)
			}
		})
	}
}

func TestDownloadAndUnzipPreservesBinaryOnChecksumError(t *testing.T) {
	archive := ossutilTestArchive(t, "center", "new binary contents")
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	offset, err := r.File[0].DataOffset()
	if err != nil {
		t.Fatal(err)
	}
	archive[offset] ^= 0xff
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	tmp := t.TempDir()
	executable := filepath.Join(tmp, "ossutil")
	if err := os.WriteFile(executable, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}
	err = DownloadAndUnzip(server.URL, filepath.Join(tmp, "download.zip"), executable, "center")
	if err == nil || !strings.Contains(err.Error(), zip.ErrChecksum.Error()) {
		t.Fatalf("expected checksum error, got %v", err)
	}
	data, err := os.ReadFile(executable)
	if err != nil || string(data) != "old binary" {
		t.Fatalf("failed download changed existing binary: %q, %v", data, err)
	}
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "ossutil" {
		t.Fatalf("download left temporary files: %v", entries)
	}
}

func TestUnzipReturnsCreateError(t *testing.T) {
	archive := createZipWithEntries(t, map[string]string{"file.txt": "contents"})
	dest := t.TempDir()
	if err := os.Mkdir(filepath.Join(dest, "file.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := unzip(archive, dest); err == nil {
		t.Fatal("creating a file over an existing directory must fail")
	}
}

func TestUnzipRejectsOutsidePaths(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{"parent", "../outside.txt", true},
		{"absolute", filepath.Join(root, "absolute.txt"), true},
		{"valid", "inside/file.txt", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := filepath.Join(root, tc.name, "extract")
			archive := createZipWithEntries(t, map[string]string{tc.entry: "contents"})
			err := unzip(archive, dest)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "invalid file path") {
					t.Fatalf("expected invalid path error, got %v", err)
				}
				outside := tc.entry
				if !filepath.IsAbs(outside) {
					outside = filepath.Join(dest, outside)
				}
				if _, err := os.Stat(outside); !os.IsNotExist(err) {
					t.Fatalf("archive wrote outside destination: %s: %v", outside, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(dest, tc.entry))
				if err != nil || string(data) != "contents" {
					t.Fatalf("valid file was not extracted: %q, %v", data, err)
				}
			}
		})
	}
}

func TestDownloadAndUnzipConcurrent(t *testing.T) {
	for _, sameTarget := range []bool{false, true} {
		t.Run(fmt.Sprintf("same_target_%t", sameTarget), func(t *testing.T) {
			const count = 4
			archives := make(map[string][]byte, count)
			payloads := make([]string, count)
			for i := range count {
				payloads[i] = strings.Repeat(fmt.Sprintf("binary-%d\n", i), 8192)
				archives[fmt.Sprintf("/%d", i)] = ossutilTestArchive(t, fmt.Sprintf("center-%d", i), payloads[i])
			}
			var requests sync.WaitGroup
			requests.Add(count)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Done()
				requests.Wait()
				_, _ = w.Write(archives[r.URL.Path])
			}))
			defer server.Close()
			tmp := t.TempDir()
			results := make(chan error, count)
			for i := range count {
				go func() {
					target := filepath.Join(tmp, fmt.Sprintf("ossutil-%d", i))
					if sameTarget {
						target = filepath.Join(tmp, "ossutil")
					}
					results <- DownloadAndUnzip(fmt.Sprintf("%s/%d", server.URL, i), filepath.Join(tmp, fmt.Sprintf("download-%d.zip", i)), target, fmt.Sprintf("center-%d", i))
				}()
			}
			succeeded := 0
			for range count {
				if err := <-results; err != nil {
					// Windows may deny replacing a destination held by a concurrent rename.
					var renameErr *os.LinkError
					if sameTarget && runtime.GOOS == "windows" && errors.As(err, &renameErr) &&
						renameErr.Op == "rename" && renameErr.New == filepath.Join(tmp, "ossutil") && errors.Is(renameErr, os.ErrPermission) {
						continue
					}
					t.Error(err)
				} else {
					succeeded++
				}
			}
			if succeeded == 0 {
				t.Fatal("no concurrent installation published a binary")
			}
			if t.Failed() {
				return
			}
			if sameTarget {
				data, err := os.ReadFile(filepath.Join(tmp, "ossutil"))
				if err != nil {
					t.Fatal(err)
				}
				complete := false
				for _, payload := range payloads {
					complete = complete || string(data) == payload
				}
				if !complete {
					t.Fatal("concurrent installation published an incomplete binary")
				}
			} else {
				for i, payload := range payloads {
					data, err := os.ReadFile(filepath.Join(tmp, fmt.Sprintf("ossutil-%d", i)))
					if err != nil || string(data) != payload {
						t.Fatalf("binary %d: contents mismatch or read error: %v", i, err)
					}
				}
			}
			entries, err := os.ReadDir(tmp)
			if err != nil {
				t.Fatal(err)
			}
			wantEntries := count
			if sameTarget {
				wantEntries = 1
			}
			if len(entries) != wantEntries {
				t.Fatalf("download left temporary files: %v", entries)
			}
		})
	}
}
