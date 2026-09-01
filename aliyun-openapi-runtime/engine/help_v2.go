package engine

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
	runtimehelp "github.com/aliyun/aliyun-openapi-runtime/help"
	"github.com/aliyun/aliyun-openapi-runtime/loader"
	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

// HelpOptions is the renderer-independent Runtime Help v1 policy.
type HelpOptions = runtimehelp.HelpOptions

// HelpDocument is one of the Runtime Help v1 document types.
type HelpDocument = any

// BuildProductHelp builds product Help without rendering it.
func (e *Engine) BuildProductHelp(req Request, options HelpOptions) (*runtimehelp.ProductDocument, error) {
	ldr, err := e.getLoader()
	if err != nil {
		return nil, fmt.Errorf("openapi-runtime loader: %w", err)
	}
	if len(req.Args) == 0 || strings.TrimSpace(req.Args[0]) == "" {
		return nil, fmt.Errorf("product is required")
	}
	options = requestHelpOptions(req, options)
	return (runtimehelp.Service{Data: ldr}).BuildProduct(
		req.Args[0], scanAPIVersion(req.Args[1:]), options,
	)
}

// BuildAPIHelp builds compact Action Help by default and complete request or
// response Help when an explicit section is selected.
func (e *Engine) BuildAPIHelp(req Request, options HelpOptions) (HelpDocument, error) {
	ldr, err := e.getLoader()
	if err != nil {
		return nil, fmt.Errorf("openapi-runtime loader: %w", err)
	}
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("product and command are required")
	}
	product := strings.ToLower(strings.TrimSpace(req.Args[0]))
	command := strings.TrimSpace(req.Args[1])
	requestedVersion := scanAPIVersion(req.Args[2:])
	ref, err := ldr.ResolveCommandVersion(product, command, requestedVersion)
	if err != nil {
		return nil, err
	}
	options = requestHelpOptions(req, options)
	service := runtimehelp.Service{Data: ldr, Responses: runtimeResponseSource{loader: ldr}}
	if options.Section == runtimehelp.SectionResponse {
		return service.BuildResponse(product, ref.Version, ref.Name, options)
	}
	if options.ExplicitSection {
		return service.BuildRequest(product, ref.Version, ref.Name, options)
	}
	return service.BuildAction(product, ref.Version, ref.Name, options)
}

// BuildParameterHelp builds kebab Help for one top-level API parameter.
func (e *Engine) BuildParameterHelp(req Request, parameter string, options HelpOptions) (*runtimehelp.APIParameterDocument, error) {
	ldr, err := e.getLoader()
	if err != nil {
		return nil, fmt.Errorf("openapi-runtime loader: %w", err)
	}
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("product and command are required")
	}
	product := strings.ToLower(strings.TrimSpace(req.Args[0]))
	command := strings.TrimSpace(req.Args[1])
	ref, err := ldr.ResolveCommandVersion(product, command, scanAPIVersion(req.Args[2:]))
	if err != nil {
		return nil, err
	}
	options = requestHelpOptions(req, options)
	return (runtimehelp.Service{Data: ldr}).BuildParameter(
		product, ref.Version, ref.Name, parameter, options,
	)
}

// RenderHelp renders a structured Runtime Help v1 document.
func RenderHelp(w io.Writer, document HelpDocument, options HelpOptions) error {
	return runtimehelp.Render(w, document, options)
}

func requestHelpOptions(req Request, options HelpOptions) HelpOptions {
	if options.Language == "" {
		options.Language = req.Lang
	}
	if req.AIMode {
		options.AIMode = true
		options.Format = runtimehelp.FormatJSON
	}
	if options.RequestedVersion == "" {
		options.RequestedVersion = scanAPIVersion(req.Args[1:])
	}
	return options
}

func helpOptionsFromReserved(req Request, reserved argparser.Reserved) HelpOptions {
	options := HelpOptions{
		Search:   reserved.HelpSearch,
		All:      reserved.HelpAll,
		AIMode:   req.AIMode,
		Language: req.Lang,
	}
	if req.AIMode {
		options.Format = runtimehelp.FormatJSON
	}
	options.ExplicitSection = reserved.HelpSection != ""
	if reserved.HelpSection == "response" {
		options.Section = runtimehelp.SectionResponse
	} else {
		options.Section = runtimehelp.SectionRequest
	}
	if reserved.HelpOutput == "json" {
		options.Format = runtimehelp.FormatJSON
	} else if !req.AIMode {
		options.Format = runtimehelp.FormatText
	}
	return options
}

