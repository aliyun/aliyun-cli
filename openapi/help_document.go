package openapi

// localizedMachineHelpPurpose is a Text-only fallback. Structured documents
// preserve title and description separately, including an absent v12 title.
func localizedMachineHelpPurpose(title, description machineHelpLocalizedText) string {
	if value := localizedMachineHelpText(title); value != "" {
		return value
	}
	return localizedMachineHelpText(description)
}

func firstNonEmptyMachineHelpString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// machineHelpParameterLogicalLines counts the complete finite parameter
// object, including all fields and element/value descendants. It never cuts a
// nested tree to satisfy the default Help budget.
func machineHelpParameterLogicalLines(parameter machineHelpParameter) int {
	lines := 1
	for _, field := range parameter.Fields {
		lines += machineHelpParameterLogicalLines(field)
	}
	lines += machineHelpShapeLogicalLines(parameter.Element)
	lines += machineHelpShapeLogicalLines(parameter.Value)
	return lines
}

func machineHelpShapeLogicalLines(shape *machineHelpShape) int {
	if shape == nil {
		return 0
	}
	lines := 0
	for _, field := range shape.Fields {
		lines += machineHelpParameterLogicalLines(field)
	}
	lines += machineHelpShapeLogicalLines(shape.Element)
	lines += machineHelpShapeLogicalLines(shape.Value)
	return lines
}
