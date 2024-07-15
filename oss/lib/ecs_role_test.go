package lib

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

const (
	TestEcsTimeout int64 = 2
)
const (
	TIME_LAYOUT = "2006-01-02T15:04:05Z"
)

var requestCount int

func StsHttpHandlerOk(w http.ResponseWriter, r *http.Request) {
	akJson := &STSAkJson{}

	if accessKeyID == "" {
		accessKeyID = os.Getenv("OSS_TEST_ACCESS_KEY_ID")
	}

	if accessKeySecret == "" {
		accessKeySecret = os.Getenv("OSS_TEST_ACCESS_KEY_SECRET")
	}

	akJson.AccessKeyId = accessKeyID
	akJson.AccessKeySecret = accessKeySecret
	akJson.SecurityToken = ""
	nowLocalTime := time.Now()

	expirationTime := nowLocalTime.Add(time.Second * time.Duration(AdvanceSeconds+TestEcsTimeout))
	akJson.Expiration = expirationTime.UTC().Format(TIME_LAYOUT)

	akJson.LastUpDated = nowLocalTime.UTC().Format(TIME_LAYOUT)
	akJson.Code = "Success"
	bs, _ := json.Marshal(akJson)
	w.Write(bs)
}

func StsHttpHandlerProviderError(w http.ResponseWriter, r *http.Request) {
	requestCount++
	akJson := &STSAkJson{}

	if requestCount <= 3 {
		time.Sleep(15 * time.Second)
	}

	if accessKeyID == "" {
		accessKeyID = os.Getenv("OSS_TEST_ACCESS_KEY_ID")
	}

	if accessKeySecret == "" {
		accessKeySecret = os.Getenv("OSS_TEST_ACCESS_KEY_SECRET")
	}

	akJson.AccessKeyId = accessKeyID
	akJson.AccessKeySecret = accessKeySecret
	akJson.SecurityToken = ""
	nowLocalTime := time.Now()

	expirationTime := nowLocalTime.Add(time.Second * time.Duration(AdvanceSeconds+TestEcsTimeout))
	akJson.Expiration = expirationTime.UTC().Format(TIME_LAYOUT)

	akJson.LastUpDated = nowLocalTime.UTC().Format(TIME_LAYOUT)
	akJson.Code = "Success"
	bs, _ := json.Marshal(akJson)
	w.Write(bs)
}

func StsHttpHandlerEmptyError(w http.ResponseWriter, r *http.Request) {
	akJson := &STSAkJson{}
	bs, _ := json.Marshal(akJson)
	w.Write(bs)
}

func StsHttpHandlerCodeError(w http.ResponseWriter, r *http.Request) {
	akJson := &STSAkJson{}
	akJson.Code = "Error"
	bs, _ := json.Marshal(akJson)
	w.Write(bs)
}

func StsHttpHandlerJsonError(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("it is not valid json"))
}

func startHttpServer(handler func(http.ResponseWriter, *http.Request)) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	svr := &http.Server{
		Addr:           "127.0.0.1:32915",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        mux,
	}

	go func() {
		svr.ListenAndServe()
	}()
	return svr
}

func (s *OssutilCommandSuite) TestEcsRoleSuccess(c *C) {
	svr := startHttpServer(StsHttpHandlerOk)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	svr.Close()

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestEcsRoleAkTimeout(c *C) {
	svr := startHttpServer(StsHttpHandlerOk)
	time.Sleep(time.Duration(1) * time.Second)

	ecsRole := EcsRoleAKBuild{url: "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"}
	cred, err := ecsRole.GetCredentialsE()
	c.Assert(err, IsNil)
	strKeyId1 := cred.GetAccessKeyID()
	c.Assert(strKeyId1 == "", Equals, false)
	Expiration1 := ecsRole.Expiration

	// wait Ak timeout
	time.Sleep(time.Duration(1+TestEcsTimeout) * time.Second)

	cred, err = ecsRole.GetCredentialsE()
	c.Assert(err, IsNil)
	strKeyId2 := cred.GetAccessKeyID()
	c.Assert(strKeyId2 == "", Equals, false)
	Expiration2 := ecsRole.Expiration

	c.Assert(strKeyId1, Equals, strKeyId2)
	c.Assert(Expiration1 == Expiration2, Equals, false)
	svr.Close()

}

func (s *OssutilCommandSuite) TestEcsRoleNotHttpServerError(c *C) {
	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)

	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestEcsRoleAkEmptyError(c *C) {

	svr := startHttpServer(StsHttpHandlerEmptyError)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)

	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	svr.Close()
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestEcsRoleCodeError(c *C) {
	svr := startHttpServer(StsHttpHandlerCodeError)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)

	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	svr.Close()
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestEcsRoleJsonError(c *C) {
	svr := startHttpServer(StsHttpHandlerJsonError)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)

	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	svr.Close()
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestEcsRoleProviderError(c *C) {
	svr := startHttpServer(StsHttpHandlerProviderError)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers"), Equals, true)

	options[OptionRetryTimes] = 3

	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	svr.Close()
	os.Remove(cfile)
}
