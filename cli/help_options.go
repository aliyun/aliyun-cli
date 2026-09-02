package cli

import (
	"fmt"
	"strings"
)

const (
	HelpAllFlagName    = "help-all"
	HelpSearchFlagName = "help-search"
	CLIOutputFlagName  = "cli-output"
	CLISectionFlagName = "cli-section"
)

type HelpOperation string

const (
	HelpOperationDefault HelpOperation = "default"
	HelpOperationAll     HelpOperation = "all"
	HelpOperationSearch  HelpOperation = "search"
)

type HelpOutput string

const (
	HelpOutputText HelpOutput = "text"
	HelpOutputJSON HelpOutput = "json"
)

type HelpSection string

const (
	HelpSectionRequest  HelpSection = "request"
	HelpSectionResponse HelpSection = "response"
)

// HelpOptions is the transport-neutral result of recognizing the public Help
// modifiers. Requested deliberately remains false for a lone --cli-output:
// output selection is orthogonal to, and cannot itself enter, Help.
type HelpOptions struct {
	Requested   bool
	Operation   HelpOperation
	SearchQuery string
	// SearchAll reports that a search is uncapped. Every public
	// --help-search is uncapped; an explicit --help-all remains accepted.
	SearchAll       bool
	Output          HelpOutput
	Section         HelpSection
	SectionExplicit bool
}

type HelpOptionErrorCode string

const (
	HelpOptionDuplicate      HelpOptionErrorCode = "DUPLICATE_HELP_OPTION"
	HelpOptionConflict       HelpOptionErrorCode = "CONFLICTING_HELP_OPTIONS"
	HelpOptionEmptySearch    HelpOptionErrorCode = "EMPTY_HELP_SEARCH"
	HelpOptionInvalidOutput  HelpOptionErrorCode = "INVALID_CLI_OUTPUT"
	HelpOptionInvalidSection HelpOptionErrorCode = "INVALID_HELP_SECTION"
)

// HelpOptionError keeps parser failures typed so the AI/local-error layer can
// normalize them without matching human-readable strings.
type HelpOptionError struct {
	Code          HelpOptionErrorCode
	Option        string
	ConflictsWith string
	Value         string
}

func (e *HelpOptionError) Error() string {
	switch e.Code {
	case HelpOptionDuplicate:
		return fmt.Sprintf("%s duplicated", e.Option)
	case HelpOptionConflict:
		return fmt.Sprintf("%s conflicts with %s", e.Option, e.ConflictsWith)
	case HelpOptionEmptySearch:
		return "--" + HelpSearchFlagName + " requires a non-empty query"
	case HelpOptionInvalidOutput:
		return fmt.Sprintf("--%s only supports json, got %q", CLIOutputFlagName, e.Value)
	case HelpOptionInvalidSection:
		return fmt.Sprintf("--%s must be request or response, got %q", CLISectionFlagName, e.Value)
	default:
		return "invalid Help options"
	}
}

// Help option failures are local CLI usage errors. Implementing the existing
// suggestion contract gives them the standard usage-error exit code without
// adding decorative suggestions.
func (*HelpOptionError) GetSuggestions() []string { return nil }

func (*HelpOptionError) AIRecoveryEligible() {}

// ParseHelpOptions recognizes only the stable public Help surface. Other argv
// tokens are intentionally preserved for the normal command parser/provider
// router. The caller still owns target parsing and plugin delegation.
func ParseHelpOptions(args []string) (HelpOptions, error) {
	opts := HelpOptions{
		Operation: HelpOperationDefault,
		Output:    HelpOutputText,
		Section:   HelpSectionRequest,
	}
	if len(args) > 0 && args[0] == "help" {
		opts.Requested = true
	}

	operationOption := ""
	outputSeen := false
	sectionSeen := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			if err := setHelpOperation(&opts, &operationOption, HelpOperationDefault, arg); err != nil {
				return HelpOptions{}, err
			}
		case arg == "--"+HelpAllFlagName:
			if opts.Operation == HelpOperationSearch {
				if opts.SearchAll {
					return HelpOptions{}, &HelpOptionError{Code: HelpOptionDuplicate, Option: arg}
				}
				// Search + All composes: the keyword keeps filtering while the
				// result cap is removed.
				opts.SearchAll = true
				break
			}
			if err := setHelpOperation(&opts, &operationOption, HelpOperationAll, arg); err != nil {
				return HelpOptions{}, err
			}
		case arg == "--"+HelpSearchFlagName || strings.HasPrefix(arg, "--"+HelpSearchFlagName+"="):
			if opts.Operation == HelpOperationAll && !opts.SearchAll {
				// All + Search composes in either flag order.
				opts.Operation = HelpOperationSearch
				opts.SearchAll = true
				operationOption = arg
			} else if err := setHelpOperation(&opts, &operationOption, HelpOperationSearch, "--"+HelpSearchFlagName); err != nil {
				return HelpOptions{}, err
			}
			value, consumed := helpOptionValue(args, i, HelpSearchFlagName)
			i += consumed
			value = strings.TrimSpace(value)
			if value == "" {
				return HelpOptions{}, &HelpOptionError{Code: HelpOptionEmptySearch, Option: "--" + HelpSearchFlagName}
			}
			opts.SearchQuery = value
		case arg == "--"+CLIOutputFlagName || strings.HasPrefix(arg, "--"+CLIOutputFlagName+"="):
			if outputSeen {
				return HelpOptions{}, &HelpOptionError{Code: HelpOptionDuplicate, Option: "--" + CLIOutputFlagName}
			}
			outputSeen = true
			value, consumed := helpOptionValue(args, i, CLIOutputFlagName)
			i += consumed
			value = strings.TrimSpace(value)
			if value != string(HelpOutputJSON) {
				return HelpOptions{}, &HelpOptionError{Code: HelpOptionInvalidOutput, Option: "--" + CLIOutputFlagName, Value: value}
			}
			opts.Output = HelpOutputJSON
		case arg == "--"+CLISectionFlagName || strings.HasPrefix(arg, "--"+CLISectionFlagName+"="):
			if sectionSeen {
				return HelpOptions{}, &HelpOptionError{Code: HelpOptionDuplicate, Option: "--" + CLISectionFlagName}
			}
			sectionSeen = true
			value, consumed := helpOptionValue(args, i, CLISectionFlagName)
			i += consumed
			value = strings.ToLower(strings.TrimSpace(value))
			switch HelpSection(value) {
			case HelpSectionRequest, HelpSectionResponse:
				opts.Section = HelpSection(value)
				opts.SectionExplicit = true
			default:
				return HelpOptions{}, &HelpOptionError{Code: HelpOptionInvalidSection, Option: "--" + CLISectionFlagName, Value: value}
			}
		}
	}
	if opts.Operation == HelpOperationSearch {
		opts.SearchAll = true
	}
	return opts, nil
}

func setHelpOperation(opts *HelpOptions, current *string, operation HelpOperation, option string) error {
	if *current != "" {
		code := HelpOptionConflict
		if opts.Operation == operation {
			code = HelpOptionDuplicate
		}
		return &HelpOptionError{Code: code, Option: option, ConflictsWith: *current}
	}
	*current = option
	opts.Requested = true
	opts.Operation = operation
	return nil
}

func helpOptionValue(args []string, index int, name string) (string, int) {
	prefix := "--" + name + "="
	if strings.HasPrefix(args[index], prefix) {
		return strings.TrimPrefix(args[index], prefix), 0
	}
	if index+1 >= len(args) {
		return "", 0
	}
	if strings.HasPrefix(args[index+1], "-") {
		return "", 0
	}
	return args[index+1], 1
}
