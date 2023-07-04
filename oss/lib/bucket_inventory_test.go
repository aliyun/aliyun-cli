package lib

import (
	"encoding/xml"
	"os"
	"strconv"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestInventoryPutSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	inventoryId := "report1"
	rulesConfigSrc := oss.InventoryConfiguration{
		Id:        inventoryId,
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1",
		},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	inventoryXml, err := xml.Marshal(&rulesConfigSrc)
	c.Assert(err, IsNil)

	inventoryFileName := "test-ossutil-file-" + randLowStr(12)
	s.createFile(inventoryFileName, string(inventoryXml), c)

	// inventory command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// check,get inventory
	inventoryDownName := inventoryFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// check inventoryDownName
	_, err = os.Stat(inventoryDownName)
	c.Assert(err, IsNil)

	inventoryBody := s.readFile(inventoryDownName, c)
	rulesConfigDest := oss.InventoryConfiguration{}
	err = xml.Unmarshal([]byte(inventoryBody), &rulesConfigDest)
	c.Assert(err, IsNil)
	c.Assert(rulesConfigDest.OSSBucketDestination.Format, Equals, rulesConfigSrc.OSSBucketDestination.Format)
	c.Assert(rulesConfigDest.OSSBucketDestination.AccountId, Equals, rulesConfigSrc.OSSBucketDestination.AccountId)
	c.Assert(rulesConfigDest.OSSBucketDestination.RoleArn, Equals, rulesConfigSrc.OSSBucketDestination.RoleArn)
	c.Assert(rulesConfigDest.OSSBucketDestination.Bucket, Equals, rulesConfigSrc.OSSBucketDestination.Bucket)
	c.Assert(rulesConfigDest.OSSBucketDestination.Prefix, Equals, rulesConfigSrc.OSSBucketDestination.Prefix)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryGetErrorResultFileOpenError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	inventoryId := "report1"
	rulesConfigSrc := oss.InventoryConfiguration{
		Id:        inventoryId,
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1"},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	inventoryXml, err := xml.Marshal(&rulesConfigSrc)
	c.Assert(err, IsNil)

	inventoryFileName := "test-ossutil-file-" + randLowStr(12)
	s.createFile(inventoryFileName, string(inventoryXml), c)

	// inventory command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// check,get inventory
	inventoryDownName := string(os.PathSeparator) + "root1" + string(os.PathSeparator) + inventoryFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	inventoryFileName := "inventory-file" + randLowStr(12)

	// inventory command test
	var str string
	strMethod := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// method is empty
	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err := cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	inventoryArgs = []string{"http://mybucket", inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(inventoryFileName)
	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(inventoryFileName, 0755)
	c.Assert(err, IsNil)
	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)
	os.Remove(inventoryFileName)

	//local file is emtpy
	s.createFile(inventoryFileName, "", c)
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)
	os.Remove(inventoryFileName)

	//local file is not xml file
	s.createFile(inventoryFileName, "aaa", c)
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)
	os.Remove(inventoryFileName)

	// StorageURLFromString error
	inventoryArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	inventoryArgs = []string{"oss:///"}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	inventoryArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	inventoryArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	//delete error,id is empty
	strMethod = "delete"
	inventoryArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	// get error,id is empty
	strMethod = "get"
	inventoryArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	os.Remove(inventoryFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryOptionsEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
	}

	versioingArgs := []string{CloudURLToString(bucketName, ""), "test-enventory-id"}
	_, err := cm.RunCommand("inventory", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	inventoryId := "report1"
	rulesConfigSrc := oss.InventoryConfiguration{
		Id:        inventoryId,
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1",
		},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	inventoryXml, err := xml.Marshal(&rulesConfigSrc)
	c.Assert(err, IsNil)

	inventoryFileName := randLowStr(12)
	s.createFile(inventoryFileName, string(inventoryXml), c)

	// inventory command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// get inventory
	inventoryDownName := inventoryFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	inventoryId := "report1"
	rulesConfigSrc := oss.InventoryConfiguration{
		Id:        inventoryId,
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1",
		},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	inventoryXml, err := xml.Marshal(&rulesConfigSrc)
	c.Assert(err, IsNil)

	inventoryFileName := randLowStr(12)
	s.createFile(inventoryFileName, string(inventoryXml), c)

	// inventory command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// get inventory
	inventoryDownName := inventoryFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// check inventoryDownName
	_, err = os.Stat(inventoryDownName)
	c.Assert(err, IsNil)
	os.Remove(inventoryDownName)

	// delete inventoryDownName
	strMethod = "delete"
	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	inventoryArgs = []string{CloudURLToString(bucketName, ""), inventoryId, inventoryDownName}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryListSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	rulesConfigSrc := oss.InventoryConfiguration{
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1",
		},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	// insert 100 inventory rule
	inventoryFileName := randLowStr(12)
	testCount := 199
	for i := 0; i < testCount; i++ {
		rulesConfigSrc.Id = strconv.Itoa(i)
		inventoryXml, err := xml.Marshal(&rulesConfigSrc)
		c.Assert(err, IsNil)
		s.createFile(inventoryFileName, string(inventoryXml), c)

		// inventory command test
		var str string
		strMethod := "put"
		options := OptionMapType{
			"endpoint":        &str,
			"accessKeyID":     &str,
			"accessKeySecret": &str,
			"stsToken":        &str,
			"configFile":      &configFile,
			"method":          &strMethod,
		}

		inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
		_, err = cm.RunCommand("inventory", inventoryArgs, options)
		c.Assert(err, IsNil)
	}

	// get inventory
	inventoryDownName := inventoryFileName + "-down"
	strMethod := "list"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryDownName}
	_, err := cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	c.Assert(bucketInventoryCommand.bwOption.ruleCount, Equals, testCount)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryListMarkerSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bl := true
	rulesConfigSrc := oss.InventoryConfiguration{
		IsEnabled: &bl,
		Prefix:    "filterPrefix/",
		OSSBucketDestination: oss.OSSBucketDestination{
			Format:    "CSV",
			AccountId: accountID,
			RoleArn:   stsARN,
			Bucket:    "acs:oss:::" + bucketName,
			Prefix:    "prefix1",
		},
		Frequency:              "Daily",
		IncludedObjectVersions: "All",
		OptionalFields: oss.OptionalFields{
			Field: []string{
				"Size", "LastModifiedDate", "ETag", "StorageClass", "IsMultipartUploaded", "EncryptionStatus",
			},
		},
	}

	// insert 100 inventory rule
	inventoryFileName := randLowStr(12)
	testCount := 80
	for i := 0; i < testCount; i++ {
		rulesConfigSrc.Id = strconv.Itoa(i)
		inventoryXml, err := xml.Marshal(&rulesConfigSrc)
		c.Assert(err, IsNil)
		s.createFile(inventoryFileName, string(inventoryXml), c)

		// inventory command test
		var str string
		strMethod := "put"
		options := OptionMapType{
			"endpoint":        &str,
			"accessKeyID":     &str,
			"accessKeySecret": &str,
			"stsToken":        &str,
			"configFile":      &configFile,
			"method":          &strMethod,
		}

		inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryFileName}
		_, err = cm.RunCommand("inventory", inventoryArgs, options)
		c.Assert(err, IsNil)
	}

	// get inventory
	inventoryDownName := inventoryFileName + "-down"
	strMethod := "list"
	str := ""
	strMarker := strconv.Itoa(testCount / 2)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"marker":          &strMarker,
	}

	inventoryArgs := []string{CloudURLToString(bucketName, ""), inventoryDownName}
	_, err := cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	inventoryBody := s.readFile(inventoryDownName, c)
	rulesConfigDest := oss.ListInventoryConfigurationsResult{}
	err = xml.Unmarshal([]byte(inventoryBody), &rulesConfigDest)

	c.Assert(len(rulesConfigDest.InventoryConfiguration) < testCount, Equals, true)

	// list to stdout
	inventoryArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// list confirm
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, IsNil)

	// list error,result file open error
	inventoryArgs = []string{CloudURLToString(bucketName, ""), string(os.PathSeparator) + "root1" + string(os.PathSeparator) + "test-file"}
	_, err = cm.RunCommand("inventory", inventoryArgs, options)
	c.Assert(err, NotNil)

	os.Remove(inventoryFileName)
	os.Remove(inventoryDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInventoryHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"inventory"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
