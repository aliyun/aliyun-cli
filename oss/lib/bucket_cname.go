package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketCname = SpecText{
	synopsisText: "管理bucket cname以及cname token配置",

	paramText: "bucket_url [options]",

	syntaxText: ` 
    ossutil bucket-cname --method put --item token oss://bucket test-domain.com [local_xml_file] [options]
    ossutil bucket-cname --method get --item token oss://bucket test-domain.com [local_xml_file] [options]
    ossutil bucket-cname --method put oss://bucket test-domain.com [options]
    ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file [options]
    ossutil bucket-cname --method delete oss://bucket test-domain.com [options]
    ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    cname命令通过设置method选项值可以创建、删除、查询bucket的cname配置

用法:
    该命令有六种用法:

    1) ossutil bucket-cname --method put --item token oss://bucket test-domain.com [local_xml_file] [options]
        该命令会创建一个内部用的token, 设置bucket cname必须先创建这个token
        如果输入参数local_xml_file，token结果将输出到该文件，否则输出到屏幕上

    2) ossutil bucket-cname --method get --item token oss://bucket test-domain.com [local_xml_file] [options]
        该命令查询token信息
        如果输入参数local_xml_file，token结果将输出到该文件，否则输出到屏幕上

    3) ossutil bucket-cname --method put oss://bucket test-domain.com
        这个命令设置bucket的cname配置
 
    4) ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file [options]
        这个命令设置bucket的cname配置,并绑定https证书。或者解绑cname的https证书
        local_xml_file是xml格式，以下是local_xml_file内容的示例

       <?xml version="1.0" encoding="utf-8"?>
       <BucketCnameConfiguration>
         <Cname>
           <Domain>example.com</Domain>
           <CertificateConfiguration>
             <CertId>493****-cn-hangzhou</CertId>
             <Certificate>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</Certificate>
             <PrivateKey>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</PrivateKey>
             <PreviousCertId>493****-cn-hangzhou</PreviousCertId>
             <Force>true</Force>
           </CertificateConfiguration>
         </Cname>
       </BucketCnameConfiguration>

       或

       <?xml version="1.0" encoding="utf-8"?>
       <BucketCnameConfiguration>
         <Cname>
           <Domain>example.com</Domain>
             <CertificateConfiguration>
             <DeleteCertificate>True</DeleteCertificate>
           </CertificateConfiguration>
         </Cname>
       </BucketCnameConfiguration>

    5) ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
        这个命令查询bucket的cname配置
        如果输入参数local_xml_file，结果将输出到该文件，否则输出到屏幕上

    6) ossutil bucket-cname --method delete oss://bucket test-domain.com [options]
        这个命令删除bucket的cname配置
`,
	sampleText: ` 
    1) 创建bucket的cname token，执行结果输出到标准输出
       ossutil bucket-cname --method put --item token oss://bucket test-domain.com

    2) 创建bucket的cname token，执行结果输出到本地文件
       ossutil bucket-cname --method put --item token oss://bucket test-domain.com local_xml_file

    3) 获取bucket的cname token，结果输出到标准输出
       ossutil bucket-cname --method get --item token oss://bucket test-domain.com

    3) 获取bucket的cname token，结果输出到本地文件
       ossutil bucket-cname --method get --item token oss://bucket test-domain.com local_xml_file

    4) 设置bucket的cname配置
        ossutil bucket-cname --method put oss://bucket test-domain.com

    5) 设置bucket的cname配置,并绑定ssl证书 或者解绑ssl证书
        ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file

    6) 查询bucket的cname配置，结果输出到标准输出
       ossutil bucket-cname --method get oss://bucket

    7) 查询bucket的cname配置，结果输出到本地文件
       ossutil bucket-cname --method get oss://bucket local_xml_file
   
    8) 删除bucket的cname配置
       ossutil bucket-cname --method delete oss://bucket test-domain.com

`,
}

