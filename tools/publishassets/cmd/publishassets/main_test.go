package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/assetmanifest"
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

// This deterministic key is a local fake, never a publisher secret.
func fakeKey() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x42}, ed25519.SeedSize))
}

func manifestFixture() assetmanifest.Manifest {
	return assetmanifest.Manifest{
		Format: assetmanifest.Format, Sequence: 7,
		IssuedAt:  time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
		Assets: []assetmanifest.Entry{{
			Product: "bl", Purpose: assetmanifest.Purpose, Version: "1.2.3",
			Platform: "darwin-arm64", Asset: "bl-1.2.3-darwin-arm64.tgz",
			URL:    "https://reviewed.invalid/bl-1.2.3-darwin-arm64.tgz",
			SHA256: strings.Repeat("ab", 32), Size: 123, MinVersion: "1.0.0",
		}},
	}
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T) (input, key, output string, args []string) {
	t.Helper()
	dir := t.TempDir()
	input, key, output = filepath.Join(dir, "reviewed.json"), filepath.Join(dir, "fake.key"), filepath.Join(dir, "output")
	data, err := json.Marshal(manifestFixture())
	if err != nil {
		t.Fatal(err)
	}
	// Input is deliberately not the stable encoded representation.
	writeTestFile(t, input, append([]byte(" \n"), data...))
	writeTestFile(t, key, publishsig.MarshalSecret(fakeKey()))
	return input, key, output, []string{"--input", input, "--output-dir", output, "--secret-key-file", key}
}

func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("unexpected output at %s: %v", path, err)
	}
}

func assertOutputNames(t *testing.T, dir string, want []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.Name())
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("output names = %v, want %v (temporary file leaked)", got, want)
	}
}

func assertSafeError(t *testing.T, err error, stdout, stderr string, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
	secret := strings.TrimSpace(string(publishsig.MarshalSecret(fakeKey())))
	_, encoded, _ := strings.Cut(secret, "\n")
	for _, text := range []string{err.Error(), stdout, stderr} {
		if strings.Contains(text, secret) || strings.Contains(text, encoded) {
			t.Fatal("error or output disclosed the fake secret")
		}
	}
}

func TestRunWritesCanonicalManifestAndSignature(t *testing.T) {
	input, _, output, args := setup(t)
	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(output, assetmanifest.FileName)
	sigPath := manifestPath + ".minisig"
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	want, err := assetmanifest.Encode(manifestFixture())
	if err != nil || !bytes.Equal(data, want) {
		t.Fatalf("noncanonical manifest: %v", err)
	}
	parsed, err := assetmanifest.Parse(data)
	if err != nil || !reflect.DeepEqual(parsed, manifestFixture()) {
		t.Fatalf("manifest changed: %+v, %v", parsed, err)
	}
	sig, err := os.ReadFile(sigPath)
	if err != nil {
		t.Fatal(err)
	}
	keys := []ed25519.PublicKey{fakeKey().Public().(ed25519.PublicKey)}
	if err := publishsig.Verify(keys, assetmanifest.FileName, bytes.NewReader(data), sig); err != nil {
		t.Fatalf("signature verification: %v", err)
	}
	if err := publishsig.Verify(keys, "reviewed.json", bytes.NewReader(data), sig); err == nil {
		t.Fatal("signature did not bind the canonical file name")
	}
	if err := publishsig.Verify(keys, assetmanifest.FileName, bytes.NewReader(append(data, ' ')), sig); err == nil {
		t.Fatal("signature did not bind the canonical bytes")
	}
	original, err := os.ReadFile(input)
	if err != nil || bytes.Equal(original, data) {
		t.Fatal("fixture must require canonical encoding")
	}
	for _, path := range []string{manifestPath, sigPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o", path, info.Mode().Perm())
		}
	}
	assertOutputNames(t, output, []string{assetmanifest.FileName, assetmanifest.FileName + ".minisig"})
	// Explicit offline outputs replace old files, including their permissions.
	for _, path := range []string{manifestPath, sigPath} {
		writeTestFile(t, path, []byte("old output"))
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for path, expected := range map[string][]byte{manifestPath: data, sigPath: sig} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, expected) {
			t.Fatalf("replacement of %s failed: %v", path, err)
		}
		info, err := os.Stat(path)
		if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
			t.Fatalf("replacement permissions: %v, %v", info, err)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("successful run wrote stderr: %s", stderr.String())
	}
}

