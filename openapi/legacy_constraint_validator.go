package openapi

import (
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

// ConstraintViolationError reports a canonical metadata constraint rejected by
// a PascalCase legacy command.
type ConstraintViolationError struct {
	Flag       string
	Value      string
	Constraint string
	Allowed    []string
	Minimum    string
	Maximum    string
	MinLength  string
	MaxLength  string
	Pattern    string
}

// LegacyDocRequiredError reports AI-mode documentation-required parameters
// missing from a PascalCase invocation.
type LegacyDocRequiredError struct {
	Flags []string
}

func (e *LegacyDocRequiredError) Error() string {
	return "missing docRequired parameter(s): " + strings.Join(e.Flags, ", ")
}

func (e *ConstraintViolationError) Error() string {
	switch e.Constraint {
	case "enum":
		return fmt.Sprintf("--%s value %q is not allowed", e.Flag, e.Value)
	case "minimum":
		return fmt.Sprintf("--%s value %q must be greater than or equal to %s", e.Flag, e.Value, e.Minimum)
	case "maximum":
		return fmt.Sprintf("--%s value %q must be less than or equal to %s", e.Flag, e.Value, e.Maximum)
	case "minLength":
		return fmt.Sprintf("--%s value %q must contain at least %s characters", e.Flag, e.Value, e.MinLength)
	case "maxLength":
		return fmt.Sprintf("--%s value %q must contain at most %s characters", e.Flag, e.Value, e.MaxLength)
	case "pattern":
		return fmt.Sprintf("--%s value %q does not match pattern %q", e.Flag, e.Value, e.Pattern)
	default:
		return fmt.Sprintf("--%s value %q violates its schema constraint", e.Flag, e.Value)
	}
}

func legacyAIModeEnabled(ctx *cli.Context) bool {
	if ctx == nil {
		return false
	}
	cfg, err := aimode.Load(config.GetConfigDir(ctx))
	if err != nil {
		cfg = aimode.DefaultAiConfig()
	}
	forceOn, forceOff := CliAIOverrides(ctx.Flags())
	return aimode.EnabledForCommand(cfg, forceOn, forceOff)
}

func validateLegacyConstraints(ctx *cli.Context, api *canonicalmeta.API) error {
	if api == nil || ctx == nil || ctx.UnknownFlags() == nil || !legacyAIModeEnabled(ctx) {
		return nil
	}
	for _, flag := range ctx.UnknownFlags().Flags() {
		if flag == nil || !flag.IsAssigned() || isRawBodyFlag(flag.Name) {
			continue
		}
		param := api.FindLegacyParameter(flag.Name)
		if param == nil {
			// Preserve the existing unknown-parameter path and its suggestions.
			continue
		}
		value, _ := flag.GetValue()
		if err := validateLegacyValue(flag.Name, value, param); err != nil {
			return err
		}
	}
	return nil
}

func validateLegacyDocRequired(ctx *cli.Context, api *canonicalmeta.API) error {
	if api == nil || ctx == nil || ctx.UnknownFlags() == nil || !legacyAIModeEnabled(ctx) {
		return nil
	}

	assigned := make(map[string]bool)
	nonEmpty := make(map[string]bool)
	for _, flag := range ctx.UnknownFlags().Flags() {
		if flag != nil && flag.IsAssigned() && !isRawBodyFlag(flag.Name) {
			assigned[flag.Name] = true
			value, _ := flag.GetValue()
			if value != "" {
				nonEmpty[flag.Name] = true
			}
		}
	}

	rawBody := (BodyFlag(ctx.Flags()) != nil && BodyFlag(ctx.Flags()).IsAssigned()) ||
		(BodyFileFlag(ctx.Flags()) != nil && BodyFileFlag(ctx.Flags()).IsAssigned())
	missing := make(map[string]bool)
	for _, parameter := range api.LegacyTopLevelParameters() {
		if rawBody && (parameter.LegacyPosition() == "Body" || parameter.LegacyPosition() == "FormData") {
			continue
		}
		validateLegacyDocRequiredNode(
			parameter,
			parameter.LegacyName(),
			assigned,
			nonEmpty,
			missing,
		)
	}

	if len(missing) == 0 {
		return nil
	}
	flags := make([]string, 0, len(missing))
	for flag := range missing {
		flags = append(flags, "--"+flag)
	}
	sort.Strings(flags)
	return &LegacyDocRequiredError{Flags: flags}
}

func validateLegacyDocRequiredNode(
	parameter *canonicalmeta.LegacyParameterView,
	path string,
	assigned map[string]bool,
	nonEmpty map[string]bool,
	missing map[string]bool,
) {
	if parameter == nil {
		return
	}

	children := parameter.LegacyChildren()
	if len(children) > 0 {
		instances := legacyRepeatListInstances(path, assigned)
		if parameter.DocRequired() && len(instances) == 0 {
			missing[path] = true
		}
		for _, instance := range instances {
			for _, child := range children {
				validateLegacyDocRequiredNode(
					child,
					instance+"."+child.LegacyName(),
					assigned,
					nonEmpty,
					missing,
				)
			}
		}
		return
	}

	if parameter.IsLegacyRepeatList() {
		if parameter.DocRequired() && len(legacyRepeatListInstances(path, nonEmpty)) == 0 {
			missing[path] = true
		}
		return
	}

	if parameter.DocRequired() && !nonEmpty[path] {
		missing[path] = true
	}
}

func legacyRepeatListInstances(path string, assigned map[string]bool) []string {
	prefix := path + "."
	seen := make(map[string]bool)
	for name := range assigned {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		index := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			index = rest[:dot]
		}
		if !isDecimalIndex(index) {
			continue
		}
		seen[prefix+index] = true
	}
	instances := make([]string, 0, len(seen))
	for instance := range seen {
		instances = append(instances, instance)
	}
	sort.Strings(instances)
	return instances
}