var specEnglishBucketCname = SpecText{
	synopsisText: "manage bucket cname and cname token configuration",

	paramText: "bucket_url [options]",

	syntaxText: ` 
    ossutil bucket-cname --method put --item token oss://bucket test-domain.com [local_xml_file] [options]
    ossutil bucket-cname --method get --item token oss://bucket test-domain.com [local_xml_file] [options]
    ossutil bucket-cname --method put oss://bucket test-domain.com [options]
    ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file [options]
    ossutil bucket-cname --method delete oss://bucket  test-domain.com [options]
    ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    The command can create, delete and query the cname configuration of a bucket by setting the method option value

Usage:
   There are six usages for this command:

    1) ossutil bucket-cname --method put --item token oss://bucket test-domain.com [local_xml_file] [options]
       This command will create an internal token, which must be created before setting bucket cname
       If you input parameter local_xml_file,the result will be output to local_xml_file
       If you don't input parameter local_xml_file,the result will be output to stdout

    2) ossutil bucket-cname --method get --item token oss://bucket test-domain.com [local_xml_file] [options]
       This command queries the token information
       If you input parameter local_xml_file,the token configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the token configuration will be output to stdout

    3) ossutil bucket-cname --method put oss://bucket test-domain.com [options]
       This command sets the cname configuration of the bucket
    
    4) ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file [options]
       This command sets the cname configuration of the bucket and binds the https certificate. Or unbind cname’s https certificate
       the local_xml_file is xml format,The following is an example of the contents of local_xml_file

       <?xml version="1.0" encoding="utf-8"?>
       <BucketCnameConfiguration>
         <Cname>
           <Domain>example.com</Domain>
           <CertificateConfiguration>
             <CertId>493****-cn-hangzhou</CertId>
             <Certificate>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</Certificate>
             <PrivateKey>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</PrivateKey>
             <PreviousCertId>493****-cn-hangzhou</PreviousCertId>
             <Force>true</Force>
           </CertificateConfiguration>
         </Cname>
       </BucketCnameConfiguration>

       or

       <?xml version="1.0" encoding="utf-8"?>
       <BucketCnameConfiguration>
         <Cname>
           <Domain>example.com</Domain>
             <CertificateConfiguration>
             <DeleteCertificate>True</DeleteCertificate>
           </CertificateConfiguration>
         </Cname>
       </BucketCnameConfiguration>

    5) ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
       This command queries the cname configuration of the bucket
       If you input parameter local_xml_file,the cname configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the cname configuration will be output to stdout

    6) ossutil bucket-cname --method delete oss://bucket test-domain.com [options]
       This command delete the cname configuration of the bucket
`,
	sampleText: ` 
    1) put bucket cname token, get result to stdout
       ossutil bucket-cname --method put --item token oss://bucket test-domain.com

    2) put bucket cname token, get result to local file
       ossutil bucket-cname --method put --item token oss://bucket test-domain.com local_xml_file

    3) get bucket cname token to stdout
       ossutil bucket-cname --method get --item token oss://bucket test-domain.com

    3) get bucket cname token to local file
       ossutil bucket-cname --method get --item token oss://bucket test-domain.com local_xml_file

    4) set bucket cname configuration
        ossutil bucket-cname --method put oss://bucket test-domain.com

    5) set bucket cname configuration,bind certificate or unbind certificate 
        ossutil bucket-cname --method put --item certificate oss://bucket local_xml_file

    6) get cname configuration to stdout
       ossutil bucket-cname --method get oss://bucket

    7) get cname configuration to local file
       ossutil bucket-cname --method get oss://bucket local_xml_file
   
    8) delete cname configuration
       ossutil bucket-cname --method delete oss://bucket test-domain.com

`,
}

type bucketCnameOptionType struct {
	bucketName string
	client     *oss.Client
}

type BucketCnameCommand struct {
	command  Command
	bwOption bucketCnameOptionType
}

var bucketCnameCommand = BucketCnameCommand{
	command: Command{
		name:        "bucket-cname",
		nameAlias:   []string{"bucket-cname"},
		minArgc:     1,
		maxArgc:     3,
		specChinese: specChineseBucketCname,
		specEnglish: specEnglishBucketCname,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionLogLevel,
			OptionMethod,
			OptionPassword,
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionReadTimeout,
			OptionConnectTimeout,
			OptionSTSRegion,
			OptionSkipVerifyCert,
			OptionUserAgent,
			OptionItem,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
		},
	},
}

// function for FormatHelper interface
func (bwc *BucketCnameCommand) formatHelpForWhole() string {
	return bwc.command.formatHelpForWhole()
}

func (bwc *BucketCnameCommand) formatIndependHelp() string {
	return bwc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bwc *BucketCnameCommand) Init(args []string, options OptionMapType) error {
	return bwc.command.Init(args, options, bwc)
}

