package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	TestingT(t)
}

type OssutilCommandSuite struct {
	startT time.Time
}

var _ = Suite(&OssutilCommandSuite{})

var (
	// Update before running test
	endpoint             = ""
	accessKeyID          = ""
	accessKeySecret      = ""
	proxyHost            = ""
	proxyUser            = ""
	proxyPwd             = ""
	accountID            = ""
	stsAccessID          = ""
	stsAccessKeySecret   = ""
	stsARN               = ""
	ecsRoleName          = ""
	stsRegion            = ""
	payerAccessKeyID     = ""
	payerAccessKeySecret = ""
	payerAccountID       = ""
	region               = ""
)

var (
	logPath           = "ossutil_test_" + time.Now().Format("20060102_150405") + ".log"
	ConfigFile        = "ossutil_test.boto" + randStr(5)
	configFile        = ConfigFile
	testLogFile, _    = os.OpenFile(logPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	testLogger        = log.New(testLogFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	resultPath        = "ossutil_test.result" + randStr(5)
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	uploadFileName    = "ossutil_test.upload_file" + randStr(5)
	downloadFileName  = "ossutil_test.download_file" + randStr(5)
	downloadDir       = "ossutil_test.download_dir" + randStr(5)
	inputFileName     = "ossutil_test.input_file" + randStr(5)
	objectFileName    = "ossutil_test.object_file" + randStr(5)
	content           = "abc"
	cm                = CommandManager{}
	out               = os.Stdout
	errout            = os.Stderr
	sleepTime         = time.Second
)

var (
	commonNamePrefix   = "ossutil-test-"
	bucketNamePrefix   = commonNamePrefix + randLowStr(6)
	bucketNameExist    = "special-" + bucketNamePrefix + "existbucket"
	bucketNameDest     = "special-" + bucketNamePrefix + "destbucket"
	bucketNameNotExist = "nodelete-ossutil-test-notexist"
)

var (
	payerBucket         = bucketNamePrefix + "-payer"
	payerConfigFile     = ConfigFile + "-payer"
	payerBucketEndPoint = ""
)

// Run once when the suite starts running
func (s *OssutilCommandSuite) SetUpSuite(c *C) {
	fmt.Printf("set up OssutilCommandSuite\n")
	os.Stdout = testLogFile
	os.Stderr = testLogFile
	processTickInterval = 1
	testLogger.Println("test command started")
	SetUpCredential()
	cm.Init()
	s.configNonInteractive(c)
	s.createFile(uploadFileName, content, c)
	s.SetUpBucketEnv(c)
	s.SetUpPayerEnv(c)
}

func SetUpCredential() {
	if endpoint == "" {
		endpoint = os.Getenv("OSS_TEST_ENDPOINT")
	}
	if strings.HasPrefix(endpoint, "https://") {
		endpoint = endpoint[8:]
	}
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = endpoint[7:]
	}
	if accessKeyID == "" {
		accessKeyID = os.Getenv("OSS_TEST_ACCESS_KEY_ID")
	}
	if accessKeySecret == "" {
		accessKeySecret = os.Getenv("OSS_TEST_ACCESS_KEY_SECRET")
	}
	if ue := os.Getenv("OSS_TEST_UPDATE_ENDPOINT"); ue != "" {
		vUpdateEndpoint = ue
	}
	if ub := os.Getenv("OSS_TEST_UPDATE_BUCKET"); ub != "" {
		vUpdateBucket = ub
	}
	if strings.HasPrefix(vUpdateEndpoint, "https://") {
		vUpdateEndpoint = vUpdateEndpoint[8:]
	}
	if strings.HasPrefix(vUpdateEndpoint, "http://") {
		vUpdateEndpoint = vUpdateEndpoint[7:]
	}
	if proxyHost == "" {
		proxyHost = os.Getenv("OSS_TEST_PROXY_HOST")
	}
	if proxyUser == "" {
		proxyUser = os.Getenv("OSS_TEST_PROXY_USER")
	}
	if proxyPwd == "" {
		proxyPwd = os.Getenv("OSS_TEST_PROXY_PASSWORD")
	}
	if accountID == "" {
		accountID = os.Getenv("OSS_TEST_ACCOUNT_ID")
	}
	if stsAccessID == "" {
		stsAccessID = os.Getenv("OSS_TEST_STS_ID")
	}
	if stsAccessKeySecret == "" {
		stsAccessKeySecret = os.Getenv("OSS_TEST_STS_KEY")
	}
	if stsARN == "" {
		stsARN = os.Getenv("OSS_TEST_STS_ARN")
	}
	if ecsRoleName == "" {
		ecsRoleName = os.Getenv("OSS_TEST_ECS_ROLE_NAME")
	}
	if stsRegion == "" {
		stsRegion = os.Getenv("OSS_TEST_STS_REGION")
	}
	if payerAccessKeyID == "" {
		payerAccessKeyID = os.Getenv("OSS_TEST_PAYER_ACCESS_KEY_ID")
	}
	if payerAccessKeySecret == "" {
		payerAccessKeySecret = os.Getenv("OSS_TEST_PAYER_ACCESS_KEY_SECRET")
	}
	if payerAccountID == "" {
		payerAccountID = os.Getenv("OSS_TEST_PAYER_UID")
	}
	if region == "" {
		region = os.Getenv("OSS_TEST_REGION")
	}
}

func (s *OssutilCommandSuite) SetUpBucketEnv(c *C) {
	s.removeBuckets(commonNamePrefix, c)
	s.putBucket(bucketNameExist, c)
	s.putBucket(bucketNameDest, c)
}

func (s *OssutilCommandSuite) SetUpPayerEnv(c *C) {
	s.putBucket(payerBucket, c)
	payerBucketEndPoint = endpoint
	policy := `
	{
		"Version":"1",
		"Statement":[
			{
				"Action":[
					"oss:*"
				],
				"Effect":"Allow",
				"Principal":["` + payerAccountID + `"],
				"Resource":["acs:oss:*:*:` + payerBucket + `", "acs:oss:*:*:` + payerBucket + `/*"]
			}
		]
	}`
	s.putBucketPolicy(payerBucket, policy, c)

	//set payerConfigfile
	data := fmt.Sprintf("[Credentials]\nlanguage=EN\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", payerBucketEndPoint, payerAccessKeyID, payerAccessKeySecret)
	s.createFile(payerConfigFile, data, c)
}

// Run before each test or benchmark starts running
func (s *OssutilCommandSuite) TearDownSuite(c *C) {
	fmt.Printf("tear down OssutilCommandSuite\n")
	s.removeBuckets(commonNamePrefix, c)
	s.removeBucket(bucketNameExist, true, c)
	s.removeBucket(bucketNameDest, true, c)
	testLogger.Println("test command completed")
	os.Remove(configFile)
	os.Remove(configFile + ".bak")
	os.Remove(resultPath)
	os.Remove(uploadFileName)
	os.Remove(downloadFileName)
	os.RemoveAll(downloadDir)
	os.RemoveAll(DefaultOutputDir)
	os.Remove(payerConfigFile)
	os.Stdout = out
	os.Stderr = errout
}

// Run after each test or benchmark runs
func (s *OssutilCommandSuite) SetUpTest(c *C) {
	fmt.Printf("set up test:%s\n", c.TestName())
	s.startT = time.Now()
	configFile = ConfigFile
}

// Run once after all tests or benchmarks have finished running
func (s *OssutilCommandSuite) TearDownTest(c *C) {
	endT := time.Now()
	cost := endT.UnixNano()/1000/1000 - s.startT.UnixNano()/1000/1000
	fmt.Printf("tear down test:%s,cost:%d(ms)\n", c.TestName(), cost)
}

func randLowStr(n int) string {
	return strings.ToLower(randStr(n))
}

func getFileList(dpath string) ([]string, error) {
	fileNames := make([]string, 0)
	err := filepath.Walk(dpath, func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		// fpath may be dir,exclude itself
		if fpath != dpath {
			fileNames = append(fileNames, fpath)
		}

		return nil
	})
	return fileNames, err
}

func (s *OssutilCommandSuite) PutObject(bucketName string, object string, body string, c *C) {
	// create client and bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.PutObject(object, strings.NewReader(body))

	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) AppendObject(bucketName string, object string, body string, position int64, c *C) {
	// create client and bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	_, err = bucket.AppendObject(object, strings.NewReader(body), position)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) configNonInteractive(c *C) {
	command := "config"
	var args []string
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 4)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
	c.Assert(opts[OptionEndpoint], Equals, endpoint)
	c.Assert(opts[OptionAccessKeyID], Equals, accessKeyID)
	c.Assert(opts[OptionAccessKeySecret], Equals, accessKeySecret)
}

func (s *OssutilCommandSuite) createFile(fileName, content string, c *C) {
	fout, err := os.Create(fileName)
	defer fout.Close()
	c.Assert(err, IsNil)
	_, err = fout.WriteString(content)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) readFile(fileName string, c *C) string {
	f, err := ioutil.ReadFile(fileName)
	c.Assert(err, IsNil)
	return string(f)
}

func (s *OssutilCommandSuite) removeBuckets(prefix string, c *C) {
	buckets := s.listBuckets(false, c)
	for _, bucket := range buckets {
		if strings.Contains(bucket, prefix) {
			s.removeBucket(bucket, true, c)
		}
	}
}

type OptionPair struct {
	Key   string
	Value string
}