func isDecimalIndex(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func legacyAPIForInvoker(invoker Invoker) *canonicalmeta.API {
	switch typed := invoker.(type) {
	case *RpcInvoker:
		return typed.api
	case *RestfulInvoker:
		return typed.api
	default:
		return nil
	}
}

func isRawBodyFlag(name string) bool {
	switch strings.ToLower(name) {
	case BodyFlagName, BodyFileFlagName:
		return true
	default:
		return false
	}
}

func validateLegacyValue(flag, value string, param *canonicalmeta.LegacyParameterView) error {
	constraints := param.Constraints()
	if len(constraints.Enum) > 0 {
		matched := false
		for _, allowed := range constraints.Enum {
			if value == allowed {
				matched = true
				break
			}
		}
		if !matched {
			return &ConstraintViolationError{
				Flag:       flag,
				Value:      value,
				Constraint: "enum",
				Allowed:    append([]string(nil), constraints.Enum...),
			}
		}
	}

	constraintType := param.ConstraintType()
	if err := validateLegacyBounds(flag, value, constraintType, constraints); err != nil {
		return err
	}

	if strings.EqualFold(constraintType, "string") {
		length := uint64(utf8.RuneCountInString(value))
		if minimum, err := strconv.ParseUint(constraints.MinLength, 10, 64); constraints.MinLength != "" && err == nil && length < minimum {
			return &ConstraintViolationError{
				Flag: flag, Value: value, Constraint: "minLength", MinLength: constraints.MinLength,
			}
		}
		if maximum, err := strconv.ParseUint(constraints.MaxLength, 10, 64); constraints.MaxLength != "" && err == nil && length > maximum {
			return &ConstraintViolationError{
				Flag: flag, Value: value, Constraint: "maxLength", MaxLength: constraints.MaxLength,
			}
		}
	}

	if constraints.Pattern != "" && strings.EqualFold(constraintType, "string") {
		pattern, err := regexp.Compile(constraints.Pattern)
		if err == nil && !pattern.MatchString(value) {
			return &ConstraintViolationError{
				Flag:       flag,
				Value:      value,
				Constraint: "pattern",
				Pattern:    constraints.Pattern,
			}
		}
	}
	return nil
}

func validateLegacyBounds(flag, value, legacyType string, constraints canonicalmeta.Constraints) error {
	switch strings.ToLower(legacyType) {
	case "int", "integer", "int32", "int64", "long", "float", "double", "number":
		actual, valid := new(big.Rat).SetString(value)
		if !valid {
			return nil
		}
		if minimum, valid := new(big.Rat).SetString(constraints.Minimum); constraints.Minimum != "" && valid && actual.Cmp(minimum) < 0 {
			return &ConstraintViolationError{Flag: flag, Value: value, Constraint: "minimum", Minimum: constraints.Minimum}
		}
		if maximum, valid := new(big.Rat).SetString(constraints.Maximum); constraints.Maximum != "" && valid && actual.Cmp(maximum) > 0 {
			return &ConstraintViolationError{Flag: flag, Value: value, Constraint: "maximum", Maximum: constraints.Maximum}
		}
	}
	return nil
}
