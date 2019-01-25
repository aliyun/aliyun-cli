package cli

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/resource"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"runtime"
	"time"
)

func PlatformCompatible() {
	if runtime.GOOS == `windows` {
		DisableColor()
	}
	var ok bool
	if utils.TZData, ok = resource.GetTZData("GMT"); ok {
		utils.LoadLocationFromTZData = time.LoadLocationFromTZData
	}
}
