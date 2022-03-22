// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/aliyun/aliyun-cli/cli"
)

const (
	configPath               = "/.aliyun"
	configFile               = "config.json"
	DefaultConfigProfileName = "default"
)

type Configuration struct {
	CurrentProfile string    `json:"current"`
	Profiles       []Profile `json:"profiles"`
	MetaPath       string    `json:"meta_path"`
	//Plugins 		[]Plugin `json:"plugin"`
}

var hookGetHomePath = func(fn func() string) func() string {
	return fn
}

func NewConfiguration() *Configuration {
	return &Configuration{
		CurrentProfile: DefaultConfigProfileName,
		Profiles: []Profile{
			NewProfile(DefaultConfigProfileName),
		},
	}
}

func (c *Configuration) NewProfile(pn string) Profile {
	p, ok := c.GetProfile(pn)
	if !ok {
		p = NewProfile(pn)
		c.PutProfile(p)
	}
	return p
}

func (c *Configuration) GetProfile(pn string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.Name == pn {
			return p, true
		}
	}
	return Profile{Name: pn}, false
}

func (c *Configuration) GetCurrentProfile(ctx *cli.Context) Profile {
	profileName := ProfileFlag(ctx.Flags()).GetStringOrDefault(c.CurrentProfile)
	if profileName == "" || profileName == "default" {
		switch {
		case os.Getenv("ALIBABACLOUD_PROFILE") != "":
			profileName = os.Getenv("ALIBABACLOUD_PROFILE")
		case os.Getenv("ALIBABA_CLOUD_PROFILE") != "":
			profileName = os.Getenv("ALIBABA_CLOUD_PROFILE")
		case os.Getenv("ALICLOUD_PROFILE") != "":
			profileName = os.Getenv("ALICLOUD_PROFILE")
		}
	}
	p, _ := c.GetProfile(profileName)
	p.OverwriteWithFlags(ctx)
	return p
}

func (c *Configuration) PutProfile(profile Profile) {
	for i, p := range c.Profiles {
		if p.Name == profile.Name {
			c.Profiles[i] = profile
			return
		}
	}
	c.Profiles = append(c.Profiles, profile)
}

func LoadCurrentProfile() (Profile, error) {
	return LoadProfile(GetConfigPath()+"/"+configFile, "")
}

func LoadProfile(path string, name string) (Profile, error) {
	var p Profile
	config, err := hookLoadConfiguration(LoadConfiguration)(path)
	if err != nil {
		return p, fmt.Errorf("init config failed %v", err)
	}
	if name == "" {
		name = config.CurrentProfile
	}
	p, ok := config.GetProfile(name)
	p.parent = config
	if !ok {
		return p, fmt.Errorf("unknown profile %s, run configure to check", name)
	}
	return p, nil
}

func LoadProfileWithContext(ctx *cli.Context) (profile Profile, err error) {
	if os.Getenv("ALIBABACLOUD_IGNORE_PROFILE") == "TRUE" {
		profile = NewProfile("default")
		profile.RegionId = "cn-hangzhou"
	} else {
		var currentPath string
		if path, ok := ConfigurePathFlag(ctx.Flags()).GetValue(); ok {
			currentPath = path
		} else {
			currentPath = GetConfigPath() + "/" + configFile
		}
		if name, ok := ProfileFlag(ctx.Flags()).GetValue(); ok {
			profile, err = LoadProfile(currentPath, name)
		} else {
			profile, err = LoadProfile(currentPath, "")
		}
		if err != nil {
			return
		}
	}

	//Load from flags
	profile.OverwriteWithFlags(ctx)
	err = profile.Validate()
	return
}

func LoadConfiguration(path string) (conf *Configuration, err error) {
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		conf, err = MigrateLegacyConfiguration()
		if err != nil {
			return
		}

		if conf != nil {
			err = SaveConfiguration(conf)
			if err != nil {
				err = fmt.Errorf("save failed %v", err)
				return
			}
			return
		}
		conf = NewConfiguration()
		return
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("reading config from '%s' failed %v", path, err)
		return
	}

	conf, err = NewConfigFromBytes(bytes)
	return
}

func SaveConfiguration(config *Configuration) (err error) {
	// fmt.Printf("conf %v\n", config)
	bytes, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return
	}
	path := GetConfigPath() + "/" + configFile
	err = ioutil.WriteFile(path, bytes, 0600)
	return
}

func NewConfigFromBytes(bytes []byte) (conf *Configuration, err error) {
	conf = NewConfiguration()
	err = json.Unmarshal(bytes, conf)
	return
}

func GetConfigPath() string {
	path := hookGetHomePath(GetHomePath)() + configPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			panic(err)
		}
	}
	return path
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
