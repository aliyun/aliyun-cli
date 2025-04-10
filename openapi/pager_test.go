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
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	jmespath "github.com/jmespath/go-jmespath"
	"github.com/stretchr/testify/assert"

	"bytes"
	"encoding/json"
	"testing"
)

func TestPager(t *testing.T) {
	var root interface{}
	err := json.Unmarshal(pagerTestJson, &root)
	if err != nil {
		t.Errorf("%s", err)
	}

	s, err := jmespath.Search("PageNumber", root)
	if err != nil {
		t.Errorf("jmespath.Search: %s\n", err.Error())
	}

	t.Logf("PageNumber: %d\n", s)

}

func TestMerge(t *testing.T) {
	sliceKey := "Images.Image[]"

	var data1 interface{}
	err := json.Unmarshal(pagerTestJson, &data1)
	if err != nil {
		panic(err)
	}
	array1, err := jmespath.Search(sliceKey, data1)
	if err != nil {
		panic(err)
	}

	var data2 interface{}
	err = json.Unmarshal(pagerTestJson2, &data2)
	if err != nil {
		panic(err)
	}
	array2, err := jmespath.Search(sliceKey, data2)
	if err != nil {
		panic(err)
	}

	resultArray := append(array1.([]interface{}), array2.([]interface{})...)

	result, err := json.Marshal(&resultArray)
	if err != nil {
		panic(err)
	}

	cli.Printf(new(bytes.Buffer), string(result))
}

func TestSearchList(t *testing.T) {
	var data2 interface{}
	err := json.Unmarshal(pagerTestJson2, &data2)
	if err != nil {
		panic(err)
	}

	m, ok := data2.(map[string]interface{})
	if !ok {
		return
	}

	for k, v := range m {
		// t.Logf("%v %v\n", k, v)
		if m2, ok := v.(map[string]interface{}); ok {
			for k2, v2 := range m2 {
				if _, ok := v2.([]interface{}); ok {
					t.Logf("%s.%s[]", k, k2)
				}
			}
		}
	}
}

func TestPager_CallWith(t *testing.T) {
	pager := GetPager()
	assert.Nil(t, pager)

	PagerFlag.SetAssigned(true)
	pager = GetPager()
	PagerFlag.SetAssigned(false)
	assert.Equal(t, "PageNumber", pager.PageNumberFlag)
	assert.Equal(t, "PageSize", pager.PageSizeFlag)
	assert.Equal(t, 0, pager.totalCount)

	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	invoker := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			client:  client,
			request: requests.NewCommonRequest(),
		},
	}
	str, err := pager.CallWith(invoker)
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "please check your accessKey with secret, and read the user guide")
}

func TestPager_NextToken(t *testing.T) {
	pager := &Pager{
		PageNumberFlag: "PageNumber",
		PageSizeFlag:   "PageSize",
		NextTokenFlag:  "NextToken",
		PageNumberExpr: "PageNumber",
		PageSizeExpr:   "PageSize",
		TotalCountExpr: "TotalCount",
		NextTokenExpr:  "NextToken",
	}
	assert.NotNil(t, pager)
	assert.Nil(t, pager.FeedResponse(pagerTestJsonNextToken1))
	assert.True(t, pager.HasMore())
	request := requests.NewCommonRequest()
	pager.MoveNextPage(request)
	assert.Equal(t, "7a758b45874db76a0147eb118b6f597f2df87e458f7fbf48e0b5e8707e68181f", request.QueryParams["NextToken"])
	assert.Nil(t, pager.FeedResponse(pagerTestJsonNextToken2))
	assert.False(t, pager.HasMore())
	responseStr := pager.GetResponseCollection()
	var j map[string]interface{}
	assert.Nil(t, json.Unmarshal([]byte(responseStr), &j))
	assert.Equal(t, 7, len(j["TagResources"].(map[string]interface{})["TagResource"].([]interface{})))
}

