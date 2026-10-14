package main

import (
	"crypto/ed25519"
	"flag"
	"fmt"
	"os"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: publishsign keygen|sign|verify")
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	secretOut := fs.String("secret-out", "", "path for the new secret key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *secretOut == "" {
		return fmt.Errorf("keygen: --secret-out is required")
	}
	if _, err := os.Stat(*secretOut); err == nil {
		return fmt.Errorf("keygen: %s already exists", *secretOut)
	}
	pub, priv, err := publishsig.Generate()
	if err != nil {
		return err
	}
	if err := os.WriteFile(*secretOut, publishsig.MarshalSecret(priv), 0o600); err != nil {
		return err
	}
	fmt.Println(publishsig.FormatPublicKey(pub))
	return nil
}

func sign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	secretFile := fs.String("secret-key-file", "", "publisher secret key file")
	asset := fs.String("asset", "", "asset file to sign")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *secretFile == "" || *asset == "" {
		return fmt.Errorf("sign: --secret-key-file and --asset are required")
	}
	raw, err := os.ReadFile(*secretFile)
	if err != nil {
		return err
	}
	priv, err := publishsig.ParseSecret(raw)
	if err != nil {
		return err
	}
	doc, err := publishsig.SignFile(priv, *asset)
	if err != nil {
		return err
	}
	return os.WriteFile(*asset+".minisig", doc, 0o644)
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	publicKey := fs.String("public-key", "", "base64 public key; default is the embedded release key")
	asset := fs.String("asset", "", "asset file to verify")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *asset == "" {
		return fmt.Errorf("verify: --asset is required")
	}
	keys := publishsig.TrustedPublicKeys()
	if *publicKey != "" {
		key, err := publishsig.ParsePublicKey(*publicKey)
		if err != nil {
			return err
		}
		keys = []ed25519.PublicKey{key}
	}
	sig, err := os.ReadFile(*asset + ".minisig")
	if err != nil {
		return err
	}
	info, err := os.Stat(*asset)
	if err != nil {
		return err
	}
	return publishsig.VerifyFile(keys, info.Name(), *asset, sig)
}
