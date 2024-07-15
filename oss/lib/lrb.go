package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseListRegionBucket = SpecText{

	synopsisText: "列举某个region下的Buckets",

	paramText: "[conf_file] [options]",

	syntaxText: ` 
    ossutil lrb [conf_file]  [-e endpoint]
`,

	detailHelpText: `
    该命令列举单个region或者多个region下的bucket列表, 多个region的endpoint信息在配置文件里面配置
`,

	sampleText: ` 
    1) ossutil lrb conf_file
       
    2) ossutil lrb -e oss-cn-shenzhen.aliyuncs.com

    3) ossutil lrb
`,
}

var specEnglishListRegionBucket = SpecText{

	synopsisText: "List region buckets",

	paramText: "[conf_file] [options]",

	syntaxText: ` 
    ossutil lrb [conf_file]  [-e endpoint]
`,

	detailHelpText: ` 
    This command lists buckets under a single region or multiple regions
    The multiple regions's endpoints are configured in the configuration file
`,

	sampleText: ` 
    1) ossutil lrb conf_file
       
    2) ossutil lrb -e oss-cn-shenzhen.aliyuncs.com

    3) ossutil lrb
`,
}

// LrbCommand is the command list region buckets or objects
type LrbCommand struct {
	command        Command
	listResult     []oss.ListBucketsResult
	errorEndpoints []string
	err            error
}

var lrbCommand = LrbCommand{
	command: Command{
		name:        "lrb",
		nameAlias:   []string{"lrb"},
		minArgc:     0,
		maxArgc:     1,
		specChinese: specChineseListRegionBucket,
		specEnglish: specEnglishListRegionBucket,
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
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
		},
	},
}

// function for FormatHelper interface
func (lc *LrbCommand) formatHelpForWhole() string {
	return lc.command.formatHelpForWhole()
}

func (lc *LrbCommand) formatIndependHelp() string {
	return lc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (lc *LrbCommand) Init(args []string, options OptionMapType) error {
	return lc.command.Init(args, options, lc)
}

// RunCommand simulate inheritance, and polymorphism
func (lc *LrbCommand) RunCommand() error {
	var err error
	if len(lc.command.args) == 0 {
		lc.err = lc.listBuckets("")
		return lc.display()
	}
	// read all endpoints from conf file
	fileName := lc.command.args[0]
	rf, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer rf.Close()

	rd := bufio.NewReader(rf)
	for {
		endpoint, err := rd.ReadString('\n')
		if endpoint == "" && io.EOF == err {
			break
		}
		endpoint = strings.TrimSpace(endpoint)
		endpoint = strings.Trim(endpoint, "\r")
		if strings.HasPrefix(endpoint, "#") {
			continue
		}
		err = lc.listBuckets(endpoint)
		if err != nil {
			lc.errorEndpoints = append(lc.errorEndpoints, endpoint)
			lc.err = err
		}
	}
	return lc.display()
}

func (lc *LrbCommand) listBuckets(endpoint string) error {
	if endpoint != "" {
		lc.command.options[OptionEndpoint] = &endpoint
	}

	var err error
	client, err := lc.command.ossClient("")
	if err != nil {
		return err
	}

	// list region bucket
	pre := oss.Prefix("")
	marker := oss.Marker("")

	for {
		lbr, err := client.ListBuckets(pre, marker, oss.AddParam("regionList", ""))
		if err != nil {
			return err
		}
		lc.listResult = append(lc.listResult, lbr)
		pre = oss.Prefix(lbr.Prefix)
		marker = oss.Marker(lbr.NextMarker)
		if !lbr.IsTruncated {
			break
		}
	}
	return nil
}

func (lc *LrbCommand) display() error {
	count := 0
	for _, result := range lc.listResult {
		for _, bucket := range result.Buckets {
			if count == 0 {
				fmt.Printf("%-30s %20s%s%12s%s%s\n", "CreationTime", "Region", FormatTAB, "StorageClass", FormatTAB, "BucketName")
			}
			fmt.Printf("%-30s %20s%s%12s%s%s\n", utcToLocalTime(bucket.CreationDate), bucket.Location, FormatTAB, bucket.StorageClass, FormatTAB, CloudURLToString(bucket.Name, ""))
			count = count + 1
		}
	}

	fmt.Printf("\nBucket Number is: %d\n\n", count)
	if len(lc.errorEndpoints) > 0 {
		fmt.Printf("list bucket failure from these endpoint:\n")
		for _, endpoint := range lc.errorEndpoints {
			fmt.Printf("%s\n", endpoint)
		}
		fmt.Printf("\n")
	}
	return lc.err
}
