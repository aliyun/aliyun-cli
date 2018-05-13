package cli

import (
	"runtime"
	"time"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/utils"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/resource"
)

func PlatformCompatible() {
	if runtime.GOOS == `windows` {
		DisableColor()

		var ok bool
		if utils.TZData, ok = resource.GetTZData("GMT"); ok {
			utils.LoadLocationFromTZData = time.LoadLocationFromTZData
		}
	}
}