func TestPager_HasMore(t *testing.T) {
	pager := Pager{
		PageSize:   5,
		totalCount: 10,
	}
	istrue := pager.HasMore()
	assert.True(t, istrue)

	pager.currentPageNumber = 3
	istrue = pager.HasMore()
	assert.False(t, istrue)

	pager.nextTokenMode = true
	pager.nextToken = "aaaaaa"
	assert.True(t, pager.HasMore())
}

func TestPager_GetResponseCollection(t *testing.T) {
	pager := Pager{
		collectionPath: "path.t[]",
	}
	str := pager.GetResponseCollection()
	assert.Equal(t, "{\"path\":{\"t\":null}}", str)
}

func TestPager_FeedResponse(t *testing.T) {
	pager := Pager{
		collectionPath: "",
		totalCount:     0,
	}
	body := "test"
	err := pager.FeedResponse(body)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal invalid character 'e' in literal true (expecting 'r')", err.Error())

	body = string(pagerTestJson)
	pager.TotalCountExpr = ""
	err = pager.FeedResponse(body)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath: '' failed SyntaxError: Incomplete expression", err.Error())

	pager.TotalCountExpr = "TotalCount"
	pager.PageNumberExpr = ""
	err = pager.FeedResponse(body)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath: '' failed SyntaxError: Incomplete expression", err.Error())

	pager.PageNumberExpr = "PageNumber"
	pager.PageSizeExpr = ""
	err = pager.FeedResponse(body)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath: '' failed SyntaxError: Incomplete expression", err.Error())

	pager.PageSizeExpr = "PageSize"
	err = pager.FeedResponse(body)
	assert.Nil(t, err)

	pager.NextTokenExpr = "NextToken"
	err = pager.FeedResponse(body)
	assert.Nil(t, err)
}

func TestPager_MoveNextPage(t *testing.T) {
	request := requests.NewCommonRequest()
	pager := Pager{
		collectionPath: "",
		totalCount:     0,
	}
	pager.MoveNextPage(request)
	assert.Equal(t, 1, pager.currentPageNumber)

	pager.nextTokenMode = true
	pager.nextToken = "aaaaa"
	pager.NextTokenFlag = "NextToken"
	pager.MoveNextPage(request)
	assert.Equal(t, "aaaaa", request.QueryParams["NextToken"])
}

func Test_detectArrayPath(t *testing.T) {
	d := "test"
	pager := Pager{
		collectionPath: "",
	}
	str := pager.detectArrayPath(d)
	assert.Equal(t, "", str)

	arr := map[string]interface{}{
		"test": "ok",
	}
	str = pager.detectArrayPath(arr)
	assert.Equal(t, "", str)
}

func Test_mergeCollections(t *testing.T) {
	pager := Pager{
		collectionPath: "",
	}
	err := pager.mergeCollections(nil)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath search failed: SyntaxError: Incomplete expression", err.Error())

	var body interface{}
	err = json.Unmarshal(pagerTestJson, &body)
	assert.Nil(t, err)

	pager.collectionPath = "test"
	err = pager.mergeCollections(body)
	assert.NotNil(t, err)
	assert.Equal(t, "jmespath result empty: test", err.Error())
}

