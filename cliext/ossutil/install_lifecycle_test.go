package ossutil

import (
	"archive/zip"
	"bytes"
	"fmt"
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
			for range count {
				if err := <-results; err != nil {
					t.Error(err)
				}
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
