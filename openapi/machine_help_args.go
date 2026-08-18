package openapi

import "strings"

// NormalizeMachineHelpArgs rewrites the two public machine-help syntaxes to
// the existing help path plus an internal format flag. Ordinary text help and
// API parameters named --format are left untouched.
func NormalizeMachineHelpArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	if args[0] == "help" {
		if normalized, ok := normalizeHelpCommand(args[1:]); ok {
			return normalized
		}
	}

	out := make([]string, 0, len(args)+2)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--help=") {
			out = append(out, "--help", "--"+MachineHelpFormatFlagName, strings.TrimPrefix(arg, "--help="))
			continue
		}
		out = append(out, arg)
	}
	return out
}

func normalizeHelpCommand(args []string) ([]string, bool) {
	target := make([]string, 0, len(args)+2)
	format := ""
	found := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
			found = true
		case arg == "--format" && i+1 < len(args):
			format = args[i+1]
			found = true
			i++
		default:
			target = append(target, arg)
		}
	}
	if !found {
		return nil, false
	}
	return append(target, "--help", "--"+MachineHelpFormatFlagName, format), true
}
