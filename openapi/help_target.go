package openapi

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

type HelpLevel string

const (
	HelpLevelRoot      HelpLevel = "root"
	HelpLevelProduct   HelpLevel = "product"
	HelpLevelAction    HelpLevel = "action"
	HelpLevelParameter HelpLevel = "parameter"
	HelpLevelUtility   HelpLevel = "utility"
)

type CommandStyle string

const (
	CommandStyleCamel CommandStyle = "camel"
	CommandStyleKebab CommandStyle = "kebab"
)

type HelpProvider string

const (
	HelpProviderHost   HelpProvider = "host"
	HelpProviderPlugin HelpProvider = "plugin"
)

type APIVersionFlag string

const (
	VersionFlagLegacy APIVersionFlag = "version"
	VersionFlagAPI    APIVersionFlag = "api-version"
)

type HelpOperation = cli.HelpOperation
type HelpOutput = cli.HelpOutput
type HelpSection = cli.HelpSection

const (
	HelpOperationDefault = cli.HelpOperationDefault
	HelpOperationAll     = cli.HelpOperationAll
	HelpOperationSearch  = cli.HelpOperationSearch

	HelpOutputText = cli.HelpOutputText
	HelpOutputJSON = cli.HelpOutputJSON

	HelpSectionRequest  = cli.HelpSectionRequest
	HelpSectionResponse = cli.HelpSectionResponse
)

// HelpTarget is the shared L0-L3/utility routing contract. Action and
// Parameter are already-resolved names for CommandStyle; the builder never
// guesses Canonical cmd_name mappings. VersionFlag records an explicit user
// spelling when present and otherwise follows CommandStyle.
type HelpTarget struct {
	Level           HelpLevel
	Product         string
	Action          string
	Parameter       string
	Utility         string
	CommandStyle    CommandStyle
	VersionFlag     APIVersionFlag
	Version         string
	Section         HelpSection
	SectionExplicit bool
	Operation       HelpOperation
	SearchQuery     string
	Output          HelpOutput
	Provider        HelpProvider
}

func (t HelpTarget) normalized() HelpTarget {
	if t.Operation == "" {
		t.Operation = HelpOperationDefault
	}
	if t.Output == "" {
		t.Output = HelpOutputText
	}
	if t.Section == "" {
		t.Section = HelpSectionRequest
	}
	if t.Provider == "" {
		t.Provider = HelpProviderHost
	}
	if t.CommandStyle == "" && t.Level != HelpLevelRoot && t.Level != HelpLevelUtility {
		if t.Action != "" && strings.ToLower(t.Action) == t.Action {
			t.CommandStyle = CommandStyleKebab
		} else {
			t.CommandStyle = CommandStyleCamel
		}
	}
	return t
}

func (t HelpTarget) Validate() error {
	t = t.normalized()
	if err := validateTargetPath(t); err != nil {
		return err
	}
	switch t.Operation {
	case HelpOperationDefault, HelpOperationAll:
		if t.SearchQuery != "" {
			return fmt.Errorf("Help search query is only valid with operation %q", HelpOperationSearch)
		}
	case HelpOperationSearch:
		if strings.TrimSpace(t.SearchQuery) == "" {
			return fmt.Errorf("Help search requires a non-empty query")
		}
	default:
		return fmt.Errorf("unsupported Help operation %q", t.Operation)
	}
	if t.Output != HelpOutputText && t.Output != HelpOutputJSON {
		return fmt.Errorf("unsupported Help output %q", t.Output)
	}
	if t.Provider != HelpProviderHost && t.Provider != HelpProviderPlugin {
		return fmt.Errorf("unsupported Help provider %q", t.Provider)
	}
	switch t.Level {
	case HelpLevelProduct, HelpLevelAction, HelpLevelParameter:
		if t.CommandStyle != CommandStyleCamel && t.CommandStyle != CommandStyleKebab {
			return fmt.Errorf("unsupported command style %q", t.CommandStyle)
		}
	case HelpLevelRoot, HelpLevelUtility:
		if t.CommandStyle != "" {
			return fmt.Errorf("command style is not valid for %s Help", t.Level)
		}
	}
	if t.Section != HelpSectionRequest && t.Section != HelpSectionResponse {
		return fmt.Errorf("unsupported Help section %q", t.Section)
	}
	if t.Section == HelpSectionResponse && !t.SectionExplicit {
		return fmt.Errorf("response Help requires an explicit section target")
	}
	if t.SectionExplicit {
		if t.Level != HelpLevelAction {
			return fmt.Errorf("Help sections require an action target")
		}
		if t.Operation == HelpOperationAll {
			return fmt.Errorf("Help sections do not support --help-all")
		}
	}
	if t.Version != "" {
		if t.Level == HelpLevelRoot || t.Level == HelpLevelUtility {
			return fmt.Errorf("API version is not valid for %s Help", t.Level)
		}
		if _, err := t.effectiveVersionFlag(); err != nil {
			return err
		}
	} else if t.VersionFlag != "" {
		return fmt.Errorf("version flag %q requires a version", t.VersionFlag)
	}
	return nil
}

