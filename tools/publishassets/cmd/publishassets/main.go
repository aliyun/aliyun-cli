package main

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun-cli/v3/cli/assetmanifest"
	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

const maxInputBytes = 2 * 1024 * 1024

var osExit = os.Exit

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
		return
	}
	osExit(0)
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("publishassets", flag.ContinueOnError)
	// Do not echo invalid arguments: they may contain secret material.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: publishassets --input <reviewed-manifest.json> --output-dir <directory> --secret-key-file <key-file>")
	}
	input := fs.String("input", "", "reviewed manifest JSON file")
	outputDir := fs.String("output-dir", "", "offline output directory")
	secretFile := fs.String("secret-key-file", "", "explicit publisher secret key file")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("publishassets: invalid arguments")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("publishassets: unexpected positional arguments")
	}
	if *input == "" || *outputDir == "" || *secretFile == "" {
		return fmt.Errorf("publishassets: --input, --output-dir and --secret-key-file are required")
	}

	f, err := os.Open(*input)
	if err != nil {
		return fmt.Errorf("publishassets: open input: %w", err)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxInputBytes+1))
	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("publishassets: read input: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("publishassets: close input: %w", closeErr)
	}
	if len(data) > maxInputBytes {
		return fmt.Errorf("publishassets: input exceeds 2 MiB")
	}
	manifest, err := assetmanifest.Parse(data)
	if err != nil {
		// Schema diagnostics may quote secret material accidentally included in the input.
		return fmt.Errorf("publishassets: invalid input manifest")
	}
	data, err = assetmanifest.Encode(manifest)
	if err != nil {
		return fmt.Errorf("publishassets: encode input manifest: %w", err)
	}
	raw, err := os.ReadFile(*secretFile)
	if err != nil {
		return fmt.Errorf("publishassets: read secret key file: %w", err)
	}
	priv, err := publishsig.ParseSecret(raw)
	if err != nil {
		return fmt.Errorf("publishassets: invalid secret key file")
	}
	sig, err := publishsig.Sign(priv, assetmanifest.FileName, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("publishassets: sign manifest failed")
	}
	if err := publishsig.Verify([]ed25519.PublicKey{priv.Public().(ed25519.PublicKey)}, assetmanifest.FileName, bytes.NewReader(data), sig); err != nil {
		return fmt.Errorf("publishassets: verify signature failed")
	}

	if err := os.MkdirAll(*outputDir, 0o700); err != nil {
		return fmt.Errorf("publishassets: create output directory: %w", err)
	}
	manifestTemp, err := stageFile(*outputDir, assetmanifest.FileName, data)
	if err != nil {
		return err
	}
	defer os.Remove(manifestTemp)
	sigName := publishsig.SignatureName(assetmanifest.FileName)
	sigTemp, err := stageFile(*outputDir, sigName, sig)
	if err != nil {
		return err
	}
	defer os.Remove(sigTemp)
	// These are per-file atomic replacements, not a two-file transaction.
	if err := os.Rename(manifestTemp, filepath.Join(*outputDir, assetmanifest.FileName)); err != nil {
		return fmt.Errorf("publishassets: rename manifest: %w", err)
	}
	if err := os.Rename(sigTemp, filepath.Join(*outputDir, sigName)); err != nil {
		return fmt.Errorf("publishassets: rename signature: %w", err)
	}
	fmt.Fprintln(stdout, "wrote "+assetmanifest.FileName+" and "+sigName)
	return nil
}

func stageFile(dir, name string, data []byte) (path string, err error) {
	f, err := os.CreateTemp(dir, "."+name+"-*") // CreateTemp uses mode 0600.
	if err != nil {
		return "", fmt.Errorf("publishassets: create temporary file: %w", err)
	}
	path = f.Name()
	defer func() {
		if err != nil {
			os.Remove(path)
		}
	}()
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return path, fmt.Errorf("publishassets: write temporary file: %w", writeErr)
	}
	if closeErr != nil {
		return path, fmt.Errorf("publishassets: close temporary file: %w", closeErr)
	}
	return path, nil
}
