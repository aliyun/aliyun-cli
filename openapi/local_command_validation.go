package openapi

import (
	"errors"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/openapi/runtimehost"
)

// validateCanonicalAPICommand performs only metadata-local command identity
// validation. Request validation and execution continue through the existing
// paths after profile resolution.
func (c *Commando) validateCanonicalAPICommand(args []string, ctx *cli.Context) error {
	if len(args) != 2 || c.library == nil || c.library.helpRepo == nil {
		return nil
	}
	method := args[1]
	if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" {
		return nil
	}
	if EstimateCostFlag(ctx.Flags()).IsAssigned() || ForceFlag(ctx.Flags()).IsAssigned() {
		return nil
	}
	version := ""
	if flag, value := rawExplicitVersion(ctx.InvocationArgs()); flag != "" {
		version = value
	}
	resolved, err := newMachineHelpService(c.library.helpRepo).resolveAPI(args[0], args[1], version)
	if err == nil {
		if resolved == nil || resolved.API == nil {
			return nil
		}
		ctx.SetParsedFlagValidator(legacyParameterFlagValidator(resolved.API))
		if ctx.UnknownFlags() == nil {
			return nil
		}
		for _, flag := range ctx.UnknownFlags().Flags() {
			if flag == nil || !flag.IsAssigned() {
				continue
			}
			name := strings.TrimSuffix(flag.Name, "-FILE")
			if resolved.API.FindLegacyParameter(name) == nil {
				return NewInvalidParameterErrorFromCanonical(name, resolved.API, args[0], ctx.Flags())
			}
		}
		return nil
	}
	var structured *machineHelpError
	if !errors.As(err, &structured) {
		// Repository I/O failures should remain on the established execution
		// path instead of making ordinary API execution depend on Help storage.
		return nil
	}
	switch structured.document.Error.Code {
	case "UNKNOWN_API":
		product, _ := c.library.GetProduct(args[0])
		return &InvalidApiError{Name: args[1], product: &product}
	case "UNKNOWN_PRODUCT":
		return &InvalidProductError{Code: args[0], library: c.library}
	default:
		return nil
	}
}

// legacyParameterFlagValidator confirms parser-recognized dynamic flags against
// the API already loaded by the execution path. The execution lookup tolerates
// arbitrary RepeatList suffixes; telemetry accepts only fixed metadata names and
// numeric array indexes, excluding business data used as map keys or indexes.
func legacyParameterFlagValidator(api *canonicalmeta.API) func(string) bool {
	if api == nil {
		return nil
	}
	parameters := api.LegacyTopLevelParameters()
	return func(flag string) bool {
		if !strings.HasPrefix(flag, "--") {
			return false
		}
		name := strings.TrimSuffix(flag[2:], "-FILE")
		return confirmedLegacyParameterName(parameters, name)
	}
}

func confirmedLegacyParameterName(parameters []*canonicalmeta.LegacyParameterView, name string) bool {
	for _, parameter := range parameters {
		if parameter == nil {
			continue
		}
		fixedName := parameter.LegacyName()
		if name == fixedName {
			return true
		}
		suffix, indexed := strings.CutPrefix(name, fixedName+".")
		if !indexed || !parameter.IsLegacyRepeatList() {
			continue
		}
		index, remainder, hasChild := strings.Cut(suffix, ".")
		if index == "" {
			continue
		}
		numeric := true
		for _, digit := range index {
			if digit < '0' || digit > '9' {
				numeric = false
				break
			}
		}
		if !numeric {
			continue
		}
		children := parameter.LegacyChildren()
		if hasChild {
			if confirmedLegacyParameterName(children, remainder) {
				return true
			}
		} else if len(children) == 0 {
			return true
		}
	}
	return false
}

// validateCanonicalRuntimeCommand validates a command for products already
// present in Canonical metadata without preventing unknown/plugin-only
// products from continuing through the established runtime and auto-install
// routing paths.
func (c *Commando) validateCanonicalRuntimeCommand(args []string, ctx *cli.Context) error {
	// This is only a routing preflight. The runtime or plugin selected next
	// owns its parameter definitions; do not retain the legacy schema observer.
	defer ctx.SetParsedFlagValidator(nil)
	err := c.validateCanonicalAPICommand(args, ctx)
	var invalidProduct *InvalidProductError
	var invalidParameter *InvalidParameterError
	if errors.As(err, &invalidProduct) || errors.As(err, &invalidParameter) {
		return nil
	}
	var invalidAPI *InvalidApiError
	if len(args) == 2 && strings.ToLower(args[1]) == args[1] && errors.As(err, &invalidAPI) {
		candidates := runtimehost.ProductCommands(args[0])
		if len(candidates) > 0 {
			return &InvalidBaselineCommandError{
				Product:    args[0],
				Command:    args[1],
				Candidates: candidates,
				Err:        err,
			}
		}
	}
	return err
}