func validateTargetPath(t HelpTarget) error {
	switch t.Level {
	case HelpLevelRoot:
		return nil
	case HelpLevelProduct:
		return validateCommandToken("product", t.Product)
	case HelpLevelAction:
		if err := validateCommandToken("product", t.Product); err != nil {
			return err
		}
		return validateCommandToken("action", t.Action)
	case HelpLevelParameter:
		if err := validateCommandToken("product", t.Product); err != nil {
			return err
		}
		if err := validateCommandToken("action", t.Action); err != nil {
			return err
		}
		parameter := strings.TrimPrefix(t.Parameter, "--")
		if strings.ContainsAny(parameter, "=:") {
			return fmt.Errorf("parameter Help requires an unassigned flag")
		}
		return validateCommandToken("parameter", parameter)
	case HelpLevelUtility:
		return validateCommandToken("utility", t.Utility)
	default:
		return fmt.Errorf("unsupported Help level %q", t.Level)
	}
}

func validateCommandToken(label, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("Help target requires %s", label)
	}
	if value != trimmed || strings.ContainsFunc(value, unicode.IsSpace) {
		return fmt.Errorf("Help target %s must be one command token", label)
	}
	if strings.HasPrefix(value, "-") {
		return fmt.Errorf("Help target %s cannot be an option", label)
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("-._", r)
	}) >= 0 {
		return fmt.Errorf("Help target %s contains unsupported command syntax", label)
	}
	return nil
}

func (t HelpTarget) effectiveVersionFlag() (APIVersionFlag, error) {
	flag := APIVersionFlag(strings.TrimPrefix(string(t.VersionFlag), "--"))
	if flag == "" {
		if t.CommandStyle == CommandStyleKebab {
			flag = VersionFlagAPI
		} else {
			flag = VersionFlagLegacy
		}
	}
	if flag != VersionFlagLegacy && flag != VersionFlagAPI {
		return "", fmt.Errorf("unsupported API version flag %q", t.VersionFlag)
	}
	return flag, nil
}

// BuildHelpCommand emits the one public navigation grammar used by Help next
// links, hints, and recovery. Ordinary L0-L3 Help is suffix form; explicit
// request/response sections are the sole prefix `aliyun help ...` form.
func BuildHelpCommand(target HelpTarget) (string, error) {
	target = target.normalized()
	if err := target.Validate(); err != nil {
		return "", err
	}

	args := []string{"aliyun"}
	if target.SectionExplicit {
		args = append(args, "help", target.Product, target.Action)
	} else {
		switch target.Level {
		case HelpLevelRoot:
		case HelpLevelProduct:
			args = append(args, target.Product)
		case HelpLevelAction:
			args = append(args, target.Product, target.Action)
		case HelpLevelParameter:
			args = append(args, target.Product, target.Action)
		case HelpLevelUtility:
			args = append(args, "utils", target.Utility)
		}
	}

	if target.Version != "" {
		flag, _ := target.effectiveVersionFlag()
		args = append(args, "--"+string(flag), shellQuoteHelpArg(target.Version))
	}
	if target.Level == HelpLevelParameter {
		args = append(args, "--"+strings.TrimPrefix(target.Parameter, "--"))
	}
	if target.SectionExplicit {
		args = append(args, "--"+CliHelpSectionFlagName, string(target.Section))
	}

	switch target.Operation {
	case HelpOperationDefault:
		if !target.SectionExplicit {
			args = append(args, "--help")
		}
	case HelpOperationAll:
		args = append(args, "--"+CliHelpAllFlagName)
	case HelpOperationSearch:
		args = append(args, "--"+CliHelpSearchFlagName, shellQuoteHelpArg(strings.TrimSpace(target.SearchQuery)))
	}
	if target.Output == HelpOutputJSON {
		args = append(args, "--"+CliOutputFlagName, string(HelpOutputJSON))
	}
	return strings.Join(args, " "), nil
}

func shellQuoteHelpArg(value string) string {
	if value == "<keyword>" {
		return value
	}
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("-._/:@", r)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