func TestRunRejectsArgumentsBeforeAnyOutput(t *testing.T) {
	_, _, output, args := setup(t)
	cases := map[string][]string{
		"none":               nil,
		"input missing":      args[2:],
		"output missing":     append(append([]string{}, args[:2]...), args[4:]...),
		"key missing":        args[:4],
		"empty input":        {"--input=", "--output-dir", output, "--secret-key-file", "unused"},
		"empty output":       {"--input", "unused", "--output-dir=", "--secret-key-file", "unused"},
		"empty key":          {"--input", "unused", "--output-dir", output, "--secret-key-file="},
		"unknown flag":       append(append([]string{}, args...), "--unknown"),
		"missing flag value": append(append([]string{}, args...), "--input"),
		"extra arg":          append(append([]string{}, args...), "extra"),
		"positional first":   append([]string{"extra"}, args...),
		"separator extra":    append(append([]string{}, args...), "--", "extra"),
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			assertSafeError(t, err, stdout.String(), stderr.String(), "publishassets:")
			assertAbsent(t, output)
		})
	}
	// Even invalid flag text is not copied to stderr or the returned error.
	secret := strings.TrimSpace(strings.Split(string(publishsig.MarshalSecret(fakeKey())), "\n")[1])
	var stdout, stderr bytes.Buffer
	err := run(append(args, "--"+secret), &stdout, &stderr)
	assertSafeError(t, err, stdout.String(), stderr.String(), "arguments")
}

func TestRunHelpCreatesNothing(t *testing.T) {
	_, _, output, args := setup(t)
	var stdout, stderr bytes.Buffer
	if err := run(append(args, "--help"), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "--input") || !strings.Contains(stderr.String(), "--secret-key-file") {
		t.Fatal("missing usage")
	}
	assertAbsent(t, output)
}

func TestRunRejectsInputAndKeysWithoutWriting(t *testing.T) {
	for _, name := range []string{"missing input", "invalid JSON", "unknown field", "duplicate field", "oversized input", "input directory", "missing key", "invalid key", "empty key", "key directory", "inconsistent key"} {
		t.Run(name, func(t *testing.T) {
			input, key, output, args := setup(t)
			want := "input"
			switch name {
			case "missing input":
				if err := os.Remove(input); err != nil {
					t.Fatal(err)
				}
			case "invalid JSON":
				writeTestFile(t, input, []byte("{not JSON"))
			case "unknown field", "duplicate field":
				data, err := os.ReadFile(input)
				if err != nil {
					t.Fatal(err)
				}
				replacement := `"sequence":7,"unknown":true`
				if name == "duplicate field" {
					replacement = `"sequence":7,"sequence":8`
				}
				writeTestFile(t, input, bytes.Replace(data, []byte(`"sequence":7`), []byte(replacement), 1))
			case "oversized input":
				writeTestFile(t, input, bytes.Repeat([]byte{' '}, 2*1024*1024+1))
				want = "2 MiB"
			case "input directory":
				args[1] = filepath.Dir(input)
			case "missing key":
				if err := os.Remove(key); err != nil {
					t.Fatal(err)
				}
				want = "secret key"
			case "invalid key":
				writeTestFile(t, key, []byte("private-secret-value-that-must-not-appear!"))
				want = "secret key"
			case "empty key":
				writeTestFile(t, key, nil)
				want = "secret key"
			case "key directory":
				args[5] = filepath.Dir(key)
				want = "secret key"
			case "inconsistent key":
				priv := fakeKey()
				priv[ed25519.SeedSize] ^= 1
				writeTestFile(t, key, publishsig.MarshalSecret(priv))
				want = "verify signature"
			}
			var stdout, stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			assertSafeError(t, err, stdout.String(), stderr.String(), want)
			if strings.Contains(err.Error()+stdout.String()+stderr.String(), "private-secret-value-that-must-not-appear!") {
				t.Fatal("invalid key content leaked")
			}
			assertAbsent(t, output)
		})
	}
}

func TestRunInvalidInputDoesNotEchoKeyMaterial(t *testing.T) {
	input, _, output, args := setup(t)
	m := manifestFixture()
	m.Assets[0].Version = strings.TrimSpace(strings.Split(string(publishsig.MarshalSecret(fakeKey())), "\n")[1])
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, input, data)
	var stdout, stderr bytes.Buffer
	err = run(args, &stdout, &stderr)
	assertSafeError(t, err, stdout.String(), stderr.String(), "input manifest")
	assertAbsent(t, output)
}

func TestRunAcceptsExactInputLimit(t *testing.T) {
	input, _, _, args := setup(t)
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, bytes.Repeat([]byte{' '}, 2*1024*1024-len(data))...)
	writeTestFile(t, input, data)
	if err := run(args, io.Discard, io.Discard); err != nil {
		t.Fatalf("exact 2 MiB input rejected: %v", err)
	}
}

