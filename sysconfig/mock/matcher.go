package mock

import (
	"regexp"
	"strings"
)

type parameterGroup struct {
	tokens []string
	star   bool
}

func FindMatch(records []Record, args []string) (int, bool) {
	for i, record := range records {
		if MatchCommand(record.Cmd, args) {
			return i, true
		}
	}
	return 0, false
}

func Consume(records []Record, index int) []Record {
	if index < 0 || index >= len(records) {
		return records
	}
	if records[index].Times == 0 {
		return records
	}

	times := records[index].Times - 1
	if times <= 0 {
		return append(records[:index], records[index+1:]...)
	}
	records[index].Times = times
	return records
}

func MatchCommand(rule string, args []string) bool {
	args = StripLeadingGlobalFlags(args)
	ruleTokens := strings.Fields(rule)
	if len(ruleTokens) == 0 {
		return len(args) == 0
	}

	ruleIdentityEnd := commandIdentityEnd(ruleTokens)
	ruleIdentity := ruleTokens[:ruleIdentityEnd]

	if len(args) < len(ruleIdentity) {
		return false
	}
	for i, token := range ruleIdentity {
		if !matchToken(token, args[i]) {
			return false
		}
	}

	ruleGroups := splitParameterGroups(ruleTokens[ruleIdentityEnd:], true)
	hasStarGroup := containsStarGroup(ruleGroups)
	if len(ruleGroups) == 0 {
		return len(args) == len(ruleIdentity)
	}
	if hasStarGroup && len(nonStarGroups(ruleGroups)) == 0 {
		return true
	}

	argParamsStart := len(ruleIdentity)
	if !hasStarGroup {
		argIdentityEnd := commandIdentityEnd(args)
		if argIdentityEnd != len(ruleIdentity) {
			return false
		}
		argParamsStart = argIdentityEnd
	}
	if len(args) < argParamsStart {
		return false
	}

	return matchParameterGroups(nonStarGroups(ruleGroups), args[argParamsStart:], true)
}

func FirstCommandToken(args []string) string {
	args = StripLeadingGlobalFlags(args)
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func StripLeadingGlobalFlags(args []string) []string {
	for i := 0; i < len(args); {
		token := args[i]
		if !strings.HasPrefix(token, "-") || strings.HasPrefix(token, "---") {
			return args[i:]
		}
		if token == "--" {
			if i+1 >= len(args) {
				return nil
			}
			return args[i+1:]
		}
		if strings.HasPrefix(token, "--") {
			name, hasInlineValue := flagNameAndInlineValue(token[2:])
			i++
			if !hasInlineValue && !isNoValueLongFlag(name) && i < len(args) {
				i++
			}
			continue
		}
		if len(token) == 2 {
			i++
			if !isNoValueShortFlag(token[1]) && i < len(args) {
				i++
			}
			continue
		}
		return args[i:]
	}
	return nil
}

func flagNameAndInlineValue(token string) (string, bool) {
	index := strings.IndexAny(token, "=:")
	if index < 0 {
		return token, false
	}
	return token[:index], true
}

func isNoValueLongFlag(name string) bool {
	switch name {
	case "help", "yes", "secure", "insecure", "force", "dryrun", "quiet",
		"skip-secure-verify", "cli-auto-prompt", "cli-no-auto-prompt":
		return true
	case "cli-ai-mode", "no-cli-ai-mode":
		return true
	default:
		return false
	}
}

func isNoValueShortFlag(ch byte) bool {
	switch ch {
	case 'h', 'y', 'q':
		return true
	default:
		return false
	}
}

func commandIdentityEnd(tokens []string) int {
	for i, token := range tokens {
		if token == "*" || isFlagToken(token) {
			return i
		}
	}
	return len(tokens)
}

func splitParameterGroups(tokens []string, rule bool) []parameterGroup {
	groups := make([]parameterGroup, 0, len(tokens))
	for _, token := range tokens {
		if rule && token == "*" {
			groups = append(groups, parameterGroup{star: true})
			continue
		}
		if isFlagToken(token) || len(groups) == 0 || groups[len(groups)-1].star {
			groups = append(groups, parameterGroup{tokens: []string{token}})
			continue
		}
		groups[len(groups)-1].tokens = append(groups[len(groups)-1].tokens, token)
	}
	return groups
}

func isFlagToken(token string) bool {
	return strings.HasPrefix(token, "--")
}

func containsStarGroup(groups []parameterGroup) bool {
	for _, group := range groups {
		if group.star {
			return true
		}
	}
	return false
}

func nonStarGroups(groups []parameterGroup) []parameterGroup {
	out := make([]parameterGroup, 0, len(groups))
	for _, group := range groups {
		if !group.star {
			out = append(out, group)
		}
	}
	return out
}

func matchParameterGroups(ruleGroups []parameterGroup, argTokens []string, allowExtra bool) bool {
	if len(ruleGroups) == 0 {
		return allowExtra || len(argTokens) == 0
	}
	if minRuleTokenCount(ruleGroups) > len(argTokens) {
		return false
	}

	used := make([]bool, len(argTokens))
	return matchParameterGroupsFrom(ruleGroups, argTokens, used, 0, allowExtra)
}

func minRuleTokenCount(ruleGroups []parameterGroup) int {
	total := 0
	for _, group := range ruleGroups {
		total += len(group.tokens)
	}
	return total
}

func matchParameterGroupsFrom(ruleGroups []parameterGroup, argTokens []string, used []bool, ruleIndex int, allowExtra bool) bool {
	if ruleIndex == len(ruleGroups) {
		if allowExtra {
			return true
		}
		for _, ok := range used {
			if !ok {
				return false
			}
		}
		return true
	}
	ruleGroup := ruleGroups[ruleIndex]
	for i := 0; i+len(ruleGroup.tokens) <= len(argTokens); i++ {
		if !matchParameterGroupAt(ruleGroup, argTokens, used, i) {
			continue
		}
		markUsed(used, i, len(ruleGroup.tokens), true)
		if matchParameterGroupsFrom(ruleGroups, argTokens, used, ruleIndex+1, allowExtra) {
			return true
		}
		markUsed(used, i, len(ruleGroup.tokens), false)
	}
	return false
}

func matchParameterGroupAt(ruleGroup parameterGroup, argTokens []string, used []bool, start int) bool {
	for i, ruleToken := range ruleGroup.tokens {
		argIndex := start + i
		if used[argIndex] || !matchToken(ruleToken, argTokens[argIndex]) {
			return false
		}
	}
	return true
}

func markUsed(used []bool, start, count int, value bool) {
	for i := 0; i < count; i++ {
		used[start+i] = value
	}
}

func matchToken(rule, arg string) bool {
	if rule == arg {
		return true
	}
	if !strings.ContainsAny(rule, "?*") {
		return false
	}

	pattern := regexp.QuoteMeta(rule)
	pattern = strings.ReplaceAll(pattern, `\?`, `[^\s]+`)
	pattern = strings.ReplaceAll(pattern, `\*`, `.*`)
	return regexp.MustCompile("^" + pattern + "$").MatchString(arg)
}
