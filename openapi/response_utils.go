/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/jmespath/go-jmespath"
)

// find array in json object
func detectArrayPath(d interface{}) string {
	m, ok := d.(map[string]interface{})
	if !ok {
		return ""
	}
	for k, v := range m {
		// t.Logf("%v %v\n", k, v)
		if m2, ok := v.(map[string]interface{}); ok {
			for k2, v2 := range m2 {
				if _, ok := v2.([]interface{}); ok {
					return fmt.Sprintf("%s.%s[]", k, k2)
				}
			}
		}
	}
	return ""
}

//
func mergeWith(to interface{}, from interface{}) interface{} {
	return nil
}

func evaluateExpr(body []byte, expr string) (string, error) {
	var entity interface{}
	err := json.Unmarshal(body, &entity)
	if err != nil {
		return "", fmt.Errorf("unmarshal failed %s", err)
	}

	obj, err := jmespath.Search(expr, entity)
	if err != nil {
		return "", fmt.Errorf("jmes search failed %s", err)
	}

	if s, ok := obj.(string); ok {
		return s, nil
	} else {
		return "", fmt.Errorf("object %v isn't string", obj)
	}
}
