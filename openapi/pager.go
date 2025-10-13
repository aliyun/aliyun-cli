// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	jmespath "github.com/jmespath/go-jmespath"
)

var PagerFlag = &cli.Flag{Category: "caller",
	Name:         "pager",
	Hidden:       false,
	AssignedMode: cli.AssignedRepeatable,
	Aliases:      []string{"--all-pages"},
	Short: i18n.T(
		"use `--pager` to merge pages for pageable APIs",
		"使用 `--pager` 在访问分页的API时合并结果分页"),
	Fields: []cli.Field{
		{Key: "path", Required: false},
		{Key: "PageNumber", DefaultValue: "PageNumber", Short: i18n.T(" PageNumber", "指定PageNumber的属性")},
		{Key: "PageSize", DefaultValue: "PageSize", Short: i18n.T("PageSize", "")},
		{Key: "TotalCount", DefaultValue: "TotalCount", Short: i18n.T("TotalCount", "")},
		{Key: "NextToken", DefaultValue: "NextToken", Short: i18n.T("NextToken", "")},
	},
	ExcludeWith: []string{WaiterFlag.Name},
}

type Pager struct {
	PageNumberFlag string
	PageSizeFlag   string
	NextTokenFlag  string

	PageNumberExpr string
	PageSizeExpr   string
	TotalCountExpr string
	NextTokenExpr  string

	PageSize int

	totalCount        int
	currentPageNumber int
	nextTokenMode     bool
	nextToken         string
	collectionPath    string

	results []interface{}
}

func GetPager() *Pager {
	if !PagerFlag.IsAssigned() {
		return nil
	}
	pager := &Pager{}
	pageNumberFlagTemp, _ := PagerFlag.GetFieldValue("PageNumber")
	tempStr := strings.Split(pageNumberFlagTemp, ".")
	pager.PageNumberFlag = tempStr[len(tempStr)-1]
	pageSizeFlagTemp, _ := PagerFlag.GetFieldValue("PageSize")
	tempStr = strings.Split(pageSizeFlagTemp, ".")
	pager.PageSizeFlag = tempStr[len(tempStr)-1]
	pager.PageNumberExpr, _ = PagerFlag.GetFieldValue("PageNumber")
	pager.PageSizeExpr, _ = PagerFlag.GetFieldValue("PageSize")
	pager.TotalCountExpr, _ = PagerFlag.GetFieldValue("TotalCount")

	nextTokenFlagTemp, _ := PagerFlag.GetFieldValue("NextToken")
	tempStr = strings.Split(nextTokenFlagTemp, ".")
	pager.NextTokenFlag = tempStr[len(tempStr)-1]
	pager.NextTokenExpr = nextTokenFlagTemp

	pager.collectionPath, _ = PagerFlag.GetFieldValue("path")
	return pager
}

func (a *Pager) CallWith(invoker Invoker) (string, error) {
	for {

		resp, err := invoker.Call()
		if err != nil {
			return "", err
		}

		err = a.FeedResponse(resp.GetHttpContentString())
		if err != nil {
			return "", fmt.Errorf("call failed %s", err)
		}

		if !a.HasMore() {
			break
		}
		a.MoveNextPage(invoker.getRequest())
	}
	return a.GetResponseCollection(), nil
}

func (a *Pager) HasMore() bool {
	if a.nextTokenMode {
		return a.nextToken != ""
	}
	pages := int(math.Ceil(float64(a.totalCount) / float64(a.PageSize)))
	return a.currentPageNumber < pages
}

func (a *Pager) GetResponseCollection() string {
	root := make(map[string]interface{})
	current := make(map[string]interface{})
	path := a.collectionPath
	l := strings.Index(path, ".")
	tempSlice := strings.Split(path, ".")
	if l > 0 {
		tempSlice = tempSlice[len(tempSlice)-2:]
		root[tempSlice[0]] = current
	} else {
		root = current
	}
	key := strings.TrimSuffix(tempSlice[len(tempSlice)-1], "[]")
	current[key] = a.results

	s, err := json.Marshal(root)
	if err != nil {
		panic(err)
	}
	return string(s)
}

func (a *Pager) FeedResponse(body string) error {
	var j interface{}

	err := json.Unmarshal([]byte(body), &j)
	if err != nil {
		return fmt.Errorf("unmarshal %s", err.Error())
	}

	a.nextToken = ""

	if a.NextTokenExpr != "" {
		a.nextTokenMode = true
		// allow to ignore NextToken mode
		if val, err := jmespath.Search(a.NextTokenExpr, j); err == nil {
			if nextToken, ok := val.(string); ok {
				a.nextToken = nextToken
			}
		} else {
			return fmt.Errorf("jmespath: '%s' failed %s", a.NextTokenExpr, err)
		}
	}

	if !a.nextTokenMode {
		if total, err := jmespath.Search(a.TotalCountExpr, j); err == nil {
			var totalCount float64
			if strCount, ok := total.(string); ok {
				totalCount, _ = strconv.ParseFloat(strCount, 64)
			} else {
				totalCount = total.(float64)
			}
			a.totalCount = int(totalCount)
		} else {
			return fmt.Errorf("jmespath: '%s' failed %s", a.TotalCountExpr, err)
		}

		if pageNumber, err := jmespath.Search(a.PageNumberExpr, j); err == nil {
			var currentPageNumber float64
			if strpageNumber, ok := pageNumber.(string); ok {
				currentPageNumber, _ = strconv.ParseFloat(strpageNumber, 64)
			} else {
				currentPageNumber = pageNumber.(float64)
			}
			a.currentPageNumber = int(currentPageNumber)
		} else {
			return fmt.Errorf("jmespath: '%s' failed %s", a.PageNumberExpr, err)
		}

		if pageSize, err := jmespath.Search(a.PageSizeExpr, j); err == nil {
			var PageSize float64
			if strpageSize, ok := pageSize.(string); ok {
				PageSize, _ = strconv.ParseFloat(strpageSize, 64)
			} else {
				PageSize = pageSize.(float64)
			}
			a.PageSize = int(PageSize)
		} else {
			return fmt.Errorf("jmespath: '%s' failed %s", a.PageSizeExpr, err)
		}
	}

	if a.collectionPath == "" {
		p2 := a.detectArrayPath(j)
		if p2 == "" {
			return fmt.Errorf("can't auto recognize collections path: you need add `--pager path=[jmespath]` to assign manually")
		}
		a.collectionPath = p2
	}
	a.mergeCollections(j)
	return nil
}

func (a *Pager) MoveNextPage(request *requests.CommonRequest) {
	if a.nextTokenMode {
		request.QueryParams[a.NextTokenFlag] = a.nextToken
	} else {
		a.currentPageNumber = a.currentPageNumber + 1
		// cli.Printf("Move to page %d", a.currentPageNumber)
		request.QueryParams[a.PageNumberFlag] = strconv.Itoa(a.currentPageNumber)
	}
}

func (a *Pager) mergeCollections(body interface{}) error {
	ar, err := jmespath.Search(a.collectionPath, body)
	if err != nil {
		return fmt.Errorf("jmespath search failed: %s", err.Error())
	} else if ar == nil {
		return fmt.Errorf("jmespath result empty: %s", a.collectionPath)
	}
	a.results = append(a.results, ar.([]interface{})...)
	return nil
}

func (a *Pager) detectArrayPath(d interface{}) string {
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
