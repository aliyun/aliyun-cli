package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/jmespath/go-jmespath"
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

	fmt.Printf(string(result))
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

var pagerTestJson = []byte(`{
	"PageNumber": 5,
	"TotalCount": 37,
	"PageSize": 5,
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