// RunCommand simulate inheritance, and polymorphism
func (bwc *BucketCnameCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bwc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strItem, _ := GetString(OptionItem, bwc.command.options)
	strMethod = strings.ToLower(strMethod)
	srcBucketUrL, err := GetCloudUrl(bwc.command.args[0], "")
	if err != nil {
		return err
	}

	bwc.bwOption.bucketName = srcBucketUrL.bucket
	bwc.bwOption.client, err = bwc.command.ossClient(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	err = nil
	switch strItem {
	case "":
		switch strMethod {
		case "get":
			err = bwc.GetBucketCname()
		case "put":
			err = bwc.PutBucketCname()
		case "delete":
			err = bwc.DeleteBucketCname()
		default:
			err = fmt.Errorf("--method only support get,put,delete")
		}
	case "token":
		switch strMethod {
		case "get":
			err = bwc.GetBucketCnameToken()
		case "put":
			err = bwc.PutBucketCnameToken()
		default:
			err = fmt.Errorf("only support get bucket token or put bucket token")
		}
	case "certificate":
		switch strMethod {
		case "put":
			err = bwc.PutBucketCnameWithCertificate()
		default:
			err = fmt.Errorf("only support put bucket with certificate")
		}
	default:
		err = fmt.Errorf("--item only support token or certificate")
	}
	return err
}

func (bwc *BucketCnameCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket cname: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bwc *BucketCnameCommand) GetBucketCname() error {
	client := bwc.bwOption.client
	output, err := client.GetBucketCname(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}
	var outFile *os.File
	if len(bwc.command.args) >= 2 {
		fileName := bwc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bwc.confirm(fileName)
			if !bConitnue {
				return nil
			}
		}

		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}
	outFile.Write([]byte(output))
	fmt.Printf("\n\n")
	return nil
}

func (bwc *BucketCnameCommand) PutBucketCname() error {
	client := bwc.bwOption.client
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("cname is emtpy")
	}
	cname := bwc.command.args[1]
	return client.PutBucketCname(bwc.bwOption.bucketName, cname)
}

func (bwc *BucketCnameCommand) PutBucketCnameWithCertificate() error {
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("missing parameters,the local cname config file is empty")
	}
	cnameFile := bwc.command.args[1]
	fileInfo, err := os.Stat(cnameFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", cnameFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", cnameFile)
	}

	// parsing the xml file
	file, err := os.Open(cnameFile)
	if err != nil {
		return err
	}
	defer file.Close()
	xmlBody, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	client := bwc.bwOption.client
	return client.PutBucketCnameXml(bwc.bwOption.bucketName, string(xmlBody))
}

func (bwc *BucketCnameCommand) DeleteBucketCname() error {
	client := bwc.bwOption.client
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("cname is emtpy")
	}
	cname := bwc.command.args[1]
	err := client.DeleteBucketCname(bwc.bwOption.bucketName, cname)
	return err
}

func (bwc *BucketCnameCommand) GetBucketCnameToken() error {
	client := bwc.bwOption.client
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("cname is emtpy")
	}
	cname := bwc.command.args[1]
	out, err := client.GetBucketCnameToken(bwc.bwOption.bucketName, cname)
	if err != nil {
		return err
	}
	var outFile *os.File
	if len(bwc.command.args) >= 3 {
		fileName := bwc.command.args[2]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bwc.confirm(fileName)
			if !bConitnue {
				return nil
			}
		}

		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}
	var strXml []byte
	var xmlError error
	if strXml, xmlError = xml.MarshalIndent(out, "", " "); xmlError != nil {
		return xmlError
	}
	outFile.Write(strXml)
	fmt.Printf("\n\n")
	return err
}

func (bwc *BucketCnameCommand) PutBucketCnameToken() error {
	client := bwc.bwOption.client
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("cname is emtpy")
	}
	cname := bwc.command.args[1]
	output, err := client.CreateBucketCnameToken(bwc.bwOption.bucketName, cname)
	if err != nil {
		return err
	}
	var outFile *os.File
	if len(bwc.command.args) >= 3 {
		fileName := bwc.command.args[2]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bwc.confirm(fileName)
			if !bConitnue {
				return nil
			}
		}

		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}
	var strXml []byte
	var xmlError error
	if strXml, xmlError = xml.MarshalIndent(output, "", " "); xmlError != nil {
		return xmlError
	}
	outFile.Write(strXml)
	fmt.Printf("\n\n")
	return err
}
