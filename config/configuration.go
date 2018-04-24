/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"io/ioutil"
	"os"
	"runtime"
)

const (
	configPath               = "/.aliyun"
	configFile               = "config.json"
	DefaultConfigProfileName = "default"
)

type Configuration struct {
	CurrentProfile string    `json:"current"`
	Profiles       []Profile `json:"profiles"`
	MetaPath	   string 	 `json:"meta_path"`
	//Plugins 		[]Plugin `json:"plugin"`
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
	profileName := ProfileFlag.GetStringOrDefault(c.CurrentProfile)
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
	return LoadProfile("")
}

func LoadProfile(name string) (Profile, error) {
	var p Profile
	config, err := LoadConfiguration()
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
	if name, ok := ProfileFlag.GetValue(); ok {
		profile, err = LoadProfile(name)
	} else {
		profile, err = LoadProfile("")
	}
	if err != nil {
		return
	}
	profile.OverwriteWithFlags(ctx)
	return
}

func LoadConfiguration() (Configuration, error) {
	path := GetConfigPath() + "/" + configFile
	if _, err := os.Stat(path); os.IsNotExist(err) {
		lc := MigrateLegacyConfiguration()
		if lc != nil {
			err := SaveConfiguration(*lc)
			if err != nil {
				return *lc, fmt.Errorf("save failed %v", err)
			}
			return *lc, nil
		} else {
			return NewConfiguration(), nil
		}
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return NewConfiguration(), fmt.Errorf("reading config from '%s' failed %v", path, err)
	}

	return NewConfigFromBytes(bytes)
}

func SaveConfiguration(config Configuration) error {
	// fmt.Printf("conf %v\n", config)
	bytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	path := GetConfigPath() + "/" + configFile
	err = ioutil.WriteFile(path, bytes, 0755)
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
	path := GetHomePath() + configPath
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