func validateReservedHelp(reserved argparser.Reserved, productLevel bool) error {
	if reserved.HelpExplicit && reserved.HelpSearch != "" {
		return errors.New("--help-search conflicts with --help")
	}
	if reserved.HelpExplicit && reserved.HelpAll {
		return errors.New("--help-all conflicts with --help")
	}
	if productLevel && reserved.HelpSection != "" {
		return errors.New("--cli-section requires an API target")
	}
	if reserved.HelpSection != "" && reserved.HelpAll && reserved.HelpSearch == "" {
		return &InvalidOptionCombinationError{
			Options: []string{"--cli-section", "--help-all"},
			Err:     errors.New("--cli-section does not support --help-all without --help-search"),
		}
	}
	return nil
}

func (e *Engine) tryParameterHelp(req Request, apiParameters []meta.Parameter, tail []string) (bool, error) {
	parameter, cleaned, found, err := detectParameterHelp(apiParameters, tail)
	if err != nil || !found {
		return found, err
	}
	parsed, err := argparser.ParseWithOptions(apiParameters, cleaned, argparser.ParseOptions{
		ExternalFlags: e.externalFlags,
	})
	if err != nil {
		return true, &UsageError{Code: "INVALID_ARGUMENT", Err: err}
	}
	if err := validateReservedHelp(parsed.Reserved, false); err != nil {
		return true, &UsageError{Code: "INVALID_OPTION_COMBINATION", Err: err}
	}
	if parsed.Reserved.HelpSection == "response" {
		return true, &UsageError{
			Code: "INVALID_OPTION_COMBINATION",
			Err:  errors.New("parameter Help only supports the request section"),
		}
	}
	options := helpOptionsFromReserved(req, parsed.Reserved)
	document, err := e.BuildParameterHelp(req, parameter, options)
	if err != nil {
		return true, err
	}
	return true, RenderHelp(req.Out, document, options)
}

func detectParameterHelp(parameters []meta.Parameter, args []string) (string, []string, bool, error) {
	known := make(map[string]string)
	for i := range parameters {
		parameter := &parameters[i]
		fallback := kebabName(parameter.Name)
		if fallback == "" {
			fallback = kebabName(parameter.RawName)
		}
		for _, option := range parameter.Options {
			if name := strings.TrimLeft(strings.TrimSpace(option), "-"); name != "" {
				known[name] = name
			}
		}
		if fallback != "" {
			known[fallback] = fallback
		}
	}

	targetIndex := -1
	target := ""
	for i := 0; i+1 < len(args); i++ {
		if !isParameterHelpOperation(args[i+1]) {
			continue
		}
		token := args[i]
		if !strings.HasPrefix(token, "--") || strings.Contains(token, "=") {
			continue
		}
		name := strings.TrimPrefix(token, "--")
		if canonical, ok := known[name]; ok {
			if targetIndex >= 0 {
				return "", nil, true, errors.New("parameter Help target is ambiguous")
			}
			targetIndex, target = i, canonical
		}
	}
	if targetIndex < 0 {
		return "", args, false, nil
	}
	cleaned := append([]string(nil), args[:targetIndex]...)
	cleaned = append(cleaned, args[targetIndex+1:]...)
	return target, cleaned, true, nil
}

func isParameterHelpOperation(token string) bool {
	switch token {
	case "--help", "-h", "--help-search", "--help-all":
		return true
	default:
		return false
	}
}

func kebabName(value string) string {
	var out []rune
	for i, r := range strings.TrimSpace(value) {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '-')
		}
		if r == '_' {
			r = '-'
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		out = append(out, r)
	}
	return strings.TrimLeft(string(out), "-")
}

type runtimeResponseSource struct {
	loader loader.Loader
}

func (s runtimeResponseSource) GetResponseDocumentation(product, version, apiName string) (*runtimehelp.ResponseDocumentation, error) {
	api, err := s.loader.GetAPI(product, version, apiName)
	if err != nil {
		return nil, err
	}
	section, err := api.ResponseSection()
	if err != nil {
		return nil, err
	}
	schema, err := api.ResponseSchema()
	if err != nil {
		return nil, err
	}
	return &runtimehelp.ResponseDocumentation{
		Responses:   section.Responses,
		Schema:      schema.Schema,
		StatusCode:  schema.StatusCode,
		ContentType: schema.ContentType,
		Components:  section.Components,
		Warnings:    append(section.Warnings, schema.Warnings...),
	}, nil
}
