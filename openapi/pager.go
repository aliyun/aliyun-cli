package openapi

import (
	"github.com/jmespath/go-jmespath"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"strconv"
	"math"
)

type Pager struct {
	PageNumberFlag string
	PageSizeFlag string

	PageNumberExpr string
	PageSizeExpr string
	TotalCountExpr string

	PageSize int

	totalCount int
	currentPageNumber int
	collectionName string
	collectionPath string

	results []interface{}
}

func NewPager(name string) (*Pager) {
	pager := &Pager {
		PageNumberFlag: "PageNumber",
		PageSizeFlag: "PageSize",
		PageNumberExpr: "PageNumber",
		PageSizeExpr: "PageSize",
		TotalCountExpr: "TotalCount",
	}

	pager.collectionName = name
	pager.collectionPath = name + "." + name[0: len(name) - 1] + "[]"
	// fmt.Printf("collection path: %s", pager.collectionPath)
	return pager
}

func (a *Pager) HasMore() bool {
	pages := int(math.Ceil(float64(a.totalCount) / float64(a.PageSize)))
	if a.currentPageNumber >= pages {
		return false
	} else {
		return true
	}
}

func (a *Pager) GetResponseCollection() string {
	r := make(map[string]interface{})
	r[a.collectionName] = a.results
	// return json.Mar
	s, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return string(s)
}

func (a *Pager) FeedResponse(body []byte) error {
	var j interface{}
	err := json.Unmarshal(body, &j)
	if err != nil {
		return fmt.Errorf("unmarshal %s", err.Error())
	}

	if total, err := jmespath.Search(a.TotalCountExpr, j); err == nil {
		a.totalCount = int(total.(float64))
	} else {
		return fmt.Errorf("jmespath: '%s' failed %s", a.TotalCountExpr, err)
	}

	if pageNumber, err := jmespath.Search(a.PageNumberExpr, j); err == nil {
		a.currentPageNumber = int(pageNumber.(float64))
	 } else {
		return fmt.Errorf("jmespath: '%s' failed %s", a.PageNumberExpr, err)
	}

	if pageSize, err := jmespath.Search(a.PageSizeExpr, j); err == nil {
		a.PageSize = int(pageSize.(float64))
	} else {
		return fmt.Errorf("jmespath: '%s' failed %s", a.PageSizeExpr, err)
	}

	a.mergeCollections(j)
	return nil
}

func (a *Pager) MoveNextPage(request *requests.CommonRequest) {
	a.currentPageNumber = a.currentPageNumber + 1
	// fmt.Printf("Move to page %d", a.currentPageNumber)
	request.QueryParams[a.PageNumberFlag] = strconv.Itoa(a.currentPageNumber)
}

func (a *Pager) mergeCollections(body interface{}) error {
	ar, err := jmespath.Search(a.collectionPath, body)
	if err != nil {
		return fmt.Errorf("jmespath search failed: %s", err.Error())
	} else if ar == nil {
		return fmt.Errorf("jmespath result empty: %s", a.collectionPath)
	}
	for _, i := range ar.([]interface{}) {
		a.results = append(a.results, i)
	}
	return nil
}

