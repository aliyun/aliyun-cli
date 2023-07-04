package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

// structs for replication get test
type GetResult struct {
	Rules []Rule `xml:"Rule"`
}

type Rule struct {
	Action                      string          `xml:"Action,omitempty"`                      // The replication action (ALL or PUT)
	ID                          string          `xml:"ID,omitempty"`                          // The rule ID
	Destination                 DestinationType `xml:"Destination,omitempty"`                 // Container for storing target bucket information
	HistoricalObjectReplication string          `xml:"HistoricalObjectReplication,omitempty"` // Whether to copy copy historical data (enabled or not)
	Status                      string          `xml:"Status,omitempty"`                      // The replication status (starting, doing or closing)
}

type DestinationType struct {
	Bucket   string `xml:"Bucket,omitempty"`
	Location string `xml:"Location,omitempty"`
}

func (s *OssutilCommandSuite) TestBucketReplicationPutAndGetSuccess(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	destinationRegion := "beijing"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	destinationEndpoint := "oss-cn-" + destinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create destination bucket
	destinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(destinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	putXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + destinationBucketName + `</Bucket>
		  <Location>oss-cn-` + destinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	putFileName := "test-putfile-" + randLowStr(5)
	s.createFile(putFileName, putXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs := []string{CloudURLToString(sourceBucketName, ""), putFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(5 * time.Second)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	var result GetResult
	err = xml.Unmarshal(content, &result)
	c.Assert(err, IsNil)

	c.Assert(result.Rules[0].Status, Equals, "starting")
	c.Assert(result.Rules[0].Destination.Location, Equals, "oss-cn-"+destinationRegion)
	c.Assert(result.Rules[0].Destination.Bucket, Equals, destinationBucketName)
	c.Assert(result.Rules[0].HistoricalObjectReplication, Equals, "enabled")

	os.Remove(putFileName)
	os.Remove(getFileName)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm destinationBucket
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(destinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}

// TestBucketReplicationDeleteSuccess
func (s *OssutilCommandSuite) TestBucketReplicationDeleteSuccess(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	destinationRegion := "beijing"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	destinationEndpoint := "oss-cn-" + destinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create destination bucket
	destinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(destinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	putXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + destinationBucketName + `</Bucket>
		  <Location>oss-cn-` + destinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	putFileName := "test-putfile-" + randLowStr(5)
	s.createFile(putFileName, putXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs := []string{CloudURLToString(sourceBucketName, ""), putFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(20 * time.Second)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	var result GetResult
	err = xml.Unmarshal(content, &result)
	c.Assert(err, IsNil)

	c.Assert(result.Rules[0].Status, NotNil) //maybe starting or doing
	c.Assert(result.Rules[0].Destination.Location, Equals, "oss-cn-"+destinationRegion)
	c.Assert(result.Rules[0].Destination.Bucket, Equals, destinationBucketName)
	c.Assert(result.Rules[0].HistoricalObjectReplication, Equals, "enabled")

	os.Remove(putFileName)
	os.Remove(getFileName)

	ruleID := result.Rules[0].ID

	strMethod = "delete"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	deleteArgs := []string{CloudURLToString(sourceBucketName, ""), ruleID}
	_, err = cm.RunCommand("replication", deleteArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(20 * time.Second)

	// output to file
	getFileNameAgain := "test-getfile-again-" + randLowStr(5)
	secondTestResultFile, _ := os.OpenFile(getFileNameAgain, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout = os.Stdout
	os.Stdout = secondTestResultFile

	// get again
	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs = []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)

	secondTestResultFile.Close()
	os.Stdout = oldStdout

	if err != nil {
		c.Assert(strings.Contains(err.Error(), "StatusCode=404"), Equals, true)
	} else {
		content, err = ioutil.ReadFile(getFileNameAgain)
		c.Assert(err, IsNil)

		var afterDeleteresult GetResult
		err = xml.Unmarshal(content, &afterDeleteresult)
		c.Assert(err, IsNil)

		c.Assert(afterDeleteresult.Rules[0].Status, Equals, "closing")
	}

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm destinationBucket
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(destinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}

func (s *OssutilCommandSuite) TestBucketReplicationDeleteWithEmptyRuleID(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	destinationRegion := "beijing"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	destinationEndpoint := "oss-cn-" + destinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create destination bucket
	destinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(destinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	putXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + destinationBucketName + `</Bucket>
		  <Location>oss-cn-` + destinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	putFileName := "test-putfile-" + randLowStr(5)
	s.createFile(putFileName, putXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs := []string{CloudURLToString(sourceBucketName, ""), putFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(5 * time.Second)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	var result GetResult
	err = xml.Unmarshal(content, &result)
	c.Assert(err, IsNil)

	c.Assert(result.Rules[0].Status, Equals, "starting")
	c.Assert(result.Rules[0].Destination.Location, Equals, "oss-cn-"+destinationRegion)
	c.Assert(result.Rules[0].Destination.Bucket, Equals, destinationBucketName)
	c.Assert(result.Rules[0].HistoricalObjectReplication, Equals, "enabled")

	os.Remove(putFileName)
	os.Remove(getFileName)

	strMethod = "delete"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// ruleID is empty
	deleteArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", deleteArgs, options)
	c.Assert(err, NotNil)
	// c.Assert(err, IsNil)

	// output to file
	getFileNameAgain := "test-getfile-again-" + randLowStr(5)
	secondTestResultFile, _ := os.OpenFile(getFileNameAgain, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout = os.Stdout
	os.Stdout = secondTestResultFile

	// get again
	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs = []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	secondTestResultFile.Close()
	os.Stdout = oldStdout

	content, err = ioutil.ReadFile(getFileNameAgain)
	c.Assert(err, IsNil)

	var afterDeleteresult GetResult
	err = xml.Unmarshal(content, &afterDeleteresult)
	c.Assert(err, IsNil)

	c.Assert(afterDeleteresult.Rules[0].Status, Equals, "starting")

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm destinationBucket
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(destinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}

func (s *OssutilCommandSuite) TestBucketReplicationGetLocation(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod := "get"
	item := "location"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &item,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	strContent := string(content)

	c.Assert(strings.Contains(strContent, "<ReplicationLocation>"), Equals, true)

	os.Remove(getFileName)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestBucketReplicationGetProgressWithRuleID(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	firstDestinationRegion := "beijing"
	secondDestinationRegion := "shenzhen"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	firstDestinationEndpoint := "oss-cn-" + firstDestinationRegion + ".aliyuncs.com"
	secondDestinationEndpoint := "oss-cn-" + secondDestinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create first destination bucket
	firstDestinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(firstDestinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &firstDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create second destination bucket
	secondDestinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(secondDestinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &secondDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	firstPutXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + firstDestinationBucketName + `</Bucket>
		  <Location>oss-cn-` + firstDestinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	firstPutFileName := "test-putfile-" + randLowStr(5)
	s.createFile(firstPutFileName, firstPutXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs := []string{CloudURLToString(sourceBucketName, ""), firstPutFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	secondPutXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + secondDestinationBucketName + `</Bucket>
		  <Location>oss-cn-` + secondDestinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	secondPutFileName := "test-putfile-" + randLowStr(5)
	s.createFile(secondPutFileName, secondPutXml, c)

	// replication command and put method test
	strMethod = "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs = []string{CloudURLToString(sourceBucketName, ""), secondPutFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(5 * time.Second)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	fmt.Println(string(content))

	var result GetResult
	err = xml.Unmarshal(content, &result)
	c.Assert(err, IsNil)

	var index int
	for i := 0; i <= 1; i++ {
		if result.Rules[i].Destination.Location == "oss-cn-"+secondDestinationRegion {
			index = i
			break
		}
	}

	ruleID := result.Rules[index].ID

	// output to file
	getProgressFileName := "test-getfile-" + randLowStr(5)
	testProgressResultFile, _ := os.OpenFile(getProgressFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout = os.Stdout
	os.Stdout = testProgressResultFile

	// replication command and get method test
	strMethod = "get"
	item := "progress"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &item,
	}

	getArgs = []string{CloudURLToString(sourceBucketName, ""), ruleID}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testProgressResultFile.Close()
	os.Stdout = oldStdout

	content, err = ioutil.ReadFile(getProgressFileName)
	c.Assert(err, IsNil)

	var progressResult GetResult
	err = xml.Unmarshal(content, &progressResult)
	c.Assert(err, IsNil)

	c.Assert(progressResult.Rules[0].ID, Equals, ruleID)
	c.Assert(progressResult.Rules[0].Status, Equals, "starting")
	c.Assert(progressResult.Rules[0].Destination.Location, Equals, "oss-cn-"+secondDestinationRegion)
	c.Assert(progressResult.Rules[0].Destination.Bucket, Equals, secondDestinationBucketName)
	c.Assert(progressResult.Rules[0].HistoricalObjectReplication, Equals, "enabled")

	// replication command and put rtc
	rtcPutXml := `<?xml version="1.0" encoding="UTF-8"?>
<ReplicationRule>
    <RTC>
        <Status>disabled</Status>
    </RTC>
    <ID>` + ruleID + `</ID>
</ReplicationRule>`

	rtcPutFileName := "test-putRtc-" + randLowStr(5)
	s.createFile(rtcPutFileName, rtcPutXml, c)
	strMethod = "put"
	item = "rtc"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &item,
	}

	getArgs = []string{CloudURLToString(sourceBucketName, ""), rtcPutFileName}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	os.Remove(firstPutFileName)
	os.Remove(secondPutFileName)
	os.Remove(getFileName)
	os.Remove(getProgressFileName)
	os.Remove(rtcPutFileName)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm first destinationBucket
	options = OptionMapType{
		"endpoint":        &firstDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(firstDestinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm second destinationBucket
	options = OptionMapType{
		"endpoint":        &secondDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(secondDestinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}

func (s *OssutilCommandSuite) TestBucketReplicationGetProgressWithEmptyRuleID(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	firstDestinationRegion := "beijing"
	secondDestinationRegion := "shenzhen"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	firstDestinationEndpoint := "oss-cn-" + firstDestinationRegion + ".aliyuncs.com"
	secondDestinationEndpoint := "oss-cn-" + secondDestinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create first destination bucket
	firstDestinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(firstDestinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &firstDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create second destination bucket
	secondDestinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(secondDestinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &secondDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	firstPutXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + firstDestinationBucketName + `</Bucket>
		  <Location>oss-cn-` + firstDestinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	firstPutFileName := "test-putfile-" + randLowStr(5)
	s.createFile(firstPutFileName, firstPutXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs := []string{CloudURLToString(sourceBucketName, ""), firstPutFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	secondPutXml := `<?xml version="1.0" encoding="UTF-8"?>
	<ReplicationConfiguration>
	  <Rule>
		<Action>PUT</Action>
		<Destination>
		  <Bucket>` + secondDestinationBucketName + `</Bucket>
		  <Location>oss-cn-` + secondDestinationRegion + `</Location>
		</Destination>
		<HistoricalObjectReplication>enabled</HistoricalObjectReplication>
	  </Rule>
	</ReplicationConfiguration>`

	secondPutFileName := "test-putfile-" + randLowStr(5)
	s.createFile(secondPutFileName, secondPutXml, c)

	// replication command and put method test
	strMethod = "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	putArgs = []string{CloudURLToString(sourceBucketName, ""), secondPutFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(5 * time.Second)

	// output to file
	getFileName := "test-getfile-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(getFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	// replication command and get method test
	strMethod = "get"
	item := "progress"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &item,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, IsNil)

	testResultFile.Close()
	os.Stdout = oldStdout

	content, err := ioutil.ReadFile(getFileName)
	c.Assert(err, IsNil)

	var result GetResult
	err = xml.Unmarshal(content, &result)
	c.Assert(err, IsNil)

	var firstIndex int
	for i := 0; i <= 1; i++ {
		if result.Rules[i].Destination.Location == ("oss-cn-" + firstDestinationRegion) {
			firstIndex = i
			break
		}
	}
	secondIndex := 1 - firstIndex

	fmt.Println(result.Rules)

	c.Assert(result.Rules[firstIndex].Status, Equals, "starting")
	c.Assert(result.Rules[firstIndex].Destination.Location, Equals, "oss-cn-"+firstDestinationRegion)
	c.Assert(result.Rules[firstIndex].Destination.Bucket, Equals, firstDestinationBucketName)
	c.Assert(result.Rules[firstIndex].HistoricalObjectReplication, Equals, "enabled")

	c.Assert(result.Rules[secondIndex].Status, Equals, "starting")
	c.Assert(result.Rules[secondIndex].Destination.Location, Equals, "oss-cn-"+secondDestinationRegion)
	c.Assert(result.Rules[secondIndex].Destination.Bucket, Equals, secondDestinationBucketName)
	c.Assert(result.Rules[secondIndex].HistoricalObjectReplication, Equals, "enabled")

	os.Remove(firstPutFileName)
	os.Remove(secondPutFileName)
	os.Remove(getFileName)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm first destinationBucket
	options = OptionMapType{
		"endpoint":        &firstDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(firstDestinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm second destinationBucket
	options = OptionMapType{
		"endpoint":        &secondDestinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(secondDestinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}

func (s *OssutilCommandSuite) TestBucketReplicationWithFalseMethod(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// replication command and false method test
	strMethod := randLowStr(5)
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	Args := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", Args, options)
	c.Assert(err, NotNil)

	// replication command and empty method test
	strMethod = ""
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	Args = []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", Args, options)
	c.Assert(err, NotNil)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestBucketReplicationGetWithFalseItem(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// replication command and get method test
	strMethod := "get"
	item := randLowStr(5)
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &item,
	}

	getArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", getArgs, options)
	c.Assert(err, NotNil)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestBucketReplicationPutError(c *C) {
	str := ""
	sourceRegion := "hangzhou"
	destinationRegion := "beijing"
	sourceEndpoint := "oss-cn-" + sourceRegion + ".aliyuncs.com"
	destinationEndpoint := "oss-cn-" + destinationRegion + ".aliyuncs.com"

	// create source bucket
	sourceBucketName := bucketNamePrefix + randLowStr(12)
	args := []string{CloudURLToString(sourceBucketName, "")}
	options := OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	// create destination bucket
	destinationBucketName := bucketNamePrefix + randLowStr(12)
	args = []string{CloudURLToString(destinationBucketName, "")}
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand("mb", args, options)
	c.Assert(err, IsNil)

	putXml := ``

	putFileName := "test-putfile-" + randLowStr(5)
	s.createFile(putFileName, putXml, c)

	// replication command and put method test
	strMethod := "put"
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// input xml file is empty
	putArgs := []string{CloudURLToString(sourceBucketName, ""), putFileName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, NotNil)

	// input xml file is dir
	dirName := "test-dir-" + randLowStr(5)
	os.Mkdir(dirName, 0664)
	putArgs = []string{CloudURLToString(sourceBucketName, ""), dirName}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, NotNil)

	//local xml file is not exist
	putArgs = []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("replication", putArgs, options)
	c.Assert(err, NotNil)

	os.Remove(dirName)
	os.Remove(putFileName)

	// use rm command to rm sourceBucket
	ok := true
	options = OptionMapType{
		"endpoint":        &sourceEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs := []string{CloudURLToString(sourceBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

	// use rm command to rm destinationBucket
	options = OptionMapType{
		"endpoint":        &destinationEndpoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}

	rmArgs = []string{CloudURLToString(destinationBucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, options)
	c.Assert(err, IsNil)

}