var pagerTestJson = []byte(`{
	"PageNumber": 5,
	"TotalCount": "37",
	"PageSize": "5",
	"RegionId": "cn-beijing",
	"RequestId": "5CC3B94E-ED7E-4481-98D4-4330C5A8FCA8",
	"Images": {
		"Image": [{
			"ImageId": "win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd",
			"Description": "",
			"ProductCode": "",
			"OSType": "windows",
			"Architecture": "x86_64",
			"OSName": "Windows Server  2016 数据中心版 64位英文版",
			"DiskDeviceMappings": {
				"DiskDeviceMapping": []
			},
			"ImageOwnerAlias": "system",
			"Progress": "100%",
			"IsSupportCloudinit": true,
			"Usage": "instance",
			"CreationTime": "2017-09-29T02:20:19Z",
			"Tags": {
				"Tag": []
			},
			"ImageVersion": "",
			"Status": "Available",
			"ImageName": "win2016_64_dtc_1607_en-us_40G_alibase_20170915.vhd",
			"IsSupportIoOptimized": true,
			"IsSelfShared": "",
			"IsCopied": false,
			"IsSubscribed": false,
			"Platform": "Windows Server 2016",
			"Size": 40
		}, {
			"ImageId": "sles_11_sp4_64_20G_alibase_20170907.vhd",
			"Description": "",
			"ProductCode": "",
			"OSType": "linux",
			"Architecture": "x86_64",
			"OSName": "SUSE Linux  Enterprise Server 11 SP4 64位",
			"DiskDeviceMappings": {
				"DiskDeviceMapping": []
			},
			"ImageOwnerAlias": "system",
			"Progress": "100%",
			"IsSupportCloudinit": true,
			"Usage": "instance",
			"CreationTime": "2017-09-07T07:22:40Z",
			"Tags": {
				"Tag": []
			},
			"ImageVersion": "",
			"Status": "Available",
			"ImageName": "sles_11_sp4_64_20G_alibase_20170907.vhd",
			"IsSupportIoOptimized": true,
			"IsSelfShared": "",
			"IsCopied": false,
			"IsSubscribed": false,
			"Platform": "SUSE",
			"Size": 20
		}, {
			"ImageId": "sles_12_sp2_64_20G_alibase_20170907.vhd",
			"Description": "",
			"ProductCode": "",
			"OSType": "linux",
			"Architecture": "x86_64",
			"OSName": "SUSE Linux  Enterprise Server 12 SP2 64位",
			"DiskDeviceMappings": {
				"DiskDeviceMapping": []
			},
			"ImageOwnerAlias": "system",
			"Progress": "100%",
			"IsSupportCloudinit": true,
			"Usage": "instance",
			"CreationTime": "2017-09-07T07:22:05Z",
			"Tags": {
				"Tag": []
			},
			"ImageVersion": "",
			"Status": "Available",
			"ImageName": "sles_12_sp2_64_20G_alibase_20170907.vhd",
			"IsSupportIoOptimized": true,
			"IsSelfShared": "",
			"IsCopied": false,
			"IsSubscribed": false,
			"Platform": "SUSE",
			"Size": 20
		}, {
			"ImageId": "centos_6_09_64_20G_alibase_20170825.vhd",
			"Description": "",
			"ProductCode": "",
			"OSType": "linux",
			"Architecture": "x86_64",
			"OSName": "CentOS  6.9 64位",
			"DiskDeviceMappings": {
				"DiskDeviceMapping": []
			},
			"ImageOwnerAlias": "system",
			"Progress": "100%",
			"IsSupportCloudinit": true,
			"Usage": "instance",
			"CreationTime": "2017-09-04T10:08:49Z",
			"Tags": {
				"Tag": []
			},
			"ImageVersion": "",
			"Status": "Available",
			"ImageName": "centos_6_09_64_20G_alibase_20170825.vhd",
			"IsSupportIoOptimized": true,
			"IsSelfShared": "",
			"IsCopied": false,
			"IsSubscribed": false,
			"Platform": "CentOS",
			"Size": 20
		}, {
			"ImageId": "freebsd_11_03_64_20G_alibase_20170901.vhd",
			"Description": "",
			"ProductCode": "",
			"OSType": "linux",
			"Architecture": "x86_64",
			"OSName": "FreeBSD  11.1 64位",
			"DiskDeviceMappings": {
				"DiskDeviceMapping": []
			},
			"ImageOwnerAlias": "system",
			"Progress": "100%",
			"IsSupportCloudinit": false,
			"Usage": "instance",
			"CreationTime": "2017-09-04T10:07:59Z",
			"Tags": {
				"Tag": []
			},
			"ImageVersion": "",
			"Status": "Available",
			"ImageName": "freebsd_11_03_64_20G_alibase_20170901.vhd",
			"IsSupportIoOptimized": true,
			"IsSelfShared": "",
			"IsCopied": false,
			"IsSubscribed": false,
			"Platform": "Freebsd",
			"Size": 20
		}]
	}
}`)

