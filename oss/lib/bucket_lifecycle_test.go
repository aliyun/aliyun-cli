package lib

import (
	"encoding/xml"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestLifecyclePutSuccess(c *C) {
	lifecycleXml := `<?xml version="1.0" encoding="UTF-8"?>
    <LifecycleConfiguration>
        <Rule>
            <Prefix>testdelete</Prefix>
            <Status>Enabled</Status>
            <Expiration>
                <Days>240</Days>
            </Expiration>
            <AbortMultipartUpload>
                <Days>30</Days>
            </AbortMultipartUpload>
        </Rule>
    </LifecycleConfiguration>`

	rulesConfigSrc := oss.LifecycleConfiguration{}
	err := xml.Unmarshal([]byte(lifecycleXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	lifecycleFileName := randLowStr(12)
	s.createFile(lifecycleFileName, lifecycleXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// lifecycle command test
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

	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// check,get lifecycle
	lifecycleDownName := lifecycleFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// check lifecycleDownName
	_, err = os.Stat(lifecycleDownName)
	c.Assert(err, IsNil)

	lifecycleBody := s.readFile(lifecycleDownName, c)

	rulesConfigDest := oss.LifecycleConfiguration{}
	err = xml.Unmarshal([]byte(lifecycleBody), &rulesConfigDest)
	c.Assert(err, IsNil)
	c.Assert(len(rulesConfigSrc.Rules), Equals, len(rulesConfigDest.Rules))
	c.Assert(rulesConfigSrc.Rules[0].Prefix, Equals, rulesConfigDest.Rules[0].Prefix)
	c.Assert(rulesConfigSrc.Rules[0].Status, Equals, rulesConfigDest.Rules[0].Status)
	c.Assert(rulesConfigSrc.Rules[0].Expiration.Days, Equals, rulesConfigDest.Rules[0].Expiration.Days)
	c.Assert(rulesConfigSrc.Rules[0].AbortMultipartUpload.Days, Equals, rulesConfigDest.Rules[0].AbortMultipartUpload.Days)

	os.Remove(lifecycleFileName)
	os.Remove(lifecycleDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestLifecyclePutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	lifecycleFileName := "lifecycle-file" + randLowStr(12)

	// lifecycle command test
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
	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err := cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	lifecycleArgs = []string{"http://mybucket", lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(lifecycleFileName)
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(lifecycleFileName, 0755)
	c.Assert(err, IsNil)
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(lifecycleFileName)

	//local file is emtpy
	s.createFile(lifecycleFileName, "", c)
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(lifecycleFileName)

	//local file is not xml file
	s.createFile(lifecycleFileName, "aaa", c)
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(lifecycleFileName)

	// StorageURLFromString error
	lifecycleArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	lifecycleArgs = []string{"oss:///"}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	lifecycleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	lifecycleArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	os.Remove(lifecycleFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestLifecycleOptionsEmptyEndpoint(c *C) {
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

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("lifecycle", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestLifecycleGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	lifecycleXml := `<?xml version="1.0" encoding="UTF-8"?>
    <LifecycleConfiguration>
      <Rule>
        <Prefix>testdelete</Prefix>
        <Status>Enabled</Status>
        <Expiration>
            <Days>240</Days>
        </Expiration>
        <AbortMultipartUpload>
            <Days>30</Days>
        </AbortMultipartUpload>
     </Rule>
    </LifecycleConfiguration>`

	lifecycleFileName := randLowStr(12)
	s.createFile(lifecycleFileName, lifecycleXml, c)

	// lifecycle command test
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

	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err := cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// get lifecycle
	lifecycleDownName := lifecycleFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	lifecycleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	os.Remove(lifecycleFileName)
	os.Remove(lifecycleDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestLifecycleDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	lifecycleXml := `<?xml version="1.0" encoding="UTF-8"?>
      <LifecycleConfiguration>
        <Rule>
           <Prefix>testdelete</Prefix>
           <Status>Enabled</Status>
           <Expiration>
              <Days>240</Days>
           </Expiration>
           <AbortMultipartUpload>
             <Days>30</Days>
           </AbortMultipartUpload>
        </Rule>
      </LifecycleConfiguration>`

	lifecycleFileName := randLowStr(12)
	s.createFile(lifecycleFileName, lifecycleXml, c)

	// lifecycle command test
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

	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err := cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// get lifecycle
	lifecycleDownName := lifecycleFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// check lifecycleDownName
	_, err = os.Stat(lifecycleDownName)
	c.Assert(err, IsNil)
	os.Remove(lifecycleDownName)

	// delete lifecycleDownName
	strMethod = "delete"
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// get again
	strMethod = "get"
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, NotNil)

	os.Remove(lifecycleFileName)
	os.Remove(lifecycleDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestLifecycleHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"lifecycle"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestLifecyclePutOverLapSuccess(c *C) {
	lifecycleXml := `<?xml version="1.0" encoding="UTF-8"?>
    <LifecycleConfiguration>
        <Rule>
            <Prefix>testdelete</Prefix>
            <Status>Enabled</Status>
            <Expiration>
                <Days>240</Days>
            </Expiration>
            <AbortMultipartUpload>
                <Days>30</Days>
            </AbortMultipartUpload>
        </Rule>
    </LifecycleConfiguration>`

	lifecycleFileName := randLowStr(12)
	s.createFile(lifecycleFileName, lifecycleXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// lifecycle command test
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

	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err := cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)
	os.Remove(lifecycleFileName)
	s.removeBucket(bucketName, true, c)
}
