package lib

import (
	"encoding/xml"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestWebsitePutSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	websiteXml := `<?xml version="1.0" encoding="UTF-8"?>
    <WebsiteConfiguration>
    <IndexDocument>
        <Suffix>index.html</Suffix>
    </IndexDocument>
    <ErrorDocument>
        <Key>error.html</Key>
    </ErrorDocument>
    <RoutingRules>
     <RoutingRule>
            <RuleNumber>1</RuleNumber>
            <Condition>
                <KeyPrefixEquals>abc/</KeyPrefixEquals>
                <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
            </Condition>
            <Redirect>
                <RedirectType>Mirror</RedirectType>
                <PassQueryString>true</PassQueryString>
                <MirrorURL>http://www.test.com/</MirrorURL>
                <MirrorPassQueryString>true</MirrorPassQueryString>
                <MirrorFollowRedirect>true</MirrorFollowRedirect>
                <MirrorCheckMd5>false</MirrorCheckMd5>
            </Redirect>
        </RoutingRule>
        </RoutingRules>
    </WebsiteConfiguration>`

	rulesConfigSrc := oss.WebsiteXML{}
	err := xml.Unmarshal([]byte(websiteXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	websiteFileName := randLowStr(12)
	s.createFile(websiteFileName, websiteXml, c)

	// website command test
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

	websiteArgs := []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	// check,get website
	websiteDownName := websiteFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	// check websiteDownName
	_, err = os.Stat(websiteDownName)
	c.Assert(err, IsNil)

	websiteBody := s.readFile(websiteDownName, c)

	rulesConfigDest := oss.WebsiteXML{}
	err = xml.Unmarshal([]byte(websiteBody), &rulesConfigDest)
	c.Assert(err, IsNil)
	c.Assert(rulesConfigSrc.IndexDocument.Suffix, Equals, rulesConfigDest.IndexDocument.Suffix)
	c.Assert(rulesConfigSrc.ErrorDocument.Key, Equals, rulesConfigDest.ErrorDocument.Key)
	c.Assert(rulesConfigSrc.RoutingRules[0].RuleNumber, Equals, rulesConfigDest.RoutingRules[0].RuleNumber)
	c.Assert(rulesConfigSrc.RoutingRules[0].Condition.KeyPrefixEquals, Equals, rulesConfigDest.RoutingRules[0].Condition.KeyPrefixEquals)
	c.Assert(rulesConfigSrc.RoutingRules[0].Condition.HTTPErrorCodeReturnedEquals, Equals, rulesConfigDest.RoutingRules[0].Condition.HTTPErrorCodeReturnedEquals)
	c.Assert(rulesConfigSrc.RoutingRules[0].Redirect.MirrorURL, Equals, rulesConfigDest.RoutingRules[0].Redirect.MirrorURL)

	os.Remove(websiteFileName)
	os.Remove(websiteDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWebsitePutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	websiteFileName := "website-file" + randLowStr(12)

	// website command test
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
	websiteArgs := []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err := cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	websiteArgs = []string{"http://mybucket", websiteFileName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(websiteFileName)
	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(websiteFileName, 0755)
	c.Assert(err, IsNil)
	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)
	os.Remove(websiteFileName)

	//local file is emtpy
	s.createFile(websiteFileName, "", c)
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)
	os.Remove(websiteFileName)

	//local file is not xml file
	s.createFile(websiteFileName, "aaa", c)
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)
	os.Remove(websiteFileName)

	// StorageURLFromString error
	websiteArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	websiteArgs = []string{"oss:///"}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	websiteArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	websiteArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	os.Remove(websiteFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWebsiteOptionsEmptyEndpoint(c *C) {
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
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("website", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWebsiteGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	websiteXml := `<?xml version="1.0" encoding="UTF-8"?>
    <WebsiteConfiguration>
    <IndexDocument>
        <Suffix>index.html</Suffix>
    </IndexDocument>
    <ErrorDocument>
        <Key>error.html</Key>
    </ErrorDocument>
    <RoutingRules>
     <RoutingRule>
            <RuleNumber>1</RuleNumber>
            <Condition>
                <KeyPrefixEquals>abc/</KeyPrefixEquals>
                <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
            </Condition>
            <Redirect>
                <RedirectType>Mirror</RedirectType>
                <PassQueryString>true</PassQueryString>
                <MirrorURL>http://www.test.com/</MirrorURL>
                <MirrorPassQueryString>true</MirrorPassQueryString>
                <MirrorFollowRedirect>true</MirrorFollowRedirect>
                <MirrorCheckMd5>false</MirrorCheckMd5>
            </Redirect>
        </RoutingRule>
        </RoutingRules>
    </WebsiteConfiguration>`

	websiteFileName := randLowStr(12)
	s.createFile(websiteFileName, websiteXml, c)

	// website command test
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

	websiteArgs := []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err := cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	// get website
	websiteDownName := websiteFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	websiteArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	os.Remove(websiteFileName)
	os.Remove(websiteDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWebsiteDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	websiteXml := `<?xml version="1.0" encoding="UTF-8"?>
    <WebsiteConfiguration>
    <IndexDocument>
        <Suffix>index.html</Suffix>
    </IndexDocument>
    <ErrorDocument>
        <Key>error.html</Key>
    </ErrorDocument>
    <RoutingRules>
     <RoutingRule>
            <RuleNumber>1</RuleNumber>
            <Condition>
                <KeyPrefixEquals>abc/</KeyPrefixEquals>
                <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
            </Condition>
            <Redirect>
                <RedirectType>Mirror</RedirectType>
                <PassQueryString>true</PassQueryString>
                <MirrorURL>http://www.test.com/</MirrorURL>
                <MirrorPassQueryString>true</MirrorPassQueryString>
                <MirrorFollowRedirect>true</MirrorFollowRedirect>
                <MirrorCheckMd5>false</MirrorCheckMd5>
            </Redirect>
        </RoutingRule>
        </RoutingRules>
    </WebsiteConfiguration>`

	websiteFileName := randLowStr(12)
	s.createFile(websiteFileName, websiteXml, c)

	// website command test
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

	websiteArgs := []string{CloudURLToString(bucketName, ""), websiteFileName}
	_, err := cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// get website
	websiteDownName := websiteFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	// check websiteDownName
	_, err = os.Stat(websiteDownName)
	c.Assert(err, IsNil)
	os.Remove(websiteDownName)

	// delete websiteDownName
	strMethod = "delete"
	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	websiteArgs = []string{CloudURLToString(bucketName, ""), websiteDownName}
	_, err = cm.RunCommand("website", websiteArgs, options)
	c.Assert(err, NotNil)

	os.Remove(websiteFileName)
	os.Remove(websiteDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWebsiteHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"website"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