func TestRunRejectsOversizedCanonicalEncoding(t *testing.T) {
	input, _, output, args := setup(t)
	m := manifestFixture()
	// This reviewed JSON fits the input limit, but JSON's standard HTML
	// escaping makes the stable encoding exceed the shared limit.
	m.Assets[0].URL += "?review=" + strings.Repeat("&", 400000)
	var data bytes.Buffer
	encoder := json.NewEncoder(&data)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(m); err != nil {
		t.Fatal(err)
	}
	if _, err := assetmanifest.Parse(data.Bytes()); err != nil {
		t.Fatalf("invalid input fixture: %v", err)
	}
	writeTestFile(t, input, data.Bytes())
	err := run(args, io.Discard, io.Discard)
	assertSafeError(t, err, "", "", "encode input manifest")
	assertAbsent(t, output)
}

func TestRunOutputNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("requires Unix directory permissions and an unprivileged user")
	}
	_, _, output, args := setup(t)
	if err := os.Mkdir(output, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(output, 0o700) })
	err := run(args, io.Discard, io.Discard)
	assertSafeError(t, err, "", "", "create temporary file")
	assertOutputNames(t, output, []string{})
}

func TestRunWriteFailure(t *testing.T) {
	const marker = "PUBLISHASSETS_TEST_WRITE_FAILURE"
	if os.Getenv(marker) == "1" {
		// Darwin and Linux use signal 25 for SIGXFSZ. Ignore it only in
		// this child so exceeding ulimit returns a real filesystem error.
		signal.Ignore(syscall.Signal(25))
		args := []string{"--input", os.Getenv(marker + "_INPUT"), "--output-dir", os.Getenv(marker + "_OUTPUT"), "--secret-key-file", os.Getenv(marker + "_KEY")}
		err := run(args, io.Discard, io.Discard)
		assertSafeError(t, err, "", "", "write temporary file")
		assertOutputNames(t, args[3], []string{})
		return
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("requires POSIX file-size limit")
	}
	input, key, output, _ := setup(t)
	m := manifestFixture()
	m.Assets[0].URL += "?review=" + strings.Repeat("x", 8192)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, input, data)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// A separate test process keeps the limit and signal policy isolated;
	// it runs the existing test binary, not a Go command or production key.
	cmd := exec.Command("sh", "-c", `ulimit -f 1 || exit 3; exec "$@"`, "sh", executable, "-test.run=^TestRunWriteFailure$")
	cmd.Env = append(os.Environ(), marker+"=1", marker+"_INPUT="+input, marker+"_KEY="+key, marker+"_OUTPUT="+output)
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("write failure child: %v\n%s", err, result)
	}
	assertOutputNames(t, output, []string{})
}

func TestRunOutputParentFailure(t *testing.T) {
	_, _, output, args := setup(t)
	writeTestFile(t, output, []byte("not a directory"))
	args[3] = filepath.Join(output, "child")
	err := run(args, io.Discard, io.Discard)
	assertSafeError(t, err, "", "", "output directory")
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "not a directory" {
		t.Fatal("parent file changed")
	}
}

func TestRunRenameConflictsCleanTemporaryFiles(t *testing.T) {
	for _, name := range []string{assetmanifest.FileName, assetmanifest.FileName + ".minisig"} {
		t.Run(name, func(t *testing.T) {
			_, _, output, args := setup(t)
			conflict := filepath.Join(output, name)
			if err := os.MkdirAll(conflict, 0o700); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(conflict, "keep"), []byte("existing directory"))
			var stdout, stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			assertSafeError(t, err, stdout.String(), stderr.String(), "rename")
			data, err := os.ReadFile(filepath.Join(conflict, "keep"))
			if err != nil || string(data) != "existing directory" {
				t.Fatal("conflict changed")
			}
			if name == assetmanifest.FileName {
				assertOutputNames(t, output, []string{name})
			} else {
				// Per-file rename is not a two-file transaction. A second rename
				// failure is reported, with the first output already replaced.
				assertOutputNames(t, output, []string{assetmanifest.FileName, name})
			}
		})
	}
}

func TestStageFileCreationFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-directory")
	writeTestFile(t, path, []byte("keep"))
	if _, err := stageFile(path, assetmanifest.FileName, []byte("data")); err == nil || !strings.Contains(err.Error(), "temporary file") {
		t.Fatalf("temporary file creation error = %v", err)
	}
}

func TestMainExit(t *testing.T) {
	oldExit, oldArgs := osExit, os.Args
	t.Cleanup(func() { osExit, os.Args = oldExit, oldArgs })
	code := -1
	osExit = func(n int) { code = n }
	os.Args = []string{"publishassets", "--help"}
	main()
	if code != 0 {
		t.Fatalf("help exit = %d", code)
	}
	_, _, _, args := setup(t)
	os.Args = append([]string{"publishassets"}, args...)
	main()
	if code != 0 {
		t.Fatalf("success exit = %d", code)
	}
	os.Args = []string{"publishassets"}
	main()
	if code != 1 {
		t.Fatalf("error exit = %d", code)
	}
}