var pagerTestJson2 = []byte(`{
  "Images": {
    "Image": [
      {
        "ImageId": "debian_9_02_64_20G_alibase_20171023.vhd",
        "Description": "",
        "ProductCode": "",
        "OSType": "linux",
        "Architecture": "x86_64",
        "OSName": "Debian  9.2 64位",
        "DiskDeviceMappings": {
          "DiskDeviceMapping": []
        },
        "ImageOwnerAlias": "system",
        "Progress": "100%",
        "IsSupportCloudinit": true,
        "Usage": "instance",
        "CreationTime": "2017-10-26T01:56:09Z",
        "Tags": {
          "Tag": []
        },
        "ImageVersion": "",
        "Status": "Available",
        "ImageName": "debian_9_02_64_20G_alibase_20171023.vhd",
        "IsSupportIoOptimized": true,
        "IsSelfShared": "",
        "IsCopied": false,
        "IsSubscribed": false,
        "Platform": "Debian",
        "Size": 20
      },
      {
        "ImageId": "coreos_1465_8_0_64_30G_alibase_20171024.vhd",
        "Description": "",
        "ProductCode": "",
        "OSType": "linux",
        "Architecture": "x86_64",
        "OSName": "CoreOS  1465.8.0 64位",
        "DiskDeviceMappings": {
          "DiskDeviceMapping": []
        },
        "ImageOwnerAlias": "system",
        "Progress": "100%",
        "IsSupportCloudinit": false,
        "Usage": "instance",
        "CreationTime": "2017-10-26T01:52:22Z",
        "Tags": {
          "Tag": []
        },
        "ImageVersion": "",
        "Status": "Available",
        "ImageName": "coreos_1465_8_0_64_30G_alibase_20171024.vhd",
        "IsSupportIoOptimized": true,
        "IsSelfShared": "",
        "IsCopied": false,
        "IsSubscribed": false,
        "Platform": "CoreOS",
        "Size": 30
      },
      {
        "ImageId": "m-2ze1cneefoj075diqyeh",
        "Description": "已修复打镜像过程中的test case",
        "ProductCode": "",
        "OSType": "linux",
        "Architecture": "x86_64",
        "OSName": "Ubuntu_64",
        "DiskDeviceMappings": {
          "DiskDeviceMapping": [
            {
              "ImportOSSObject": "alicloud-kvm-ubuntu.raw",
              "Format": "RAW",
              "Device": "/dev/xvda",
              "Type": "system",
              "SnapshotId": "s-2ze1cneefoj075dld00h",
              "ImportOSSBucket": "bosh-stemcell",
              "Size": "40"
            }
          ]
        },
        "ImageOwnerAlias": "others",
        "Progress": "100%",
        "IsSupportCloudinit": true,
        "Usage": "none",
        "CreationTime": "2017-10-23T12:58:20Z",
        "Tags": {
          "Tag": []
        },
        "ImageVersion": "",
        "Status": "Available",
        "ImageName": "bosh-stemcell-1008",
        "IsSupportIoOptimized": true,
        "IsSelfShared": "",
        "IsCopied": false,
        "IsSubscribed": false,
        "Platform": "Ubuntu",
        "Size": 40
      },
      {
        "ImageId": "m-2zehotvqa0rcish17xpn",
        "Description": "",
        "ProductCode": "",
        "OSType": "linux",
        "Architecture": "x86_64",
        "OSName": "Ubuntu_64",
        "DiskDeviceMappings": {
          "DiskDeviceMapping": [
            {
              "ImportOSSObject": "alicloud-kvm-ubuntu.raw",
              "Format": "RAW",
              "Device": "/dev/xvda",
              "Type": "system",
              "SnapshotId": "s-2zehotvqa0rcisgzqri5",
              "ImportOSSBucket": "bosh",
              "Size": "20"
            }
          ]
        },
        "ImageOwnerAlias": "self",
        "Progress": "100%",
        "IsSupportCloudinit": true,
        "Usage": "none",
        "CreationTime": "2017-10-16T12:46:38Z",
        "Tags": {
          "Tag": []
        },
        "ImageVersion": "",
        "Status": "Available",
        "ImageName": "bosh-stemcell-alicloud-kvm-ubuntu-trusty.raw",
        "IsSupportIoOptimized": true,
        "IsSelfShared": "",
        "IsCopied": false,
        "IsSubscribed": false,
        "Platform": "Ubuntu",
        "Size": 20
      },
      {
        "ImageId": "centos_7_04_64_20G_alibase_201701015.vhd",
        "Description": "",
        "ProductCode": "",
        "OSType": "linux",
        "Architecture": "x86_64",
        "OSName": "CentOS  7.4 64位",
        "DiskDeviceMappings": {
          "DiskDeviceMapping": []
        },
        "ImageOwnerAlias": "system",
        "Progress": "100%",
        "IsSupportCloudinit": true,
        "Usage": "instance",
        "CreationTime": "2017-10-16T07:45:45Z",
        "Tags": {
          "Tag": []
        },
        "ImageVersion": "",
        "Status": "Available",
        "ImageName": "centos_7_04_64_20G_alibase_201701015.vhd",
        "IsSupportIoOptimized": true,
        "IsSelfShared": "",
        "IsCopied": false,
        "IsSubscribed": false,
        "Platform": "CentOS",
        "Size": 20
      }
    ]
  }
}`)

