package saectl

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func installZip(t *testing.T, binary string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range []string{"folder/", "README", "folder/" + binary} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/") {
			if _, err := f.Write([]byte("binary")); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

type zipBodyError struct{}

func (zipBodyError) Read([]byte) (int, error) { return 0, errors.New("body failed") }
func (zipBodyError) Close() error             { return nil }

func TestZipDownloadAndInstallFailurePaths(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	binary := "saectl"
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	for _, tc := range []struct {
		name      string
		status    int
		data      []byte
		body      io.ReadCloser
		reqErr    error
		dest, exe string
		want      string
	}{
		{name: "success", status: 200, data: installZip(t, binary)},
		{name: "network", reqErr: errors.New("offline"), want: "offline"},
		{name: "status", status: 403, want: "403"},
		{name: "read", status: 200, body: zipBodyError{}, want: "body failed"},
		{name: "bad destination", status: 200, dest: "missing/download.zip", want: "create"},
		{name: "invalid zip", status: 200, data: []byte("invalid"), want: "extract"},
		{name: "missing binary", status: 200, data: installZip(t, "other"), want: "not exist"},
		{name: "unwritable executable", status: 200, data: installZip(t, binary), exe: "block/binary", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dest := tc.dest
			if dest == "" {
				dest = "download.zip"
			}
			exe := tc.exe
			if exe == "" {
				exe = binary
			}
			if tc.exe != "" {
				if err := os.WriteFile(filepath.Join(dir, "block"), []byte("file"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.reqErr != nil {
					return nil, tc.reqErr
				}
				body := tc.body
				if body == nil {
					body = io.NopCloser(bytes.NewReader(tc.data))
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			}
			err := DownloadAndExtract("https://example.invalid/download", filepath.Join(dir, dest), filepath.Join(dir, exe), "folder")
			if tc.want != "" || tc.exe != "" {
				if err == nil || (tc.want != "" && !strings.Contains(err.Error(), tc.want)) {
					t.Fatalf("error=%v,want=%q", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, exe))
			if err != nil || string(got) != "binary" {
				t.Fatalf("installed=%q,%v", got, err)
			}
		})
	}
}
