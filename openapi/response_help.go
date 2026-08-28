package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func renderResponseHelpText(w io.Writer, document *machineHelpAPIResponseDocument) error {
	if document == nil {
		return fmt.Errorf("response Help document is nil")
	}
	// Explicit Response Section Help is lossless: retain every response and
	// print only its reachable component closure without inlining refs. Search
	// projections continue through the filtered OutputSchema branch below.
	if len(bytes.TrimSpace(document.Responses)) > 0 && len(document.Matches) == 0 && document.Notice == "" {
		if _, err := fmt.Fprintln(w, "Responses:"); err != nil {
			return err
		}
		if err := writeIndentedJSON(w, document.Responses); err != nil {
			return err
		}
		if err := renderMachineHelpComponents(w, document.Components); err != nil {
			return err
		}
		if err := renderMachineHelpResponseWarnings(w, document.Warnings); err != nil {
			return err
		}
		if err := renderMachineHelpResponseQuery(w, document.ResponseQuery); err != nil {
			return err
		}
		return renderHelpProjectionResult(w, "matches", document.Result, document.Next)
	}
	if document.OutputSchema == nil {
		_, err := fmt.Fprintln(w, document.Notice)
		return err
	}
	if len(document.Matches) > 0 {
		if _, err := fmt.Fprintln(w, "Matched Response Paths:"); err != nil {
			return err
		}
		for _, path := range document.Matches {
			if _, err := fmt.Fprintf(w, "- %s\n", path); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	heading := fmt.Sprintf("Response Schema (HTTP %s", document.OutputSchema.StatusCode)
	if document.OutputSchema.ContentType != "" {
		heading += ", " + document.OutputSchema.ContentType
	}
	if _, err := fmt.Fprintf(w, "%s):\n", heading); err != nil {
		return err
	}
	if err := writeIndentedJSON(w, document.OutputSchema.Schema); err != nil {
		return err
	}

	if err := renderMachineHelpComponents(w, document.OutputSchema.Components); err != nil {
		return err
	}

	if err := renderMachineHelpResponseWarnings(w, document.Warnings); err != nil {
		return err
	}
	if err := renderMachineHelpResponseQuery(w, document.ResponseQuery); err != nil {
		return err
	}
	return renderHelpProjectionResult(w, "matches", document.Result, document.Next)
}

func renderMachineHelpComponents(w io.Writer, components *machineHelpComponents) error {
	if components == nil || len(components.Schemas) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nComponents:"); err != nil {
		return err
	}
	encoded, err := json.Marshal(struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	}{Schemas: components.Schemas})
	if err != nil {
		return err
	}
	return writeIndentedJSON(w, encoded)
}

func renderMachineHelpResponseWarnings(w io.Writer, warnings []string) error {
	if len(warnings) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nWarnings:"); err != nil {
		return err
	}
	for _, warning := range warnings {
		if _, err := fmt.Fprintf(w, "- %s\n", warning); err != nil {
			return err
		}
	}
	return nil
}

func renderMachineHelpResponseQuery(w io.Writer, query *machineHelpQueryExample) error {
	if query == nil {
		return nil
	}
	_, err := fmt.Fprintf(w, "\nQuery this array directly:\n  %s\n", query.QueryCommand)
	return err
}

func writeIndentedJSON(w io.Writer, raw json.RawMessage) error {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, raw, "", "  "); err != nil {
		return err
	}
	formatted.WriteByte('\n')
	_, err := w.Write(formatted.Bytes())
	return err
}

func mergeMachineHelpWarnings(groups ...[]string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, group := range groups {
		for _, warning := range group {
			if warning == "" || seen[warning] {
				continue
			}
			seen[warning] = true
			result = append(result, warning)
		}
	}
	return result
}

func completeMachineHelpJSONResult(raw json.RawMessage) HelpResult {
	var entries map[string]json.RawMessage
	if json.Unmarshal(raw, &entries) != nil {
		return HelpResult{}
	}
	return HelpResult{Shown: len(entries), Total: len(entries)}
}
