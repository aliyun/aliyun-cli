package config

import (
	"os"
	"runtime"
)

func GetXDGConfigHome() string {
	if xgh := os.Getenv("XDG_CONFIG_HOME"); xgh != "" {
		return xgh
	} else {
		return GetHomePath() + "/.config"
	}
}

func GetConfigDirPath() string {
	// ~/.aliyun/ 存在则是老的配置路径
	// 否则：使用 XDG 规范
	home := GetHomePath()
	path := home + "/.aliyun"
	_, err := os.Stat(path)
	// 目录存在
	if err != nil {
		return path
	}

	return GetXDGConfigHome() + "/aliyun"
}

func GetHomePath() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}
