package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aliyun/aliyun-cli/v3/tools/buildtrace"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	if len(args) > 1 && args[1] == "retarget" {
		if err := retarget(args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	if err := write(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func write(args []string) error {
	fs := flag.NewFlagSet("buildtrace", flag.ContinueOnError)
	binary := fs.String("binary", "", "built binary")
	asset := fs.String("asset", "", "archive to describe")
	version := fs.String("version", "", "release version without v")
	tag := fs.String("tag", "", "git tag")
	commit := fs.String("commit", "", "full commit SHA")
	gitlink := fs.String("gitlink", "", "aliyun-openapi-meta gitlink")
	runURL := fs.String("run-url", "", "GitHub Actions run URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return buildtrace.Write(buildtrace.Options{
		Binary:  *binary,
		Asset:   *asset,
		Version: *version,
		Tag:     *tag,
		Commit:  *commit,
		Gitlink: *gitlink,
		RunURL:  *runURL,
	})
}

func retarget(args []string) error {
	fs := flag.NewFlagSet("retarget", flag.ContinueOnError)
	from := fs.String("from", "", "existing provenance JSON")
	asset := fs.String("asset", "", "renamed archive")
	sbom := fs.String("sbom", "", "SBOM for the renamed archive")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *asset == "" || *sbom == "" {
		return fmt.Errorf("retarget: --from, --asset, and --sbom are required")
	}
	return buildtrace.Retarget(*from, *asset, *sbom)
}
