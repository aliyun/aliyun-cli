package cloudapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

var GroupIdFlag = &cli.Flag{
	Name: "GroupId",
	Short: i18n.T("AppGroupId, in guid", "App分组ID, Guid格式"),
	AssignedMode: cli.AssignedOnce,
	Required: true,
}

var AoneAppNameFlag = &cli.Flag {
	Name: "AoneAppName",
	Short: i18n.T("", ""),
	AssignedMode: cli.AssignedOnce,
	Hidden: true,
}

var DeleteAllFlag = &cli.Flag {
	Name: "delete-all",
	Short: i18n.T("delete all api with in swagger", ""),
	AssignedMode: cli.AssignedNone,
}


