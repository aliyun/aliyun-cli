package releasever

import (
	"fmt"
	"io"
	"strings"
)

// Execute runs the releasever CLI. Commands:
//
//	classify <version>  prints stable or prerelease
//	plan <version>      prints the publish policy as key=value lines
func Execute(args []string, stdout, stderr io.Writer) int {
	cmd, version, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch cmd {
	case "classify":
		kind, err := Classify(version)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, kind)
		return 0
	case "plan":
		plan, err := ReleasePlan(version)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := writePlan(stdout, plan); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", cmd)
		return 1
	}
}

func parseArgs(args []string) (string, string, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--" {
			continue
		}
		filtered = append(filtered, arg)
	}
	if len(filtered) != 2 || filtered[1] == "" {
		return "", "", fmt.Errorf("usage: releasever classify|plan <version>")
	}
	return filtered[0], filtered[1], nil
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
