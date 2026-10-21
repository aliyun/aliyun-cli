package releasever

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli/publishsig"
)

// Execute runs the releasever CLI. Commands:
//
//	classify <version>                 prints stable or prerelease
//	plan <version>                     prints the publish policy as key=value lines
//	append-version <version> <file> <public-keys>
//	supported-public-keys              prints the key generations embedded in this source, comma-separated
func Execute(args []string, stdout, stderr io.Writer) int {
	filtered, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch filtered[0] {
	case "classify":
		if len(filtered) != 2 || filtered[1] == "" {
			fmt.Fprintln(stderr, usageText())
			return 1
		}
		kind, err := Classify(filtered[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, kind)
		return 0
	case "plan":
		if len(filtered) != 2 || filtered[1] == "" {
			fmt.Fprintln(stderr, usageText())
			return 1
		}
		plan, err := ReleasePlan(filtered[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := writePlan(stdout, plan); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "supported-public-keys":
		if len(filtered) != 1 {
			fmt.Fprintln(stderr, usageText())
			return 1
		}
		fmt.Fprintln(stdout, strings.Join(publishsig.SupportedPublicKeyVersions, ","))
		return 0
	case "append-version":
		if len(filtered) != 4 || filtered[1] == "" || filtered[2] == "" || filtered[3] == "" {
			fmt.Fprintln(stderr, usageText())
			return 1
		}
		existing, err := os.ReadFile(filtered[2])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		updated, err := AppendStableVersion(existing, filtered[1], filtered[3])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if _, err := stdout.Write(updated); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", filtered[0])
		return 1
	}
}

func usageText() string {
	return "usage: releasever classify|plan <version> | append-version <version> <file> <public-keys> | supported-public-keys"
}

func parseArgs(args []string) ([]string, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--" {
			continue
		}
		filtered = append(filtered, arg)
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("%s", usageText())
	}
	return filtered, nil
}

func writePlan(w io.Writer, plan Plan) error {
	var b strings.Builder
	fmt.Fprintf(&b, "kind=%s\n", plan.Kind)
	fmt.Fprintf(&b, "update_latest=%s\n", boolFlag(plan.UpdateLatest))
	fmt.Fprintf(&b, "write_stable_version=%s\n", boolFlag(plan.WriteStableVersion))
	fmt.Fprintf(&b, "mark_official=%s\n", boolFlag(plan.MarkOfficial))
	_, err := io.WriteString(w, b.String())
	return err
}

func boolFlag(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