const pagerTestJsonNextToken1 = `{
    "NextToken": "7a758b45874db76a0147eb118b6f597f2df87e458f7fbf48e0b5e8707e68181f",
    "RequestId": "EAB4BE0E-8CB2-4445-B671-40C8648308D0",
    "TagResources": {
        "TagResource": [
            {
                "ResourceId": "d-8vbi818ykkw7pmbvb2q7",
                "ResourceType": "disk",
                "TagKey": "AppGroup",
                "TagValue": "daily-test-ecs"
            },
            {
                "ResourceId": "d-8vbi818ykkw7pmbvb2q7",
                "ResourceType": "disk",
                "TagKey": "Provider",
                "TagValue": "sigma"
            },
            {
                "ResourceId": "d-8vb0bdoq6qxw1s11flho",
                "ResourceType": "disk",
                "TagKey": "AppGroup",
                "TagValue": "daily-test-ecs"
            },
            {
                "ResourceId": "d-8vb0bdoq6qxw1s11flho",
                "ResourceType": "disk",
                "TagKey": "Provider",
                "TagValue": "sigma"
            },
            {
                "ResourceId": "d-8vb4qthb8rk0uswg3apn",
                "ResourceType": "disk",
                "TagKey": "AppGroup",
                "TagValue": "daily-test-ecs"
            },
            {
                "ResourceId": "d-8vb4qthb8rk0uswg3apn",
                "ResourceType": "disk",
                "TagKey": "Provider",
                "TagValue": "sigma"
            }
        ]
    }
}`

const pagerTestJsonNextToken2 = `{
    "RequestId": "EADUSE0E-8CB2-4445-B671-40C8648308D0",
    "TagResources": {
        "TagResource": [
            {
                "ResourceId": "d-8vbi818ykkw7pdsadadf",
                "ResourceType": "disk",
                "TagKey": "AppGroup",
                "TagValue": "daily-test-ecs"
            }
		]
	}
}`
