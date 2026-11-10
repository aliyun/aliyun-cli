package dgate

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/endpoint"
)

func TestDgateDownloadEndpointSuffix(t *testing.T) {
	t.Setenv(endpoint.EnvSuffix, "aliyuncs.alibaba")
	previousClient, previousDo := httpClient, httpDoFunc
	t.Cleanup(func() { httpClient, httpDoFunc = previousClient, previousDo })
	const binary = "test dgate binary"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(binary)))
	calls := 0
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "assets.oss.aliyuncs.alibaba" {
			t.Fatalf("request host=%q", r.URL.Host)
		}
		var body string
		switch r.URL.Path {
		case "/latest.txt":
			body = "1.2.3"
		case "/dgate":
			body = binary
		case "/dgate.sha256":
			body = sum + "  dgate"
		default:
			t.Fatalf("unexpected path=%s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	httpDoFunc = httpClient.Do
	const base = "https://assets.oss.aliyuncs.com"
	version, err := getRemoteVersion(base + "/latest.txt")
	if err != nil || version != "1.2.3" {
		t.Fatalf("version=%q, err=%v", version, err)
	}
	path := filepath.Join(t.TempDir(), "dgate")
	if err := downloadBinary(base+"/dgate", path); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(base+"/dgate.sha256", path); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != binary || calls != 3 {
		t.Fatalf("content=%q calls=%d err=%v", content, calls, err)
	}
	t.Setenv(endpoint.EnvSuffix, "invalid/suffix")
	if _, err := getRemoteVersion(base + "/latest.txt"); err == nil || calls != 3 {
		t.Fatal("invalid suffix reached transport")
	}
}
