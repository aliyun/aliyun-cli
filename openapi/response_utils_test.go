package openapi

import (
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestMergeWith(t *testing.T) {
	inter := mergeWith(nil, nil)
	assert.Nil(t, inter)
}

func TestDetectArrayPath(t *testing.T) {
	str := detectArrayPath("test")
	assert.Equal(t, "", str)

	array := map[string]interface{}{
		"test": map[string]interface{}{
			"utils": []interface{}{
				"inter",
			},
		},
	}
	str = detectArrayPath(array)
	assert.Equal(t, "test.utils[]", str)
}

func TestEvaluateExpr(t *testing.T) {
	body := []byte("test")
	str, err := evaluateExpr(body, "")
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal failed invalid character 'e' in literal true (expecting 'r')", err.Error())
	assert.Equal(t, "", str)

	body = []byte(`"test"`)
	str, err = evaluateExpr(body, "")
	assert.NotNil(t, err)
	assert.Equal(t, "jmes search failed SyntaxError: Incomplete expression", err.Error())
	assert.Equal(t, "", str)

	body = []byte(`{
           "PageNumber": 5,
	       "TotalCount": 37,
	       "PageSize": 5,
	       "RegionId": "cn-beijing",
	       "Images": {
				"Image": [{
					"ImageId": "win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd"
		         }]
           }
    }`)
	str, err = evaluateExpr(body, "RegionId")
	assert.Nil(t, err)
	assert.Equal(t, "cn-beijing", str)

	str, err = evaluateExpr(body, "Images.Image[]")
	assert.NotNil(t, err)
	assert.Equal(t, "object [map[ImageId:win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd]] isn't string", err.Error())
	assert.Equal(t, "", str)
}
