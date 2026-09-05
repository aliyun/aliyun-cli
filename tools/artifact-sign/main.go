// Command artifact-sign creates an Ed25519 detached signature for a manifest file.
//
// Usage:
//
//	artifact-sign -keyid plugins-v1 -seed-b64 <base64-32-byte-seed> -in plugin_pkg_index.json -out plugin_pkg_index.json.sig
//
// Prefer KMS in production; this tool is for local/dev and CI dry-runs.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli/trust"
)

func main() {
	keyID := flag.String("keyid", "", "signing key id")
	seedB64 := flag.String("seed-b64", "", "base64-encoded Ed25519 private seed (32 bytes)")
	inPath := flag.String("in", "", "path to payload file to sign")
	outPath := flag.String("out", "", "path to write .sig JSON (default: <in>.sig)")
	flag.Parse()

	if *keyID == "" || *seedB64 == "" || *inPath == "" {
		fmt.Fprintln(os.Stderr, "usage: artifact-sign -keyid ID -seed-b64 SEED -in FILE [-out FILE.sig]")
		os.Exit(2)
	}
	if *outPath == "" {
		*outPath = *inPath + ".sig"
	}

	priv, err := trust.DecodePrivateSeed(*seedB64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	payload, err := os.ReadFile(*inPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	sig, err := trust.Sign(*keyID, priv, payload, time.Now())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, err := trust.MarshalSignature(sig)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, append(raw, '\n'), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (keyid=%s)\n", *outPath, *keyID)
}