func (s *OssutilCommandSuite) rawList(args []string, cmdline string, optionPairs ...OptionPair) (bool, error) {
	array := strings.Split(cmdline, " ")
	if len(array) < 2 {
		return false, fmt.Errorf("ls test wrong cmdline given")
	}

	parameter := strings.Split(array[1], "-")
	if len(parameter) < 2 {
		return false, fmt.Errorf("ls test wrong cmdline given")
	}

	command := array[0]
	sf := strings.Contains(parameter[1], "s")
	d := strings.Contains(parameter[1], "d")
	m := strings.Contains(parameter[1], "m")
	a := strings.Contains(parameter[1], "a")

	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"shortFormat":     &sf,
		"directory":       &d,
		"multipart":       &m,
		"allType":         &a,
		"limitedNum":      &limitedNum,
	}

	for k, _ := range optionPairs {
		options[optionPairs[k].Key] = &optionPairs[k].Value
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) listLimitedMarker(bucket, prefix, cmdline string, limitedNum int64, marker, uploadIDMarker string, c *C) []string {
	args := []string{CloudURLToString(bucket, prefix)}
	out := os.Stdout
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testResultFile
	showElapse, err := s.rawListLimitedMarker(args, cmdline, limitedNum, marker, uploadIDMarker)
	os.Stdout = out
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// get result
	results := []string{}
	if bucket == "" && prefix == "" {
		results = s.getBucketResults(c)
	} else {
		results = s.getObjectResults(c)
	}
	os.Remove(resultPath)
	return results
}

