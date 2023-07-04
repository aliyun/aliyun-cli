package lib

import (
	"os"
	"time"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketStyleHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"style"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}

func (s *OssutilCommandSuite) TestBucketStyleError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	styleFileName := "style" + randLowStr(5) + ".xml"
	styleName := "style" + randLowStr(12)
	// style command test
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
	styleArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod = "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// args is empty
	strMethod = "put"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	//params just two
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	styleArgs = []string{CloudURLToString(bucketName, ""), styleFileName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// local file is empty
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(styleFileName)
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// local file is dir
	err = os.MkdirAll(styleFileName, 0755)
	c.Assert(err, IsNil)
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(styleFileName)

	//local file is empty
	s.createFile(styleFileName, "", c)
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(styleFileName)

	//local file is not xml file
	s.createFile(styleFileName, "aaa", c)
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(styleFileName)

	// StorageURLFromString error
	styleArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	styleArgs = []string{"oss:///"}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	styleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	styleArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)
	os.Remove(styleFileName)

	// put style success
	s.createFile(styleFileName, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Style> <Content>image/resize,p_50</Content></Style>", c)
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	// test style get
	// method is empty
	strMethod = ""
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// method error
	strMethod = "gett"
	options["method"] = &strMethod
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// miss style name
	strMethod = "get"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// test style delete
	// method is empty
	strMethod = ""
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// method error
	strMethod = "deletedeed"
	options["method"] = &strMethod
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	// miss style name
	strMethod = "delete"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, NotNil)

	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	os.Remove(styleFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketStyleSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	styleFileName := "style" + randLowStr(5) + ".xml"
	styleName := "style" + randLowStr(12)

	styleDownName := "style-down" + randLowStr(5) + ".xml"
	// style command test
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

	// put style success
	s.createFile(styleFileName, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<Style> <Content>image/resize,p_50</Content></Style>", c)

	styleArgs := []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err := cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	// get bucket style with output
	strMethod = "get"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	// get bucket style with file
	strMethod = "get"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleDownName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)
	_, err = os.Stat(styleDownName)
	c.Assert(err, IsNil)
	os.Remove(styleDownName)
	time.Sleep(time.Second * 1)

	// list bucket style with output
	strMethod = "list"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	// list bucket style with file
	strMethod = "list"
	options["method"] = &strMethod
	styleArgs = []string{CloudURLToString(bucketName, ""), styleDownName}
	_, err = cm.RunCommand("style", styleArgs, options)
	c.Assert(err, IsNil)

	_, err = os.Stat(styleDownName)
	c.Assert(err, IsNil)
	os.Remove(styleDownName)

	os.Remove(styleFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetBucketStyleConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	styleXml := `<?xml version="1.0" encoding="UTF-8"?>
<Style>
 <Content>image/resize,p_50</Content>
</Style>`

	styleName := "style-" + randLowStr(5)
	styleFileName := inputFileName + randLowStr(5)
	s.createFile(styleFileName, styleXml, c)

	// style command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	command := "style"

	styleArgs := []string{CloudURLToString(bucketName, ""), styleName, styleFileName}
	_, err := cm.RunCommand(command, styleArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(time.Second * 5)
	// get resource group
	styleDownName := styleFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleDownName}
	_, err = cm.RunCommand(command, styleArgs, options)
	c.Assert(err, IsNil)

	styleArgs = []string{CloudURLToString(bucketName, ""), styleName, styleDownName}
	_, err = cm.RunCommand(command, styleArgs, options)
	c.Assert(err, IsNil)

	styleArgs = []string{CloudURLToString(bucketName, ""), styleName}
	_, err = cm.RunCommand(command, styleArgs, options)
	c.Assert(err, IsNil)

	os.Remove(styleFileName)
	os.Remove(styleDownName)
	s.removeBucket(bucketName, true, c)
}
