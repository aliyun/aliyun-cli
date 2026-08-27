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

	if document.OutputSchema.Components != nil && len(document.OutputSchema.Components.Schemas) > 0 {
		if _, err := fmt.Fprintln(w, "\nComponents:"); err != nil {
			return err
		}
		encoded, err := json.Marshal(struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		}{Schemas: document.OutputSchema.Components.Schemas})
		if err != nil {
			return err
		}
		if err := writeIndentedJSON(w, encoded); err != nil {
			return err
		}
	}

	if len(document.Warnings) > 0 {
		if _, err := fmt.Fprintln(w, "\nWarnings:"); err != nil {
			return err
		}
		for _, warning := range document.Warnings {
			if _, err := fmt.Fprintf(w, "- %s\n", warning); err != nil {
				return err
			}
		}
	}

	if document.ResponseQuery != nil {
		if _, err := fmt.Fprintf(w, "\nQuery this array directly:\n  %s\n", document.ResponseQuery.QueryCommand); err != nil {
			return err
		}
	}
	return nil
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