func (s *OssutilCommandSuite) rawListLimitedMarker(args []string, cmdline string, limitedNum int64, marker, uploadIDMarker string) (bool, error) {
	array := strings.SplitN(cmdline, " ", 2)
	if len(array) < 2 {
		return false, fmt.Errorf("ls test wrong cmdline given")
	}

	command := array[0]

	encodingType := ""
	if pos := strings.Index(array[1], "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		array[1] = array[1][0:pos] + array[1][pos+len("--encoding-type url"):]
	}

	parameter := strings.Split(array[1], "-")
	sf := false
	d := false
	m := false
	a := false
	if len(parameter) >= 2 {
		sf = strings.Contains(parameter[1], "s")
		d = strings.Contains(parameter[1], "d")
		m = strings.Contains(parameter[1], "m")
		a = strings.Contains(parameter[1], "a")
	}

	str := ""
	limitedNumStr := strconv.FormatInt(limitedNum, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"shortFormat":     &sf,
		"directory":       &d,
		"multipart":       &m,
		"allType":         &a,
		"limitedNum":      &limitedNumStr,
		"marker":          &marker,
		"uploadIDMarker":  &uploadIDMarker,
		"encodingType":    &encodingType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) listBuckets(shortFormat bool, c *C) []string {
	var args []string
	out := os.Stdout
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testResultFile
	showElapse, err := s.rawList(args, "ls -a")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	os.Stdout = out

	// get result
	buckets := s.getBucketResults(c)
	os.Remove(resultPath)
	return buckets
}

func (s *OssutilCommandSuite) getBucketResults(c *C) []string {
	result := s.getResult(c)
	c.Assert(len(result) >= 1, Equals, true)
	buckets := []string{}
	shortEndpoint := strings.TrimRight(endpoint, ".aliyuncs.com")
	shortEndpoint = strings.TrimRight(shortEndpoint, "-internal")
	for _, str := range result {
		pos := strings.Index(str, SchemePrefix)
		if pos != -1 && strings.Contains(str, shortEndpoint) {
			buckets = append(buckets, str[pos+len(SchemePrefix):])
		}
	}
	return buckets
}

func (s *OssutilCommandSuite) getResult(c *C) []string {
	return s.getFileResult(resultPath, c)
}

func (s *OssutilCommandSuite) getFileResult(fileName string, c *C) []string {
	str := s.readFile(fileName, c)
	sli := strings.Split(str, "\n")
	result := []string{}
	for _, str := range sli {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func (s *OssutilCommandSuite) getReportResult(fileName string, c *C) []string {
	result := s.getFileResult(fileName, c)
	c.Assert(len(result) >= 1, Equals, true)
	c.Assert(strings.HasPrefix(result[0], "#"), Equals, true)
	result = result[1:]
	for _, r := range result {
		c.Assert(strings.HasPrefix(r, "[Error]"), Equals, true)
	}
	return result
}

func (s *OssutilCommandSuite) removeBucket(bucket string, clearObjects bool, c *C) {
	args := []string{CloudURLToString(bucket, "")}
	var showElapse bool
	var err error
	if !clearObjects {
		showElapse, err = s.rawRemove(args, false, true, true)
	} else {
		s.removeBucketObjectVersions(bucket, c)
		showElapse, err = s.removeWrapper("rm -arfb", bucket, "", c)
	}
	if err != nil {
		bNoBucket := strings.Contains(err.Error(), "NoSuchBucket")
		bBucketEmpty := strings.Contains(err.Error(), "BucketNotEmpty")
		c.Assert((bBucketEmpty || bNoBucket), Equals, true)
	} else {
		c.Assert(showElapse, Equals, true)
	}
}

func (s *OssutilCommandSuite) rawRemove(args []string, recursive, force, bucket bool) (bool, error) {
	command := "rm"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &recursive,
		"force":           &force,
		"bucket":          &bucket,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) removeBucketObjectVersions(bucket string, c *C) (bool, error) {
	allVersions := true
	recursive := true
	force := true

	args := []string{CloudURLToString(bucket, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &recursive,
		"force":           &force,
		"allVersions":     &allVersions,
	}
	showElapse, err := cm.RunCommand("rm", args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) removeWrapper(cmdline string, bucket string, object string, c *C) (bool, error) {
	array := strings.SplitN(cmdline, " ", 2)
	if len(array) < 2 {
		return false, fmt.Errorf("rm test wrong cmdline given")
	}

	encodingType := ""
	if pos := strings.Index(array[1], "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		array[1] = array[1][0:pos] + array[1][pos+len("--encoding-type url"):]
	}

	parameter := strings.Split(array[1], "-")
	if len(parameter) < 2 {
		return false, fmt.Errorf("rm test wrong cmdline given")
	}

	command := array[0]
	a := strings.Contains(parameter[1], "a")
	m := strings.Contains(parameter[1], "m")
	b := strings.Contains(parameter[1], "b")
	r := strings.Contains(parameter[1], "r")
	f := strings.Contains(parameter[1], "f")

	args := []string{CloudURLToString(bucket, object)}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &b,
		"allType":         &a,
		"multipart":       &m,
		"recursive":       &r,
		"force":           &f,
		"encodingType":    &encodingType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) initRemove(bucket string, object string, cmdline string) error {
	array := strings.Split(cmdline, " ")
	if len(array) < 2 {
		return fmt.Errorf("rm test wrong cmdline given")
	}

	parameter := strings.Split(array[1], "-")
	if len(parameter) < 2 {
		return fmt.Errorf("rm test wrong cmdline given")
	}

	a := strings.Contains(parameter[1], "a")
	m := strings.Contains(parameter[1], "m")
	b := strings.Contains(parameter[1], "b")
	r := strings.Contains(parameter[1], "r")
	f := strings.Contains(parameter[1], "f")

	args := []string{CloudURLToString(bucket, object)}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &b,
		"allType":         &a,
		"multipart":       &m,
		"recursive":       &r,
		"force":           &f,
	}
	err := removeCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) removeObjects(bucket, prefix string, recursive, force bool, c *C) {
	args := []string{CloudURLToString(bucket, prefix)}
	showElapse, err := s.rawRemove(args, recursive, force, false)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) clearObjects(bucket, prefix string, c *C) {
	showElapse, err := s.removeWrapper("rm -afr", bucket, prefix, c)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) listObjects(bucket, prefix string, cmdline string, c *C) []string {
	args := []string{CloudURLToString(bucket, prefix)}
	out := os.Stdout
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testResultFile
	showElapse, err := s.rawList(args, cmdline)
	os.Stdout = out
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// get result
	objects := s.getObjectResults(c)
	os.Remove(resultPath)
	return objects
}

func (s *OssutilCommandSuite) getObjectResults(c *C) []string {
	result := s.getResult(c)
	c.Assert(len(result) >= 1, Equals, true)
	objects := []string{}
	for _, str := range result {
		pos := strings.Index(str, SchemePrefix)
		if pos != -1 {
			url := str[pos:]
			cloudURL, err := CloudURLFromString(url, "")
			c.Assert(err, IsNil)
			c.Assert(cloudURL.object != "", Equals, true)
			objects = append(objects, cloudURL.object)
		}
	}
	return objects
}

func (s *OssutilCommandSuite) putBucketWithACL(bucket string, acl string) (bool, error) {
	args := []string{CloudURLToString(bucket, "")}
	showElapse, err := s.rawPutBucketWithACL(args, acl)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawPutBucketWithACL(args []string, acl string) (bool, error) {
	command := "mb"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"acl":             &acl,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawPutBucketWithACLLanguage(args []string, acl, language string) (bool, error) {
	command := "mb"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"acl":             &acl,
		"language":        &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) putBucket(bucket string, c *C) {
	command := "mb"
	args := []string{CloudURLToString(bucket, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) putBucketWithEndPoint(bucket string, strEndPoint string, c *C) {
	command := "mb"
	args := []string{CloudURLToString(bucket, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &strEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) putBucketVersioning(bucket string, status string, c *C) {
	command := "bucket-versioning"
	args := []string{CloudURLToString(bucket, ""), status}
	str := ""
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) putBucketWithStorageClass(bucket string, storageClass string, c *C) error {
	args := []string{CloudURLToString(bucket, "")}
	err := s.initPutBucketWithStorageClass(args, storageClass)
	c.Assert(err, IsNil)
	err = makeBucketCommand.RunCommand()
	return err
}

func (s *OssutilCommandSuite) putBucketPolicy(bucket string, policyJson string, c *C) {
	command := "bucket-policy"
	policyFileName := randLowStr(12)
	s.createFile(policyFileName, policyJson, c)
	args := []string{CloudURLToString(bucket, ""), policyFileName}
	str := ""
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	os.Remove(policyFileName)
}

func (s *OssutilCommandSuite) initPutBucketWithStorageClass(args []string, storageClass string) error {
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"storageClass":    &storageClass,
	}
	err := makeBucketCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) rawCP(srcURL, destURL string, recursive, force, update bool, threshold int64, cpDir string) (bool, error) {
	args := []string{srcURL, destURL}
	showElapse, err := s.rawCPWithArgs(args, recursive, force, update, threshold, cpDir)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawCPWithArgs(args []string, recursive, force, update bool, threshold int64, cpDir string) (bool, error) {
	command := "cp"
	str := ""
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) createTestFiles(dir, subdir string, c *C, contents map[string]string) []string {
	// Create dirs
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	filenames := make([]string, 0)

	// Create files
	num := 3
	for i := 1020; i < 1020+num; i++ {
		filename := fmt.Sprintf("testfile%d.txt", i)
		content := fmt.Sprintf("include测试文件：%d内容", i)
		filenames = append(filenames, filename)
		contents[filename] = content
		filename = dir + "/" + filename
		s.createFile(filename, content, c)
	}

	num = 2
	for i := 2; i < 2+num; i++ {
		filename := fmt.Sprintf("test%d.jpg", i)
		content := fmt.Sprintf("include test jpg\n%dcontent", i)
		filenames = append(filenames, filename)
		contents[filename] = content
		filename = dir + "/" + filename
		s.createFile(filename, content, c)
	}

	num = 2
	for i := 103; i < 103+num; i++ {
		filename := fmt.Sprintf("testfile%d.txt", i)
		filename = subdir + "/" + filename
		content := fmt.Sprintf("include测试文件：%d内容", i)
		filenames = append(filenames, filename)
		contents[filename] = content
		filename = dir + "/" + filename
		s.createFile(filename, content, c)
	}
	filename := subdir + "/" + "my.rtf"
	content := fmt.Sprintf("include测试文件：%s内容", "my.rtf")
	filenames = append(filenames, filename)
	contents[filename] = content
	filename = dir + "/" + filename
	s.createFile(filename, content, c)

	return filenames
}

func (s *OssutilCommandSuite) prepareTestFiles(dir, subdir, filePrefix, fileSuffix string, num int, c *C) []string {
	// Create dirs
	f, err := os.Stat(dir)
	if err != nil {
		err := os.MkdirAll(dir, 0755)
		c.Assert(err, IsNil)
	} else {
		c.Assert(f.IsDir(), Equals, true)
	}

	f, err = os.Stat(dir + string(os.PathSeparator) + subdir)
	if err != nil {
		err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
		c.Assert(err, IsNil)
	} else {
		c.Assert(f.IsDir(), Equals, true)
	}

	filenames := make([]string, 0)

	// Create files
	for i := 0; i < num; i++ {
		filename := fmt.Sprintf("%s-testfile%d.txt", filePrefix, i) + fileSuffix
		content := fmt.Sprintf("include测试文件：%d内容", i)
		filenames = append(filenames, filename)
		filename = dir + string(os.PathSeparator) + filename
		s.createFile(filename, content, c)
	}

	for i := num; i < 2*num; i++ {
		filename := fmt.Sprintf("%s-testfile%d.jpg", filePrefix, i) + fileSuffix
		content := fmt.Sprintf("include test jpg\n%dcontent", i)
		filenames = append(filenames, subdir+string(os.PathSeparator)+filename)
		filename = dir + string(os.PathSeparator) + subdir + string(os.PathSeparator) + filename
		s.createFile(filename, content, c)
	}
	return filenames
}

func (s *OssutilCommandSuite) rawCPWithFilter(args []string, recursive, force, update bool, threshold int64, cpDir string, cmdline []string, meta, acl string) (bool, error) {
	command := "cp"
	str := ""
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)

	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"meta":             &meta,
		"acl":              &acl,
	}
	os.Args = cmdline
	showElapse, err := cm.RunCommand(command, args, options)
	os.Args = []string{}
	return showElapse, err
}

func (s *OssutilCommandSuite) rawCPWithOutputDir(srcURL, destURL string, recursive, force, update bool, threshold int64, outputDir string) (bool, error) {
	command := "cp"
	str := ""
	args := []string{srcURL, destURL}
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"outputDir":        &outputDir,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawCPWithPayer(args []string, recursive, force, update bool, threshold int64, payer string) (bool, error) {
	command := "cp"
	str := ""
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":         &payerBucketEndPoint,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"payer":            &payer,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) initCopyCommand(srcURL, destURL string, recursive, force, update bool, threshold int64, cpDir, outputDir string) error {
	str := ""
	args := []string{srcURL, destURL}
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"outputDir":        &outputDir,
	}
	err := copyCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initCopyWithSnapshot(srcURL, destURL string, recursive, force, update bool, threshold int64, snapshotPath string) error {
	str := ""
	args := []string{srcURL, destURL}
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"snapshotPath":     &snapshotPath,
	}
	err := copyCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initCopyWithRange(srcURL, destURL string, recursive, force, update bool, threshold int64, vrange string) error {
	str := ""
	args := []string{srcURL, destURL}
	thre := strconv.FormatInt(threshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"recursive":        &recursive,
		"force":            &force,
		"update":           &update,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"range":            &vrange,
	}
	err := copyCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) putObject(bucket, object, fileName string, c *C) {
	args := []string{fileName, CloudURLToString(bucket, object)}
	showElapse, err := s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) getObject(bucket, object, fileName string, c *C) {
	args := []string{CloudURLToString(bucket, object), fileName}
	showElapse, err := s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) copyObject(srcBucket, srcObject, destBucket, destObject string, c *C) {
	args := []string{CloudURLToString(srcBucket, srcObject), CloudURLToString(destBucket, destObject)}
	showElapse, err := s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) rawGetStat(bucket, object string) (bool, error) {
	args := []string{CloudURLToString(bucket, object)}
	showElapse, err := s.rawGetStatWithArgs(args)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawGetStatWithArgs(args []string) (bool, error) {
	command := "stat"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) getStat(bucket, object string, c *C) map[string]string {
	args := []string{CloudURLToString(bucket, object)}
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	showElapse, err := s.rawGetStatWithArgs(args)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	os.Stdout = out

	// get result
	stat := s.getStatResults(c)
	os.Remove(resultPath)
	return stat
}

func (s *OssutilCommandSuite) getStatResults(c *C) map[string]string {
	result := s.getResult(c)
	c.Assert(len(result) > 1, Equals, true)

	stat := map[string]string{}
	for _, str := range result {
		sli := strings.SplitN(str, ":", 2)
		if len(sli) == 2 {
			stat[strings.TrimSpace(sli[0])] = strings.TrimSpace(sli[1])
		}
	}
	return stat
}

func (s *OssutilCommandSuite) getHashResults(c *C) map[string]string {
	result := s.getResult(c)
	c.Assert(len(result) >= 1, Equals, true)

	stat := map[string]string{}
	for _, str := range result {
		sli := strings.SplitN(str, ":", 2)
		if len(sli) == 2 {
			stat[strings.TrimSpace(sli[0])] = strings.TrimSpace(sli[1])
		}
	}
	return stat
}

func (s *OssutilCommandSuite) rawSetBucketACL(bucket, acl string, force bool) (bool, error) {
	args := []string{CloudURLToString(bucket, ""), acl}
	showElapse, err := s.rawSetACLWithArgs(args, false, true, force)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetObjectACL(bucket, object, acl string, recursive, force bool) (bool, error) {
	args := []string{CloudURLToString(bucket, object), acl}
	showElapse, err := s.rawSetACLWithArgs(args, recursive, false, force)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetACLWithArgs(args []string, recursive, bucket, force bool) (bool, error) {
	command := "set-acl"
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"recursive":       &recursive,
		"bucket":          &bucket,
		"force":           &force,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetAclWithFilter(args []string, recursive, force bool, cmdline []string) (bool, error) {
	command := "set-acl"
	str := ""
	routines := strconv.Itoa(Routines)

	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"recursive":       &recursive,
		"bucket":          false,
		"force":           &force,
	}

	os.Args = cmdline
	showElapse, err := cm.RunCommand(command, args, options)
	os.Args = []string{}
	return showElapse, err
}

func (s *OssutilCommandSuite) initSetACL(bucket, object, acl string, recursive, tobucket, force bool) error {
	args := []string{CloudURLToString(bucket, object), acl}
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"recursive":       &recursive,
		"bucket":          &tobucket,
		"force":           &force,
	}
	err := setACLCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initSetACLWithArgs(args []string, cmdline string, outputDir string) error {
	encodingType := ""
	if pos := strings.Index(cmdline, "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		cmdline = cmdline[0:pos] + cmdline[pos+len("--encoding-type url"):]
	}

	parameter := strings.Split(cmdline, "-")
	r := false
	f := false
	b := false
	if len(parameter) >= 2 {
		r = strings.Contains(parameter[1], "r")
		f = strings.Contains(parameter[1], "f")
		b = strings.Contains(parameter[1], "b")
	}

	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &r,
		"force":           &f,
		"bucket":          &b,
		"encodingType":    &encodingType,
		"routines":        &routines,
		"outputDir":       &outputDir,
	}
	err := setACLCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) setBucketACL(bucket, acl string, c *C) {
	args := []string{CloudURLToString(bucket, ""), acl}
	showElapse, err := s.rawSetACLWithArgs(args, false, true, false)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) setObjectACL(bucket, object, acl string, recursive, force bool, c *C) {
	args := []string{CloudURLToString(bucket, object), acl}
	showElapse, err := s.rawSetACLWithArgs(args, recursive, false, force)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) rawSetMeta(bucket, object, meta string, update, delete, recursive, force bool, language string) (bool, error) {
	args := []string{CloudURLToString(bucket, object), meta}
	showElapse, err := s.rawSetMetaWithArgs(args, update, delete, recursive, force, language)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetMetaWithArgs(args []string, update, delete, recursive, force bool, language string) (bool, error) {
	command := "set-meta"
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &update,
		"delete":          &delete,
		"recursive":       &recursive,
		"force":           &force,
		"routines":        &routines,
		"language":        &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetMetaWithPattern(bucket, object, meta string, update, delete, recursive, force bool, language, pattern string) (bool, error) {
	args := []string{CloudURLToString(bucket, object), meta}
	showElapse, err := s.rawSetMetaWithArgsWithPattern(args, update, delete, recursive, force, language, pattern)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetMetaWithArgsWithPattern(args []string, update, delete, recursive, force bool, language, pattern string) (bool, error) {
	command := "set-meta"
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &update,
		"delete":          &delete,
		"recursive":       &recursive,
		"force":           &force,
		"routines":        &routines,
		"language":        &language,
		"include":         &pattern,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) rawSetMetaWithFilter(args []string, update, delete, recursive, force bool, language string, cmdline []string) (bool, error) {
	command := "set-meta"
	str := ""
	routines := strconv.Itoa(Routines)

	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &update,
		"delete":          &delete,
		"recursive":       &recursive,
		"force":           &force,
		"routines":        &routines,
		"language":        &language,
	}

	os.Args = cmdline
	showElapse, err := cm.RunCommand(command, args, options)
	os.Args = []string{}
	return showElapse, err
}

func (s *OssutilCommandSuite) createTestObjects(dir, subdir, bucketStr string, c *C) []string {
	// Create dirs
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	objs := make([]string, 0)

	// Create files
	num := 3
	for i := 1020; i < 1020+num; i++ {
		filename := fmt.Sprintf("testfile%d.txt", i)
		objs = append(objs, filename)
		filename = dir + "/" + filename
		s.createFile(filename, filename, c)
	}

	num = 2
	for i := 2; i < 2+num; i++ {
		filename := fmt.Sprintf("test%d.jpg", i)
		objs = append(objs, filename)
		filename = dir + "/" + filename
		s.createFile(filename, filename, c)
	}

	num = 2
	for i := 103; i < 103+num; i++ {
		filename := fmt.Sprintf("testfile%d.txt", i)
		filename = subdir + "/" + filename
		objs = append(objs, filename)
		filename = dir + "/" + filename
		s.createFile(filename, filename, c)
	}
	filename := subdir + "/" + "my.rtf"
	objs = append(objs, filename)
	filename = dir + "/" + filename
	s.createFile(filename, filename, c)

	// Upload these files
	args := []string{dir, bucketStr}
	showElapse, err := s.rawCPWithArgs(args, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	return objs
}

func (s *OssutilCommandSuite) setObjectMeta(bucket, object, meta string, update, delete, recursive, force bool, c *C) {
	showElapse, err := s.rawSetMeta(bucket, object, meta, update, delete, recursive, force, DefaultLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) initSetMeta(bucket, object, meta string, update, delete, recursive, force bool, language string) error {
	args := []string{CloudURLToString(bucket, object), meta}
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &update,
		"delete":          &delete,
		"recursive":       &recursive,
		"force":           &force,
		"routines":        &routines,
		"language":        &language,
	}
	err := setMetaCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initSetMetaWithArgs(args []string, cmdline string, outputDir string) error {
	encodingType := ""
	if pos := strings.Index(cmdline, "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		cmdline = cmdline[0:pos] + cmdline[pos+len("--encoding-type url"):]
	}

	update := false
	if pos := strings.Index(cmdline, "--update"); pos != -1 {
		update = true
		cmdline = cmdline[0:pos] + cmdline[pos+len("--update"):]
	}

	delete := false
	if pos := strings.Index(cmdline, "--delete"); pos != -1 {
		delete = true
		cmdline = cmdline[0:pos] + cmdline[pos+len("--delete"):]
	}

	var disableIgnoreError bool
	if pos := strings.Index(cmdline, "--disable-ignore-error"); pos != -1 {
		disableIgnoreError = true
		cmdline = cmdline[0:pos] + cmdline[pos+len("--disable-ignore-error"):]
	}

	objectFile := ""
	snapshotPath := ""
	objectFile, snapshotPath, cmdline = s.handleObjectFileSnapshot(cmdline)

	parameter := strings.Split(cmdline, "-")
	r := false
	f := false
	if len(parameter) >= 2 {
		r = strings.Contains(parameter[1], "r")
		f = strings.Contains(parameter[1], "f")
	}

	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":           &str,
		"accessKeyID":        &str,
		"accessKeySecret":    &str,
		"stsToken":           &str,
		"configFile":         &configFile,
		"update":             &update,
		"delete":             &delete,
		"recursive":          &r,
		"force":              &f,
		"routines":           &routines,
		"encodingType":       &encodingType,
		"objectFile":         &objectFile,
		"snapshotPath":       &snapshotPath,
		"disableIgnoreError": &disableIgnoreError,
	}
	err := setMetaCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initCreateSymlink(cmdline string) error {
	encodingType := ""
	if pos := strings.Index(cmdline, "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		cmdline = cmdline[0:pos] + cmdline[pos+len("--encoding-type url"):]
	}

	cmds := strings.Split(cmdline, " ")
	args := []string{}
	for _, cmd := range cmds {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			args = append(args, cmd)
		}
	}

	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"encodingType":    &encodingType,
	}
	err := createSymlinkCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) readSymlink(cmdline string, c *C) map[string]string {
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err := s.initReadSymlink(cmdline)
	c.Assert(err, IsNil)
	err = readSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out

	// get result
	stat := s.getStatResults(c)
	os.Remove(resultPath)
	return stat
}

func (s *OssutilCommandSuite) initReadSymlink(cmdline string) error {
	encodingType := ""
	if pos := strings.Index(cmdline, "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		cmdline = cmdline[0:pos] + cmdline[pos+len("--encoding-type url"):]
	}

	cmds := strings.Split(cmdline, " ")
	args := []string{}
	for _, cmd := range cmds {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			args = append(args, cmd)
		}
	}

	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"encodingType":    &encodingType,
	}
	err := readSymlinkCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) initSignURL(cmdline, encodingType string, timeout int64) error {
	cmds := strings.Split(cmdline, " ")
	args := []string{}
	for _, cmd := range cmds {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			args = append(args, cmd)
		}
	}

	str := ""
	t := strconv.FormatInt(timeout, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"encodingType":    &encodingType,
		"timeout":         &t,
	}
	err := signURLCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) signURL(cmdline, encodingType string, timeout int64, c *C) string {
	out := os.Stdout
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testResultFile
	err := s.initSignURL(cmdline, encodingType, timeout)
	c.Assert(err, IsNil)
	err = signURLCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out

	results := s.getResult(c)
	c.Assert(len(results) >= 1, Equals, true)
	return results[0]
}

func (s *OssutilCommandSuite) getFileList(dpath string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(dpath, func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		dpath = filepath.Clean(dpath)
		fpath = filepath.Clean(fpath)
		fileName, err := filepath.Rel(dpath, fpath)
		if err != nil {
			return fmt.Errorf("list file error: %s, info: %s", fpath, err.Error())
		}

		if f.IsDir() {
			if fpath != dpath {
				fileList = append(fileList, fileName+string(os.PathSeparator))
			}
			return nil
		}
		fileList = append(fileList, fileName)
		return nil
	})
	return fileList, err
}

func (s *OssutilCommandSuite) handleObjectFileSnapshot(cmdline string) (objectFile, snapshotPath, newCmdline string) {
	cmds := strings.Split(cmdline, " ")
	for i := 0; i < len(cmds); i++ {
		if cmds[i] == "--object-file" && !strings.HasPrefix(cmds[i+1], "-") {
			objectFile = cmds[i+1]
		} else {
			continue
		}
		cmds = append(cmds[:i], cmds[i+2:]...)
	}
	for i := 0; i < len(cmds); i++ {
		if cmds[i] == "--snapshot-path" && !strings.HasPrefix(cmds[i+1], "-") {
			snapshotPath = cmds[i+1]
		} else {
			continue
		}
		cmds = append(cmds[:i], cmds[i+2:]...)
	}

	// remake cmdline
	cmdline = ""
	for i := 0; i < len(cmds); i++ {
		cmdline = cmdline + " " + cmds[i]
	}
	return objectFile, snapshotPath, cmdline
}

func (s *OssutilCommandSuite) initRestoreObject(args []string, cmdline string, outputDir string) error {
	encodingType := ""
	if pos := strings.Index(cmdline, "--encoding-type url"); pos != -1 {
		encodingType = URLEncodingType
		cmdline = cmdline[0:pos] + cmdline[pos+len("--encoding-type url"):]
	}

	var disableIgnoreError bool
	if pos := strings.Index(cmdline, "--disable-ignore-error"); pos != -1 {
		disableIgnoreError = true
		cmdline = cmdline[0:pos] + cmdline[pos+len("--disable-ignore-error"):]
	}

	objectFile := ""
	snapshotPath := ""
	objectFile, snapshotPath, cmdline = s.handleObjectFileSnapshot(cmdline)

	parameter := strings.Split(cmdline, "-")
	r := false
	f := false
	if len(parameter) >= 2 {
		r = strings.Contains(parameter[1], "r")
		f = strings.Contains(parameter[1], "f")
	}

	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":           &str,
		"accessKeyID":        &str,
		"accessKeySecret":    &str,
		"stsToken":           &str,
		"configFile":         &configFile,
		"recursive":          &r,
		"force":              &f,
		"encodingType":       &encodingType,
		"routines":           &routines,
		"outputDir":          &outputDir,
		"objectFile":         &objectFile,
		"snapshotPath":       &snapshotPath,
		"disableIgnoreError": &disableIgnoreError,
	}
	err := restoreCommand.Init(args, options)
	return err
}

func (s *OssutilCommandSuite) getErrorOSSBucket(bucketName string, c *C) *oss.Bucket {
	client, err := oss.New(endpoint, "abc", accessKeySecret, oss.EnableCRC(true))
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)
	return bucket
}

func (s *OssutilCommandSuite) TestParseOptions(c *C) {
	bucket := bucketNameExist
	s.putBucket(bucket, c)

	s.createFile(uploadFileName, content, c)

	// put object
	object := "中文"
	v := []string{"", "cp", uploadFileName, CloudURLToString(bucket, object), "-f", "--update", "--bigfile-threshold=1", "--checkpoint-dir=checkpoint_dir", "-c", configFile, "--loglevel=info"}
	os.Args = v
	err := ParseAndRunCommand()
	c.Assert(err, IsNil)

	// get object
	s.getObject(bucket, object, downloadFileName, c)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, content)
}

func (s *OssutilCommandSuite) TestNotExistCommand(c *C) {
	command := "notexistcmd"
	args := []string{}
	options := OptionMapType{}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestDecideConfigFile(c *C) {
	file := DecideConfigFile("")
	c.Assert(file, Equals, strings.Replace(DefaultConfigFile, "~", currentHomeDir(), 1))
	input := "~" + string(os.PathSeparator) + "a"
	file = DecideConfigFile(input)
	c.Assert(file, Equals, strings.Replace(input, "~", currentHomeDir(), 1))
}

func (s *OssutilCommandSuite) TestCheckConfig(c *C) {
	// config file not exist
	configMap := OptionMapType{OptionRetryTimes: "abc"}
	err := checkConfig(configMap)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestOptions(c *C) {
	option := Option{"", "", "", OptionTypeString, "", "", "", ""}
	_, err := stringOption(option)
	c.Assert(err, NotNil)

	option = Option{"", "", "", OptionTypeFlagTrue, "", "", "", ""}
	_, err = flagTrueOption(option)
	c.Assert(err, NotNil)

	option = Option{"-a", "", "", OptionTypeFlagTrue, "", "", "", ""}
	_, err = flagTrueOption(option)
	c.Assert(err, IsNil)

	str := "abc"
	options := OptionMapType{OptionRetryTimes: &str}
	err = checkOption(options)
	c.Assert(err, NotNil)

	str = "-1"
	options = OptionMapType{OptionRetryTimes: &str}
	err = checkOption(options)
	c.Assert(err, NotNil)

	str = "1001"
	options = OptionMapType{OptionRetryTimes: &str}
	err = checkOption(options)
	c.Assert(err, NotNil)

	language := "unknown"
	options = OptionMapType{OptionLanguage: &language}
	err = checkOption(options)
	c.Assert(err, NotNil)

	options = OptionMapType{OptionConfigFile: &configFile}
	ok, err := GetBool(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(ok, Equals, false)

	i, err := GetInt(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(i, Equals, int64(0))

	str = ""
	options = OptionMapType{OptionConfigFile: &str}
	i, err = GetInt(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(i, Equals, int64(0))

	options = OptionMapType{OptionRetryTimes: &str}
	i, err = GetInt(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(i, Equals, int64(0))

	ok = true
	options = OptionMapType{OptionRetryTimes: &ok}
	i, err = GetInt(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(i, Equals, int64(0))

	options = OptionMapType{OptionConfigFile: &ok}
	i, err = GetInt(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(i, Equals, int64(0))

	options = OptionMapType{OptionConfigFile: "a"}
	val, err := GetString(OptionConfigFile, options)
	c.Assert(err, NotNil)
	c.Assert(val, Equals, "")
}

func (s *OssutilCommandSuite) TestErrors(c *C) {
	err := CommandError{"help", "abc"}
	c.Assert(err.Error(), Equals, "invalid usage of \"help\" command, reason: abc, please try \"help help\" for more information")

	berr := BucketError{err, "b"}
	c.Assert(berr.Error(), Equals, fmt.Sprintf("%s, Bucket=%s", err.Error(), "b"))

	ferr := FileError{err, "f"}
	c.Assert(ferr.Error(), Equals, fmt.Sprintf("%s, File=%s", err.Error(), "f"))
}

func (s *OssutilCommandSuite) TestStorageURL(c *C) {
	var cloudURL CloudURL
	err := cloudURL.Init("/abc/d", "")
	c.Assert(err, IsNil)
	c.Assert(cloudURL.bucket, Equals, "abc")
	c.Assert(cloudURL.object, Equals, "d")

	dir := currentHomeDir()
	url := "~" + string(os.PathSeparator) + "test"
	var fileURL FileURL
	fileURL.Init(url, "")
	c.Assert(fileURL.urlStr, Equals, strings.Replace(url, "~", dir, 1))

	_, err = CloudURLFromString("oss:///object", "")
	c.Assert(err, NotNil)

	_, err = CloudURLFromString("./file", "")
	c.Assert(err, NotNil)

	cloudURL, err = CloudURLFromString("oss://bucket/%e4%b8%ad%e6%96%87%e6%b5%8b%e8%af%95", URLEncodingType)
	c.Assert(err, IsNil)
	c.Assert(cloudURL.bucket, Equals, "bucket")
	c.Assert(cloudURL.object, Equals, "中文测试")

	cloudURL, err = CloudURLFromString("oss://bucket/%e4%b8%ad%e6%96%87%e6%b5%8b%e8%af%95", "")
	c.Assert(err, IsNil)
	c.Assert(cloudURL.bucket, Equals, "bucket")
	c.Assert(cloudURL.object, Equals, "%e4%b8%ad%e6%96%87%e6%b5%8b%e8%af%95")

	cloudURL, err = CloudURLFromString("oss%3a%2f%2fbucket%2f%e4%b8%ad%e6%96%87%e6%b5%8b%e8%af%95", URLEncodingType)
	c.Assert(err, NotNil)

	storageURL, err := StorageURLFromString("oss%3a%2f%2fbucket%2f%e4%b8%ad%e6%96%87%e6%b5%8b%e8%af%95", URLEncodingType)
	c.Assert(err, IsNil)
	c.Assert(storageURL.IsCloudURL(), Equals, false)
	c.Assert(storageURL.IsFileURL(), Equals, true)
	c.Assert(storageURL.ToString(), Equals, "oss://bucket/中文测试")

	err = cloudURL.Init("oss:///abc/d", URLEncodingType)
	c.Assert(err, NotNil)

	cloudURL.object = "\\d"
	err = cloudURL.checkObjectPrefix()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestErrOssDownloadFile(c *C) {
	bucketName := bucketNamePrefix + "b1"
	str := ""
	args := []string{"", ""}
	thre := strconv.FormatInt(1, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"bigfileThreshold": &thre,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	err := copyCommand.Init(args, options)
	c.Assert(err, IsNil)
	bucket, err := copyCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	object := "object"
	err = copyCommand.ossDownloadFileRetry(bucket, object, object)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestUserAgent(c *C) {
	userAgent := getUserAgent("")
	c.Assert(userAgent != "", Equals, true)

	client, err := listCommand.command.ossClient("")
	c.Assert(err, IsNil)
	c.Assert(client, NotNil)
}

func (s *OssutilCommandSuite) TestParseAndRunCommand(c *C) {
	args := []string{}
	options := OptionMapType{}
	showElapse, err := RunCommand(args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestGetSizeString(c *C) {
	c.Assert(getSizeString(0), Equals, "0")
	c.Assert(getSizeString(1), Equals, "1")
	c.Assert(getSizeString(12), Equals, "12")
	c.Assert(getSizeString(123), Equals, "123")
	c.Assert(getSizeString(1234), Equals, "1,234")
	c.Assert(getSizeString(12345), Equals, "12,345")
	c.Assert(getSizeString(123456), Equals, "123,456")
	c.Assert(getSizeString(1234567), Equals, "1,234,567")
	c.Assert(getSizeString(123456789012), Equals, "123,456,789,012")
	c.Assert(getSizeString(1234567890123), Equals, "1,234,567,890,123")
	c.Assert(getSizeString(-0), Equals, "0")
	c.Assert(getSizeString(-1), Equals, "-1")
	c.Assert(getSizeString(-12), Equals, "-12")
	c.Assert(getSizeString(-123), Equals, "-123")
	c.Assert(getSizeString(-1234), Equals, "-1,234")
	c.Assert(getSizeString(-12345), Equals, "-12,345")
	c.Assert(getSizeString(-123456), Equals, "-123,456")
	c.Assert(getSizeString(-1234567), Equals, "-1,234,567")
	c.Assert(getSizeString(-123456789012), Equals, "-123,456,789,012")
	c.Assert(getSizeString(-1234567890123), Equals, "-1,234,567,890,123")
}

func (s *OssutilCommandSuite) TestNeedConfig(c *C) {
	str := ""
	args := []string{"", ""}
	thre := strconv.FormatInt(1, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	e := "a"
	options := OptionMapType{
		"endpoint":         &e,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &str,
		"bigfileThreshold": &thre,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	err := copyCommand.Init(args, options)
	c.Assert(err, IsNil)

	c.Assert(copyCommand.command.needConfigFile(), Equals, false)
}

func getFilesFromChanToArray(chFiles <-chan fileInfoType) []fileInfoType {
	files := make([]fileInfoType, 0)
	for f := range chFiles {
		files = append(files, f)
	}
	return files
}

func matchFiltersForFiles(files []fileInfoType, filters []filterOptionType) []fileInfoType {
	if len(filters) == 0 {
		return files
	}

	vsf := make([]fileInfoType, 0)

	for i, filter := range filters {
		if filter.name == IncludePrompt {
			res := filterFilesWithInclude(files, filter.pattern)
			for _, v := range res {
				if containsInFileSlice(vsf, v) {
					continue
				}
				vsf = append(vsf, v)
			}
		} else {
			if i == 0 {
				vsf = append(vsf, filterFilesWithExclude(files, filter.pattern)...)
			} else {
				vsf = filterFilesWithExclude(vsf, filter.pattern)
			}
		}
	}

	return vsf
}

func makeFileChanFromArray(files []fileInfoType, chFiles chan<- fileInfoType) {
	for _, f := range files {
		chFiles <- f
	}
}

func filterFilesFromChanWithPatterns(chFiles <-chan fileInfoType, filters []filterOptionType, dstFiles chan<- fileInfoType) {
	files := getFilesFromChanToArray(chFiles)
	vsf := matchFiltersForFiles(files, filters)
	makeFileChanFromArray(vsf, dstFiles)
	defer close(dstFiles)
}

func (s *OssutilCommandSuite) TestFilter(c *C) {
	var strs = []string{"peach", "apple", "pear", "plum"}

	res := filter(strs, func(v string) bool { return strings.Contains(v, "e") })
	var expect = []string{"peach", "apple", "pear"}
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"peach"}
	res = filter(strs, func(v string) bool { return strings.Contains(v, "h") })
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	suffix := ".txt"
	strs = []string{"a.jpg", "b.txt", "c.txt", "d"}
	expect = []string{"b.txt", "c.txt"}
	res = filter(strs, func(v string) bool { return strings.HasSuffix(v, suffix) })
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilter2(c *C) {
	strs := []string{"aa.jpg", "bb.txt", "cc.txt", "dd"}

	expect := []string{"bb.txt", "cc.txt"}
	res := filter2(strs, ".txt", strings.HasSuffix)
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"aa.jpg"}
	res = filter2(strs, ".jpg", strings.HasSuffix)
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"aa.jpg", "bb.txt", "cc.txt", "dd"}
	res = filter2(strs, "", strings.HasSuffix)
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterStr(c *C) {
	str := "aa.jpg"
	res := filterStr(str, ".jpg", strings.HasSuffix)
	c.Assert(res, Equals, true)

	res = filterStr(str, ".txt", strings.HasSuffix)
	c.Assert(res, Equals, false)
}

func (s *OssutilCommandSuite) TestFilterStrWithPattern(c *C) {
	str := "aabb1234ccdd.jpg"
	res := filterStrWithPattern(str, "*.jpg")
	c.Assert(res, Equals, true)

	res = filterStrWithPattern(str, "aaa*dd*")
	c.Assert(res, Equals, false)
}

func (s *OssutilCommandSuite) TestFilterStrsWithPattern(c *C) {
	strs := []string{"test1.jpg", "test8.txt", "test18.txt", "testfile"}

	expect := []string{"test8.txt", "test18.txt"}
	res := filterStrsWithPattern(strs, "*.txt")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test1.jpg"}
	res = filterStrsWithPattern(strs, "*.jpg")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test8.txt", "test18.txt"}
	res = filterStrsWithPattern(strs, "te*8.*xt")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterObjectsFromListResultWithPattern(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	num := 5
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestPattern_%d.txt", i)
		s.putObject(bucketName, object, uploadFileName, c)

		object = fmt.Sprintf("TestPattern_%d.jpg", i)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	err := s.initSetMeta(bucketName, "TestPattern", "", true, false, true, true, DefaultLanguage)
	c.Assert(err, IsNil)

	bucket, err := setMetaCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	lor, err := bucket.ListObjects()
	c.Assert(err, IsNil)

	objs := filterObjectsFromListResultWithPattern(lor, "*.txt")
	for _, obj := range objs {
		c.Assert(strings.HasSuffix(obj, ".txt"), Equals, true)
	}

	expect := []string{"TestPattern_4.jpg", "TestPattern_4.txt"}
	objs = filterObjectsFromListResultWithPattern(lor, "*att*4*")
	same := reflect.DeepEqual(objs, expect)
	c.Assert(same, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestFilterObjectsFromChanWithPattern(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	num := 5
	chObjects := make(chan string, ChannelBuf)
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("Test_Pattern_%d.txt", i)
		s.putObject(bucketName, object, uploadFileName, c)
		chObjects <- object

		object = fmt.Sprintf("Test_Pattern_%d.jpg", i)
		s.putObject(bucketName, object, uploadFileName, c)
		chObjects <- object
	}
	close(chObjects)

	chObjs := make(chan string, ChannelBuf)
	filterObjectsFromChanWithPattern(chObjects, "*Pat*[1-3]*.jpg", chObjs)

	expect := []string{"Test_Pattern_1.jpg", "Test_Pattern_2.jpg", "Test_Pattern_3.jpg"}
	objs := []string{}
	for obj := range chObjs {
		objs = append(objs, obj)
	}
	same := reflect.DeepEqual(objs, expect)
	c.Assert(same, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetFilter(c *C) {
	//e.g., ossutil cp oss://tempb4/ . -rf --include "*.txt" --exclude "*.jpg"
	cmdline := []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "*.txt", "--exclude", "*.jpg"}
	expect := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*.jpg"}}
	res, fts := getFilter(cmdline)
	same := reflect.DeepEqual(fts, expect)
	c.Assert(same, Equals, true)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "*/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "bin/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "/*/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--include", "/usr/bin/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "*/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "bin/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "/*/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "/usr/bin/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "bin/*.txt", "--include", "/usr/bin/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)

	cmdline = []string{"ossutil", "cp", "oss://tempb4", ".", "-rf", "--exclude", "/*/*.txt", "--include", "/*/*.txt"}
	res, fts = getFilter(cmdline)
	c.Assert(res, Equals, false)
}

func (s *OssutilCommandSuite) TestContainsInStrsSlice(c *C) {
	strs := []string{"dir1/my.rtf", "dir1/testfile103.txt", "testfile1021.txt"}

	tar := "dir1/my.rtf"
	res := containsInStrsSlice(strs, tar)
	c.Assert(res, Equals, true)

	tar = "dir1/myxx.rtf"
	res = containsInStrsSlice(strs, tar)
	c.Assert(res, Equals, false)

	strs = []string{}

	tar = "dir1/testfile103.txt"
	res = containsInStrsSlice(strs, tar)
	c.Assert(res, Equals, false)
}

func (s *OssutilCommandSuite) TestFilterSingleStr(c *C) {
	res := filterSingleStr("test18.txt", "*.txt", true)
	c.Assert(res, Equals, true)

	res = filterSingleStr("test28.txt", "te*8.*xt", true)
	c.Assert(res, Equals, true)

	res = filterSingleStr("test28.txt", "t8.*xt", true)
	c.Assert(res, Equals, false)

	res = filterSingleStr("test28.txt", "*.txt", false)
	c.Assert(res, Equals, false)

	res = filterSingleStr("test28.txt", "t8.*xt", false)
	c.Assert(res, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterStrsWithInclude(c *C) {
	strs := []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}

	expect := []string{"test18.txt", "test28.txt"}
	res := filterStrsWithInclude(strs, "*.txt")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test11.jpg"}
	res = filterStrsWithInclude(strs, "*.jpg")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test18.txt", "test28.txt"}
	res = filterStrsWithInclude(strs, "te*8.*xt")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterStrsWithExclude(c *C) {
	strs := []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}

	expect := []string{"test11.jpg", "testfile"}
	res := filterStrsWithExclude(strs, "*.txt")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test18.txt", "test28.txt", "testfile"}
	res = filterStrsWithExclude(strs, "*.jpg")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []string{"test11.jpg", "testfile"}
	res = filterStrsWithExclude(strs, "te*8.*xt")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMatchFiltersForStr(c *C) {
	fts := []filterOptionType{{"--include", "*.txt"}}
	res := matchFiltersForStr("test18.txt", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test28.txt", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test11.jpg", fts)
	c.Assert(res, Equals, false)
	res = matchFiltersForStr("testfile", fts)
	c.Assert(res, Equals, false)

	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = matchFiltersForStr("test11.jpg", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("testfile", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test18.txt", fts)
	c.Assert(res, Equals, false)
	res = matchFiltersForStr("test28.txt", fts)
	c.Assert(res, Equals, false)

	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	res = matchFiltersForStr("test18.txt", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test11.jpg", fts)
	c.Assert(res, Equals, false)
	res = matchFiltersForStr("test28.txt", fts)
	c.Assert(res, Equals, false)
	res = matchFiltersForStr("testfile", fts)
	c.Assert(res, Equals, false)

	fts = []filterOptionType{}
	res = matchFiltersForStr("test18.txt", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test28.txt", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("test11.jpg", fts)
	c.Assert(res, Equals, true)
	res = matchFiltersForStr("testfile", fts)
	c.Assert(res, Equals, true)
}

func (s *OssutilCommandSuite) TestMatchFiltersForStrs(c *C) {
	strs := []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}

	fts := []filterOptionType{{"--include", "*.txt"}}
	res := matchFiltersForStrs(strs, fts)
	expect := []string{"test18.txt", "test28.txt"}
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = matchFiltersForStrs(strs, fts)
	expect = []string{"test11.jpg", "testfile"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	res = matchFiltersForStrs(strs, fts)
	expect = []string{"test18.txt"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{}
	res = matchFiltersForStrs(strs, fts)
	expect = []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMatchFiltersForStrsInArray(c *C) {
	strs := []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}

	fts := []filterOptionType{{"--include", "*.txt"}}
	res := matchFiltersForStrsInArray(strs, fts)
	expect := []string{"test18.txt", "test28.txt"}
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = matchFiltersForStrsInArray(strs, fts)
	expect = []string{"test11.jpg", "testfile"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	res = matchFiltersForStrsInArray(strs, fts)
	expect = []string{"test18.txt"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{}
	res = matchFiltersForStrsInArray(strs, fts)
	expect = []string{"test11.jpg", "test18.txt", "test28.txt", "testfile"}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestDoesSingleFileMatchPatterns(c *C) {
	filename := "testfile12.jpg"
	fts := []filterOptionType{}
	res := doesSingleFileMatchPatterns(filename, fts)
	c.Assert(res, Equals, true)

	filename = "dir1/testfile1.txt"
	fts = []filterOptionType{{"--include", "*.txt"}}
	res = doesSingleFileMatchPatterns(filename, fts)
	c.Assert(res, Equals, true)

	filename = "dir1/testfile2.txt"
	fts = []filterOptionType{{"--include", "*.jpg"}}
	res = doesSingleFileMatchPatterns(filename, fts)
	c.Assert(res, Equals, false)

	filename = "dir1/testfile3.txt"
	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = doesSingleFileMatchPatterns(filename, fts)
	c.Assert(res, Equals, false)

	filename = "dir1/testfile3.txt"
	fts = []filterOptionType{{"--include", "*.txt"}, {"--include", "*.jpg"}, {"--exclude", "*2*"}}
	res = doesSingleFileMatchPatterns(filename, fts)
	c.Assert(res, Equals, true)
}

func (s *OssutilCommandSuite) TestGetFilesFromChanToArray(c *C) {
	chFiles := make(chan fileInfoType, ChannelBuf)
	chFiles <- fileInfoType{"dir1/my.rtf", "testdir/"}
	chFiles <- fileInfoType{"dir1/testfile103.txt", "testdir/"}
	chFiles <- fileInfoType{"testfile1021.txt", "testdir/"}
	close(chFiles)

	files := getFilesFromChanToArray(chFiles)
	expect := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	same := reflect.DeepEqual(files, expect)
	c.Assert(same, Equals, true)

	chFiles2 := make(chan fileInfoType, ChannelBuf)
	close(chFiles2)
	files = getFilesFromChanToArray(chFiles2)
	expect = []fileInfoType{}
	same = reflect.DeepEqual(files, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestcontainsInFileSlice(c *C) {
	files := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}

	tar := fileInfoType{"dir1/my.rtf", "testdir/"}
	res := containsInFileSlice(files, tar)
	c.Assert(res, Equals, true)

	tar = fileInfoType{"dir1/myxx.rtf", "testdirxx/"}
	res = containsInFileSlice(files, tar)
	c.Assert(res, Equals, false)

	files = []fileInfoType{}

	tar = fileInfoType{"dir1/testfile103.txt", "testdir/"}
	res = containsInFileSlice(files, tar)
	c.Assert(res, Equals, false)
}

func (s *OssutilCommandSuite) TestFilterFilesWithInclude(c *C) {
	files := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}

	expect := []fileInfoType{{"dir1/my.rtf", "testdir/"}}
	res := filterFilesWithInclude(files, "*.rtf")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []fileInfoType{}
	res = filterFilesWithInclude(files, "*.jpg")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	res = filterFilesWithInclude(files, "*")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []fileInfoType{}
	res = filterFilesWithInclude(files, "")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterFilesWithExclude(c *C) {
	files := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}

	expect := []fileInfoType{{"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	res := filterFilesWithExclude(files, "*.rtf")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []fileInfoType{}
	res = filterFilesWithExclude(files, "*")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	res = filterFilesWithExclude(files, "")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMatchFiltersForFiles(c *C) {
	files := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}

	fts := []filterOptionType{}
	res := matchFiltersForFiles(files, fts)
	expect := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}}
	res = matchFiltersForFiles(files, fts)
	expect = []fileInfoType{{"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = matchFiltersForFiles(files, fts)
	expect = []fileInfoType{{"dir1/my.rtf", "testdir/"}}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	res = matchFiltersForFiles(files, fts)
	expect = []fileInfoType{{"dir1/testfile103.txt", "testdir/"}}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMakeFileChanFromArray(c *C) {
	files := []fileInfoType{{"dir1/my.rtf", "testdir/"}, {"dir1/testfile103.txt", "testdir/"}, {"testfile1021.txt", "testdir/"}}
	chFiles := make(chan fileInfoType, ChannelBuf)
	makeFileChanFromArray(files, chFiles)
	defer close(chFiles)
	c.Assert(len(chFiles), Equals, len(files))

	files = []fileInfoType{}
	chFiles2 := make(chan fileInfoType, ChannelBuf)
	makeFileChanFromArray(files, chFiles2)
	defer close(chFiles2)
	c.Assert(len(chFiles2), Equals, len(files))
}

func (s *OssutilCommandSuite) TestFilterFileFromChanWithPatterns(c *C) {
	chFiles := make(chan fileInfoType, ChannelBuf)
	chFiles <- fileInfoType{"dir1/my.rtf", "testdir/"}
	chFiles <- fileInfoType{"dir1/testfile103.txt", "testdir/"}
	chFiles <- fileInfoType{"testfile1021.txt", "testdir/"}
	close(chFiles)
	fts := []filterOptionType{{"--include", "*.txt"}}
	dstFiles := make(chan fileInfoType, ChannelBuf)
	filterFilesFromChanWithPatterns(chFiles, fts, dstFiles)
	c.Assert(len(dstFiles), Equals, 2)

	chFiles = make(chan fileInfoType, ChannelBuf)
	chFiles <- fileInfoType{"dir1/my.rtf", "testdir/"}
	chFiles <- fileInfoType{"dir1/testfile103.txt", "testdir/"}
	chFiles <- fileInfoType{"testfile1021.txt", "testdir/"}
	close(chFiles)
	fts = []filterOptionType{{"--exclude", "*.txt"}}
	dstFiles2 := make(chan fileInfoType, ChannelBuf)
	filterFilesFromChanWithPatterns(chFiles, fts, dstFiles2)
	c.Assert(len(dstFiles2), Equals, 1)

	chFiles = make(chan fileInfoType, ChannelBuf)
	chFiles <- fileInfoType{"dir1/my.rtf", "testdir/"}
	chFiles <- fileInfoType{"dir1/testfile103.txt", "testdir/"}
	chFiles <- fileInfoType{"testfile1021.txt", "testdir/"}
	close(chFiles)
	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	dstFiles3 := make(chan fileInfoType, ChannelBuf)
	filterFilesFromChanWithPatterns(chFiles, fts, dstFiles3)
	c.Assert(len(dstFiles3), Equals, 1)
}

func (s *OssutilCommandSuite) TestDoesSingleObjectMatchPatterns(c *C) {
	object := "testfile3.jpg"
	fts := []filterOptionType{}
	res := doesSingleObjectMatchPatterns(object, fts)
	c.Assert(res, Equals, true)

	object = "dir1/testfile1.txt"
	fts = []filterOptionType{{"--include", "*.txt"}}
	res = doesSingleObjectMatchPatterns(object, fts)
	c.Assert(res, Equals, true)

	object = "dir1/testfile2.txt"
	fts = []filterOptionType{{"--include", "*.jpg"}}
	res = doesSingleObjectMatchPatterns(object, fts)
	c.Assert(res, Equals, false)

	object = "dir1/testfile3.txt"
	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = doesSingleObjectMatchPatterns(object, fts)
	c.Assert(res, Equals, false)

	object = "dir1/testfile4.txt"
	fts = []filterOptionType{{"--include", "*.txt"}, {"--include", "*.jpg"}, {"--exclude", "*2*"}}
	res = doesSingleObjectMatchPatterns(object, fts)
	c.Assert(res, Equals, true)
}

func (s *OssutilCommandSuite) TestGetObjectsFromChanToArray(c *C) {
	chObjects := make(chan objectInfoType, ChannelBuf)
	chObjects <- objectInfoType{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)}
	chObjects <- objectInfoType{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)}
	chObjects <- objectInfoType{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)}
	close(chObjects)

	files := getObjectsFromChanToArray(chObjects)
	expect := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	same := reflect.DeepEqual(files, expect)
	c.Assert(same, Equals, true)

	chObjects2 := make(chan objectInfoType, ChannelBuf)
	close(chObjects2)
	files = getObjectsFromChanToArray(chObjects2)
	expect = []objectInfoType{}
	same = reflect.DeepEqual(files, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterObjectsWithInclude(c *C) {
	objects := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}

	expect := []objectInfoType{{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)}}
	res := filterObjectsWithInclude(objects, "*.rtf")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []objectInfoType{}
	res = filterObjectsWithInclude(objects, "*.jpg")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	res = filterObjectsWithInclude(objects, "*")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []objectInfoType{}
	res = filterObjectsWithInclude(objects, "")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestFilterObjectsWithExclude(c *C) {
	objects := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}

	expect := []objectInfoType{
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	res := filterObjectsWithExclude(objects, "*.rtf")
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []objectInfoType{}
	res = filterObjectsWithExclude(objects, "*")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	expect = []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	res = filterObjectsWithExclude(objects, "")
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMatchFiltersForObjects(c *C) {
	objects := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}

	fts := []filterOptionType{}
	res := matchFiltersForObjects(objects, fts)
	expect := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	same := reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}}
	res = matchFiltersForObjects(objects, fts)
	expect = []objectInfoType{
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--exclude", "*.txt"}}
	res = matchFiltersForObjects(objects, fts)
	expect = []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
	}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)

	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	res = matchFiltersForObjects(objects, fts)
	expect = []objectInfoType{
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	same = reflect.DeepEqual(res, expect)
	c.Assert(same, Equals, true)
}

func (s *OssutilCommandSuite) TestMakeObjectChanFromArray(c *C) {
	objects := []objectInfoType{
		{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)},
		{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)},
	}
	chObjects := make(chan objectInfoType, ChannelBuf)
	makeObjectChanFromArray(objects, chObjects)
	defer close(chObjects)
	c.Assert(len(chObjects), Equals, len(objects))

	objects = []objectInfoType{}
	chObjects2 := make(chan objectInfoType, ChannelBuf)
	makeObjectChanFromArray(objects, chObjects2)
	defer close(chObjects2)
	c.Assert(len(chObjects2), Equals, len(objects))
}

func (s *OssutilCommandSuite) TestFilterObjectFromChanWithPatterns(c *C) {
	chObjects := make(chan objectInfoType, ChannelBuf)
	chObjects <- objectInfoType{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)}
	chObjects <- objectInfoType{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)}
	chObjects <- objectInfoType{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)}
	close(chObjects)
	fts := []filterOptionType{{"--include", "*.txt"}}
	dstObjects := make(chan objectInfoType, ChannelBuf)
	filterObjectsFromChanWithPatterns(chObjects, fts, dstObjects)
	c.Assert(len(dstObjects), Equals, 2)

	chObjects = make(chan objectInfoType, ChannelBuf)
	chObjects <- objectInfoType{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)}
	chObjects <- objectInfoType{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)}
	chObjects <- objectInfoType{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)}
	close(chObjects)
	fts = []filterOptionType{{"--exclude", "*.txt"}}
	dstObjects2 := make(chan objectInfoType, ChannelBuf)
	filterObjectsFromChanWithPatterns(chObjects, fts, dstObjects2)
	c.Assert(len(dstObjects2), Equals, 1)

	chObjects = make(chan objectInfoType, ChannelBuf)
	chObjects <- objectInfoType{"dir1/", "my.rtf", 10240, time.Date(2017, 10, 1, 7, 0, 0, 0, time.Local)}
	chObjects <- objectInfoType{"dir1/", "testfile103.txt", 1040, time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)}
	chObjects <- objectInfoType{"", "testfile1021.txt", 1024, time.Date(2017, 1, 19, 7, 10, 35, 0, time.UTC)}
	close(chObjects)
	fts = []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	dstObjects3 := make(chan objectInfoType, ChannelBuf)
	filterObjectsFromChanWithPatterns(chObjects, fts, dstObjects3)
	c.Assert(len(dstObjects3), Equals, 1)
}

func (s *OssutilCommandSuite) TestCommandLoglevel(c *C) {
	cfile := "ossutil-config" + randLowStr(8)
	level := "info"
	data := "[Credentials]" + "\n" + "language=" + DefaultLanguage + "\n" + "accessKeyID=" + accessKeyID + "\n" + "accessKeySecret=" + accessKeySecret + "\n" + "endpoint=" +
		endpoint + "\n" + "[Default]" + "\n" + "loglevel=" + level + "\n"
	s.createFile(cfile, data, c)

	f, err := os.Stat(cfile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)
	os.Args = []string{"ossutil", "ls", "oss://", "--config-file=" + cfile}
	os.Remove(logName)
	err = ParseAndRunCommand()
	c.Assert(err, IsNil)
	f, err = os.Stat(logName)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)
	strContent := s.readFile(logName, c)
	c.Assert(strings.Contains(strContent, "[info]"), Equals, true)

	os.Remove(logName)
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestGetLoglevelFromOptions(c *C) {
	level := "info"
	level2 := "debug"
	str := ""
	data := "[Credentials]" + "\n" + "language=" + DefaultLanguage + "\n" + "accessKeyID=" + accessKeyID + "\n" + "accessKeySecret=" + accessKeySecret + "\n" + "endpoint=" +
		endpoint + "\n" + "[Default]" + "\n" + "loglevel=" + level + "\n"
	s.createFile(configFile, data, c)
	options := OptionMapType{
		"loglevel": &level,
	}

	strLevel, err := getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level)
	testLogger.Print("loglevel 1" + strLevel)

	options = OptionMapType{
		"configFile": &configFile,
		"loglevel":   &level,
	}
	strLevel, err = getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level)
	testLogger.Print("loglevel 2" + strLevel)

	options = OptionMapType{
		"configFile": &configFile,
		"loglevel":   &level2,
	}
	strLevel, err = getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level2)
	testLogger.Print("loglevel 3" + strLevel)

	options = OptionMapType{
		"configFile": &configFile,
		"loglevel":   &str,
	}
	strLevel, err = getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level)
	testLogger.Print("loglevel 4" + strLevel)

	os.Remove(configFile)
	data = "[Credentials]" + "\n" + "language=" + DefaultLanguage + "\n" + "accessKeyID=" + accessKeyID + "\n" + "accessKeySecret=" + accessKeySecret + "\n" + "endpoint=" +
		endpoint + "\n" + "loglevel=" + level2 + "\n" + "[Default]" + "\n" + "loglevel=" + level + "\n"
	s.createFile(configFile, data, c)
	options = OptionMapType{
		"configFile": &configFile,
		"loglevel":   &str,
	}
	strLevel, err = getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level2)
	testLogger.Print("loglevel 5" + strLevel)

	os.Remove(configFile)
	data = "[Credentials]" + "\n" + "language=" + DefaultLanguage + "\n" + "accessKeyID=" + accessKeyID + "\n" + "accessKeySecret=" + accessKeySecret + "\n" + "endpoint=" +
		endpoint + "\n" + "log-level=" + level2 + "\n" + "[Default]" + "\n" + "loglevel=" + level + "\n"
	s.createFile(configFile, data, c)
	options = OptionMapType{
		"configFile": &configFile,
		"loglevel":   &str,
	}
	strLevel, err = getLoglevelFromOptions(options)
	c.Assert(err, IsNil)
	c.Assert(strLevel, Equals, level2)
	testLogger.Print("loglevel 5" + strLevel)
}

// test command objectProducer
func (s *OssutilCommandSuite) TestCommandObjectProducer(c *C) {
	chObjects := make(chan string, ChannelBuf)
	chListError := make(chan error, 1)
	cloudURL, err := CloudURLFromString(CloudURLToString(bucketNameNotExist, "demo.txt"), "")
	c.Assert(err, IsNil)
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)
	bucket, err := client.Bucket(bucketNameNotExist)
	c.Assert(err, IsNil)
	var filters []filterOptionType
	command := Command{}
	command.objectProducer(bucket, cloudURL, chObjects, chListError, filters)
	err = <-chListError
	c.Assert(err, NotNil)
	select {
	case _, ok := <-chObjects:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}

	chObjects2 := make(chan string, ChannelBuf)
	chListError2 := make(chan error, 1)
	storageURL2, err := CloudURLFromString(CloudURLToString(bucketNameExist, ""), "")
	c.Assert(err, IsNil)
	bucket2, err := client.Bucket(bucketNameExist)
	c.Assert(err, IsNil)
	command.objectProducer(bucket2, storageURL2, chObjects2, chListError2, filters)
	err = <-chListError2
	c.Assert(err, IsNil)
	select {
	case _, ok := <-chObjects2:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}
}
