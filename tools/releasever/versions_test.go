package releasever

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

func TestAppendStableVersion(t *testing.T) {
	got, err := AppendStableVersion(nil, "v3.5.2", "1")
	if err != nil {
		t.Fatal(err)
	}
	doc := decodeCatalog(t, got)
	if doc.Format != publishsig.VersionsFormat || len(doc.Versions) != 1 {
		t.Fatalf("first record = %s", got)
	}
	if doc.Versions[0].Version != "3.5.2" || strings.Join(doc.Versions[0].PublicKeys, ",") != "1" {
		t.Fatalf("entry = %+v", doc.Versions[0])
	}
	if doc.Versions[0].Description != "publisher key generation 1" {
		t.Fatalf("description = %q", doc.Versions[0].Description)
	}
	if strings.Contains(doc.Versions[0].Description, "Ed25519") || strings.Contains(strings.ToLower(doc.Versions[0].Description), "ed25519") {
		t.Fatalf("description names an algorithm: %q", doc.Versions[0].Description)
	}

	got, err = AppendStableVersion(got, "3.5.3", "1, 2, 1")
	if err != nil {
		t.Fatal(err)
	}
	doc = decodeCatalog(t, got)
	if len(doc.Versions) != 2 || doc.Versions[0].Version != "3.5.2" || doc.Versions[1].Version != "3.5.3" {
		t.Fatalf("append = %+v", doc.Versions)
	}
	if strings.Join(doc.Versions[1].PublicKeys, ",") != "1,2" {
		t.Fatalf("public keys = %#v", doc.Versions[1].PublicKeys)
	}
	if doc.Versions[1].Description != "publisher key generations 1, 2" {
		t.Fatalf("description = %q", doc.Versions[1].Description)
	}

	again, err := AppendStableVersion(got, "v3.5.2", "1")
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(got) {
		t.Fatalf("duplicate changed history:\n%s\n%s", again, got)
	}

	updated, err := AppendStableVersion(got, "3.5.2", "1,2")
	if err != nil {
		t.Fatal(err)
	}
	doc = decodeCatalog(t, updated)
	if len(doc.Versions) != 2 || strings.Join(doc.Versions[0].PublicKeys, ",") != "1,2" || doc.Versions[1].Version != "3.5.3" {
		t.Fatalf("in-place update = %+v", doc.Versions)
	}
	if doc.Versions[0].Description != "publisher key generations 1, 2" {
		t.Fatalf("updated description = %q", doc.Versions[0].Description)
	}

	if _, err := AppendStableVersion(nil, "3.5.2-beta.1", "1"); err == nil || !strings.Contains(err.Error(), "prerelease") {
		t.Fatalf("prerelease err = %v", err)
	}
	if _, err := AppendStableVersion(nil, "latest", "1"); err == nil {
		t.Fatal("invalid version was recorded")
	}
	if _, err := AppendStableVersion(nil, "3.5.2", "no spaces"); err == nil {
		t.Fatal("invalid public key version was recorded")
	}
	if _, err := AppendStableVersion([]byte(`{"format":"aliyun-cli-versions v9"}`), "3.5.2", "1"); err == nil {
		t.Fatal("foreign catalog was rewritten")
	}
}

func TestVersionCatalogEdges(t *testing.T) {
	got, err := AppendStableVersion(nil, "3.5.4", "1,,2")
	if err != nil {
		t.Fatal(err)
	}
	doc := decodeCatalog(t, got)
	if strings.Join(doc.Versions[0].PublicKeys, ",") != "1,2" {
		t.Fatalf("public keys = %#v", doc.Versions[0].PublicKeys)
	}
	got, err = AppendStableVersion(nil, "3.5.4", "A.b_c-1")
	if err != nil {
		t.Fatal(err)
	}
	doc = decodeCatalog(t, got)
	if doc.Versions[0].PublicKeys[0] != "A.b_c-1" {
		t.Fatalf("public keys = %#v", doc.Versions[0].PublicKeys)
	}
	if _, err := AppendStableVersion(nil, "3.5.4", strings.Repeat("a", 65)); err == nil {
		t.Fatal("oversized public key version was accepted")
	}
	if validSignatureVersion("") || validSignatureVersion(strings.Repeat("b", 65)) {
		t.Fatal("empty or oversized version was valid")
	}
	if !validSignatureVersion(strings.Repeat("c", 64)) {
		t.Fatal("64-character version was rejected")
	}
	for _, keys := range []string{"", ",", " , "} {
		if _, err := AppendStableVersion(nil, "3.5.4", keys); err == nil || !strings.Contains(err.Error(), "required") {
			t.Fatalf("keys %q err = %v", keys, err)
		}
	}
	if _, err := AppendStableVersion([]byte("{"), "3.5.4", "1"); err == nil || !strings.Contains(err.Error(), "versions file") {
		t.Fatalf("invalid json err = %v", err)
	}
	got, err = AppendStableVersion([]byte(`{"format":"`+publishsig.VersionsFormat+`"}`), "3.5.4", "1")
	if err != nil {
		t.Fatal(err)
	}
	doc = decodeCatalog(t, got)
	if len(doc.Versions) != 1 || doc.Versions[0].Version != "3.5.4" {
		t.Fatalf("missing versions key = %+v", doc.Versions)
	}

	orig := marshalIndent
	marshalIndent = func(any, string, string) ([]byte, error) { return nil, os.ErrInvalid }
	defer func() { marshalIndent = orig }()
	if _, err := AppendStableVersion(nil, "3.5.4", "1"); err == nil {
		t.Fatal("marshal error was ignored")
	}
}

func TestAppendVersionCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions")
	first, err := AppendStableVersion(nil, "3.5.1", "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, first, 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runCLI(t, "append-version", "--", "3.5.2", path, "1,2")
	if code != 0 || stderr != "" {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	doc := decodeCatalog(t, []byte(stdout))
	if len(doc.Versions) != 2 || doc.Versions[1].Version != "3.5.2" || strings.Join(doc.Versions[1].PublicKeys, ",") != "1,2" {
		t.Fatalf("stdout = %s", stdout)
	}
	if _, stderr, code := runCLI(t, "append-version", "3.5.2"); code == 0 || !strings.Contains(stderr, "usage:") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	out, stderr, code := runCLI(t, "supported-public-keys")
	if code != 0 || stderr != "" || out != strings.Join(publishsig.SupportedPublicKeyVersions, ",")+"\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out, stderr)
	}
}

func decodeCatalog(t *testing.T, body []byte) versionCatalog {
	t.Helper()
	var doc versionCatalog
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}
