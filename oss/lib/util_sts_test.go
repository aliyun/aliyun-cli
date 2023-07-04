package lib

import (
	"fmt"
	"log"
	"os"
	"time"

	. "gopkg.in/check.v1"
)

type StsTestSuite struct {
}

var _ = Suite(&StsTestSuite{})

// Run once when the suite starts running
func (s *StsTestSuite) SetUpSuite(c *C) {
	stsAccessID = os.Getenv("OSS_TEST_STS_ID")
	stsAccessKeySecret = os.Getenv("OSS_TEST_STS_KEY")
	stsARN = os.Getenv("OSS_TEST_STS_ARN")
}

// Run after each test or benchmark starts running
func (s *StsTestSuite) TearDownSuite(c *C) {
}

func (s *StsTestSuite) TestSendRequest(c *C) {
	client := NewClient("", "", "", "")
	_, _, err := client.sendRequest(StsHost)
	c.Assert(err, IsNil)

	// negative
	_, _, err = client.sendRequest("https//x.y.z.com")
	c.Assert(err, NotNil)
}

func (s *StsTestSuite) TestHandleResponse(c *C) {
	client := NewClient("", "", "", "")

	body := "{\"RequestId\":\"784B99C1-895F-426C-8E1F-008955D418FB\"," +
		"\"HostId\":\"sts.aliyuncs.com\"," +
		"\"Code\":\"NoPermission\"," +
		"\"Message\":\"Roles may not be assumed by root accounts.\"}"
	resp, err := client.handleResponse([]byte(body), 400)
	_, isSuc := err.(*ServiceError)
	c.Assert(isSuc, Equals, true)
	c.Assert(resp, IsNil)

	body = "{{}}"
	resp, err = client.handleResponse([]byte(body), 400)
	_, isSuc = err.(*ServiceError)
	c.Assert(isSuc, Equals, false)
	c.Assert(resp, IsNil)

	body = "{\"RequestId\":\"4AB89022-25A3-4427-84A5-4C7E72BD63BE\"}"
	resp, err = client.handleResponse([]byte(body), 200)
	c.Assert(err, IsNil)
	c.Assert(resp, NotNil)

	body = "{{}}"
	resp, err = client.handleResponse([]byte(body), 200)
	_, isSuc = err.(*ServiceError)
	c.Assert(isSuc, Equals, false)
	c.Assert(resp, IsNil)
}

func (s *StsTestSuite) TestAssumeRoleSuccess(c *C) {
	now := time.Now()
	client := NewClient(stsAccessID, stsAccessKeySecret, stsARN, "sts_test")

	resp, err := client.AssumeRole(900, "")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("success!")
	}
	c.Assert(err, IsNil)

	c.Assert(resp.RequestId, Not(Equals), "")

	c.Assert(resp.AssumedRoleUser.Arn, Not(Equals), "")
	c.Assert(resp.AssumedRoleUser.AssumedRoleId, Not(Equals), "")

	c.Assert(resp.Credentials.AccessKeyId, Not(Equals), "")
	c.Assert(resp.Credentials.AccessKeySecret, Not(Equals), "")
	c.Assert(resp.Credentials.SecurityToken, Not(Equals), "")
	c.Assert(resp.Credentials.Expiration.After(now), Equals, true)
}

func (s *StsTestSuite) TestAssumeRoleNegative(c *C) {
	// AccessKeyID invalid
	client := NewClient("", accessKeySecret, stsARN, "sts_test")
	resp, err := client.AssumeRole(900, "")
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)
	log.Println("Error:", err)

	srvErr, isSuc := err.(*ServiceError)
	c.Assert(isSuc, Equals, true)
	c.Assert(srvErr.StatusCode, Equals, 400)
	log.Println("ServiceError:", srvErr)

	// AccessKeySecret invalid
	client = NewClient(stsAccessID, stsAccessKeySecret+" ", stsARN, "sts_test")
	resp, err = client.AssumeRole(900, "")
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)

	srvErr, isSuc = err.(*ServiceError)
	c.Assert(isSuc, Equals, true)
	c.Assert(srvErr.StatusCode, Equals, 400)
	c.Assert(srvErr.Code, Equals, "SignatureDoesNotMatch")
	log.Println("ServiceError:", srvErr)

	// SessionName invalid
	client = NewClient(stsAccessID, stsAccessKeySecret, stsARN, "x")

	resp, err = client.AssumeRole(900, "")
	c.Assert(resp, IsNil)
	c.Assert(err, NotNil)

	srvErr, isSuc = err.(*ServiceError)
	c.Assert(isSuc, Equals, true)
	c.Assert(srvErr.StatusCode, Equals, 400)
	c.Assert(srvErr.Code, Equals, "InvalidParameter.RoleSessionName")
	log.Println("ServiceError:", srvErr)
}
