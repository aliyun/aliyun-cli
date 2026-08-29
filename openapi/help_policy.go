package openapi

const defaultHelpLogicalLineBudget = 100

// These are internal policy switches, deliberately not flags or environment
// variables. Keeping them here makes Text/JSON projection share one policy.
const (
	truncateNonAIModeTextHelp                  = true
	truncateNonAIModeJSONHelp                  = true
	showProductActionDescriptionsInDefaultHelp = false
)

// HelpResult describes the selected object range. It is present for both
// truncated and complete projections so JSON clients need no inference.
type HelpResult struct {
	Shown     int  `json:"shown"`
	Total     int  `json:"total"`
	Truncated bool `json:"truncated"`
}

// HelpNext carries only meaningful follow-up commands. Callers build commands
// with the shared target command builder so style/version/output stay intact.
type HelpNext struct {
	ShowAll string `json:"showAll,omitempty"`
	Search  string `json:"search,omitempty"`
	// SearchAll re-runs the same Help search with --help-all to lift the
	// result cap while keeping the keyword.
	SearchAll string `json:"searchAll,omitempty"`
}

// HelpProjectionMode contains only policy inputs; operation parsing remains in
// the HelpTarget layer owned by the routing module.
type HelpProjectionMode struct {
	AIMode bool
	JSON   bool
	All    bool
}

// HelpBudgetObject associates one renderer-independent object with the number
// of logical lines its complete Text rendering consumes.
type HelpBudgetObject[T any] struct {
	Value        T
	LogicalLines int
	Required     bool
}

// HelpDefaultProjectionOptions controls default/all selection. ReservedLines
// accounts for headings, usage and fixed navigation outside list objects.
type HelpDefaultProjectionOptions struct {
	Mode           HelpProjectionMode
	Budget         int
	ReservedLines  int
	RequiredSafety bool
	ShowAllCommand string
	SearchCommand  string
}

// HelpObjectProjection is shared by Text and JSON renderers. Renderers receive
// the same selected objects and must not independently truncate them.
type HelpObjectProjection[T any] struct {
	Items  []T
	Result HelpResult
	Next   *HelpNext
}

// ShouldTruncateDefaultHelp applies the internal mode switches. AI mode always
// uses compact default Help; explicit All is always complete.
func ShouldTruncateDefaultHelp(mode HelpProjectionMode) bool {
	if mode.All {
		return false
	}
	if mode.AIMode {
		return true
	}
	if mode.JSON {
		return truncateNonAIModeJSONHelp
	}
	return truncateNonAIModeTextHelp
}

// ProjectDefaultHelpObjects selects a stable prefix of complete objects under
// the 100-logical-line default budget. For Action parameters, RequiredSafety
// emits every required object first even when they alone exceed the budget;
// optional objects are then omitted.
func ProjectDefaultHelpObjects[T any](objects []HelpBudgetObject[T], options HelpDefaultProjectionOptions) HelpObjectProjection[T] {
	total := len(objects)
	if !ShouldTruncateDefaultHelp(options.Mode) {
		return HelpObjectProjection[T]{
			Items:  helpBudgetValues(objects),
			Result: HelpResult{Shown: total, Total: total},
		}
	}

	budget := options.Budget
	if budget <= 0 {
		budget = defaultHelpLogicalLineBudget
	}
	remaining := budget - options.ReservedLines
	if remaining < 0 {
		remaining = 0
	}

	selected := make([]T, 0, total)
	consumed := 0
	if options.RequiredSafety {
		for _, object := range objects {
			if !object.Required {
				continue
			}
			selected = append(selected, object.Value)
			consumed += normalizedHelpLogicalLines(object.LogicalLines)
		}
		// Required objects are rendered as one complete group before optional
		// objects. If that group exhausts the budget, no optional object is added.
	}

	if !options.RequiredSafety || consumed <= remaining {
		for _, object := range objects {
			if options.RequiredSafety && object.Required {
				continue
			}
			lines := normalizedHelpLogicalLines(object.LogicalLines)
			if consumed+lines > remaining {
				break
			}
			selected = append(selected, object.Value)
			consumed += lines
		}
	}

	result := HelpResult{Shown: len(selected), Total: total, Truncated: len(selected) < total}
	projection := HelpObjectProjection[T]{Items: selected, Result: result}
	if result.Truncated && (options.ShowAllCommand != "" || options.SearchCommand != "") {
		projection.Next = &HelpNext{ShowAll: options.ShowAllCommand, Search: options.SearchCommand}
	}
	return projection
}

func normalizedHelpLogicalLines(lines int) int {
	if lines <= 0 {
		return 1
	}
	return lines
}

func helpBudgetValues[T any](objects []HelpBudgetObject[T]) []T {
	values := make([]T, 0, len(objects))
	for _, object := range objects {
		values = append(values, object.Value)
	}
	return values
}
