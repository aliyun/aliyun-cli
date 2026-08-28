package openapi

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectDefaultHelpObjectsUsesCompleteObjectBudget(t *testing.T) {
	objects := []HelpBudgetObject[string]{
		{Value: "first", LogicalLines: 45},
		{Value: "second", LogicalLines: 45},
		{Value: "does-not-fit", LogicalLines: 20},
		{Value: "later-small-object", LogicalLines: 1},
	}

	projection := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode:           HelpProjectionMode{AIMode: true, JSON: true},
		ReservedLines:  10,
		ShowAllCommand: "aliyun ecs --help-all",
		SearchCommand:  "aliyun ecs --help-search <keyword>",
	})

	assert.Equal(t, []string{"first", "second"}, projection.Items)
	assert.Equal(t, HelpResult{Shown: 2, Total: 4, Truncated: true}, projection.Result)
	require.NotNil(t, projection.Next)
	assert.Equal(t, "aliyun ecs --help-all", projection.Next.ShowAll)
	assert.Equal(t, "aliyun ecs --help-search <keyword>", projection.Next.Search)
}

func TestProjectDefaultHelpObjectsRequiredSafetyException(t *testing.T) {
	objects := make([]HelpBudgetObject[string], 0, 6)
	for index := 0; index < 4; index++ {
		objects = append(objects, HelpBudgetObject[string]{
			Value:        fmt.Sprintf("required-%d", index),
			LogicalLines: 30,
			Required:     true,
		})
	}
	objects = append(objects,
		HelpBudgetObject[string]{Value: "optional-one", LogicalLines: 1},
		HelpBudgetObject[string]{Value: "optional-two", LogicalLines: 1},
	)

	projection := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode:           HelpProjectionMode{AIMode: true, JSON: true},
		RequiredSafety: true,
	})

	assert.Equal(t, []string{"required-0", "required-1", "required-2", "required-3"}, projection.Items)
	assert.Equal(t, HelpResult{Shown: 4, Total: 6, Truncated: true}, projection.Result)
	assert.Nil(t, projection.Next, "commands are omitted when the caller did not provide them")
}

func TestProjectDefaultHelpObjectsAllAndInternalSwitches(t *testing.T) {
	objects := []HelpBudgetObject[int]{{Value: 1, LogicalLines: 101}, {Value: 2, LogicalLines: 101}}

	all := ProjectDefaultHelpObjects(objects, HelpDefaultProjectionOptions{
		Mode: HelpProjectionMode{All: true},
	})
	assert.Equal(t, []int{1, 2}, all.Items)
	assert.Equal(t, HelpResult{Shown: 2, Total: 2, Truncated: false}, all.Result)
	assert.Nil(t, all.Next)

	assert.True(t, ShouldTruncateDefaultHelp(HelpProjectionMode{AIMode: true, JSON: true}))
	assert.True(t, ShouldTruncateDefaultHelp(HelpProjectionMode{JSON: true}))
	assert.True(t, ShouldTruncateDefaultHelp(HelpProjectionMode{}))
	assert.False(t, ShouldTruncateDefaultHelp(HelpProjectionMode{All: true}))
	assert.False(t, showProductActionDescriptionsInDefaultHelp)
}
