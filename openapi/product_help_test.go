package openapi

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
)

func TestPlanProductLevelHelp_BuiltinPluginNotInstalled(t *testing.T) {
	c := NewCommando(new(bytes.Buffer), config.Profile{Language: "en"})
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-ecs", ProductCode: "ecs"}},
	}
	c.localManifest = &plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{}}

	plan, _, pluginName, hasBuiltIn := c.planProductLevelHelp("ecs")
	assert.True(t, hasBuiltIn)
	assert.Equal(t, "aliyun-cli-ecs", pluginName)
	assert.True(t, plan.deliverBuiltInHelp)
	assert.True(t, plan.showInstallHint)
	assert.False(t, plan.deliverPluginHelp)
	assert.Nil(t, plan.abortErr)
}

func TestPlanProductLevelHelp_BuiltinPluginInstalled(t *testing.T) {
	c := NewCommando(new(bytes.Buffer), config.Profile{Language: "en"})
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-ecs", ProductCode: "ecs"}},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{"aliyun-cli-ecs": {Name: "aliyun-cli-ecs"}},
	}

	plan, _, _, hasBuiltIn := c.planProductLevelHelp("ecs")
	assert.True(t, hasBuiltIn)
	assert.True(t, plan.deliverPluginHelp)
	assert.False(t, plan.deliverBuiltInHelp)
}

func TestPlanProductLevelHelp_ShowOriginalWithInstalledPlugin(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP", "1")
	c := NewCommando(new(bytes.Buffer), config.Profile{Language: "en"})
	c.library.builtinRepo = getRepository()
	c.pluginIndex = &plugin.Index{
		Plugins: []plugin.PluginInfo{{Name: "aliyun-cli-ecs", ProductCode: "ecs"}},
	}
	c.localManifest = &plugin.LocalManifest{
		Plugins: map[string]plugin.LocalPlugin{"aliyun-cli-ecs": {Name: "aliyun-cli-ecs"}},
	}

	plan, _, _, hasBuiltIn := c.planProductLevelHelp("ecs")
	assert.True(t, hasBuiltIn)
	assert.False(t, plan.deliverPluginHelp)
	assert.True(t, plan.deliverBuiltInHelp)
}

func TestPlanProductLevelHelp_UnknownProduct(t *testing.T) {
	repo, _ := meta.MockLoadRepository([]meta.Product{})
	c := NewCommando(new(bytes.Buffer), config.Profile{Language: "en"})
	c.library.builtinRepo = repo
	c.pluginIndex = &plugin.Index{}

	plan, _, _, hasBuiltIn := c.planProductLevelHelp("nope")
	assert.False(t, hasBuiltIn)
	assert.Error(t, plan.abortErr)
	assert.IsType(t, &InvalidProductOrPluginError{}, plan.abortErr)
}

func TestInvalidProductOrPluginError_SuggestsBuiltinAndPlugin(t *testing.T) {
	err := &InvalidProductOrPluginError{
		Code: "ec",
		plugins: []plugin.PluginInfo{
			{Name: "aliyun-cli-ecs", ProductCode: "ecs"},
		},
		library: &Library{
			builtinRepo: &meta.Repository{
				Products: []meta.Product{{Code: "ecs"}, {Code: "eci"}},
			},
		},
	}
	suggestions := err.GetSuggestions()
	assert.Contains(t, suggestions, "ecs")
}
