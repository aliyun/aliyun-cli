// Copyright 1999-2019 Alibaba Group Holding Limited
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
	"io"
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

func NewConfiguration() Configuration {
	return Configuration{
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

func LoadCurrentProfile(w io.Writer) (Profile, error) {
	return LoadProfile(GetConfigPath()+"/"+configFile, w, "")
}

func LoadProfile(path string, w io.Writer, name string) (Profile, error) {
	var p Profile
	config, err := hookLoadConfiguration(LoadConfiguration)(path, w)
	if err != nil {
		return p, fmt.Errorf("init config failed %v", err)
	}
	if name == "" {
		name = config.CurrentProfile
	}
	p, ok := config.GetProfile(name)
	p.parent = &config
	if !ok {
		return p, fmt.Errorf("unknown profile %s, run configure to check", name)
	}
	return p, nil
}

func LoadProfileWithContext(ctx *cli.Context) (profile Profile, err error) {
	var currentPath string
	if path, ok := ConfigurePathFlag(ctx.Flags()).GetValue(); ok {
		currentPath = path
	} else {
		currentPath = GetConfigPath() + "/" + configFile
	}
	if name, ok := ProfileFlag(ctx.Flags()).GetValue(); ok {
		profile, err = LoadProfile(currentPath, ctx.Writer(), name)

	} else {
		profile, err = LoadProfile(currentPath, ctx.Writer(), "")
	}
	if err != nil {
		return
	}
	//Load from flags
	profile.OverwriteWithFlags(ctx)
	err = profile.Validate()
	return
}

func LoadConfiguration(path string, w io.Writer) (Configuration, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		lc := MigrateLegacyConfiguration(w)
		if lc != nil {
			err := SaveConfiguration(*lc)
			if err != nil {
				return *lc, fmt.Errorf("save failed %v", err)
			}
			return *lc, nil
		}
		return NewConfiguration(), nil
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return NewConfiguration(), fmt.Errorf("reading config from '%s' failed %v", path, err)
	}

	return NewConfigFromBytes(bytes)
}

func SaveConfiguration(config Configuration) error {
	// fmt.Printf("conf %v\n", config)
	bytes, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	path := GetConfigPath() + "/" + configFile
	err = ioutil.WriteFile(path, bytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

func NewConfigFromBytes(bytes []byte) (Configuration, error) {
	var conf Configuration
	err := json.Unmarshal(bytes, &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
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
