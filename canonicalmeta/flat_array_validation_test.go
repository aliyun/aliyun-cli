package canonicalmeta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCFlatArrayRejectsInvalidIndices(t *testing.T) {
	api := &API{Operation: &Operation{APIStyle: "RPC"}, Parameters: []Parameter{{RawName: "Ids", Type: "array", ParamStyle: "flat", Location: "query", Element: &TypeShape{Type: "string"}}}}
	require.NotNil(t, api.FindLegacyParameter("Ids.1"))
	for _, name := range []string{"Ids.0", "Ids.-1", "Ids.+1", "Ids.bad", "Ids.1.Field"} {
		require.Nil(t, api.FindLegacyParameter(name), name)
	}
}
