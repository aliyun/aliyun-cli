/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package meta

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
)

type Reader interface {
	ReadFrom(path string) ([]byte, error)
}

func ReadYamlFrom(r Reader, path string, v interface{}) error {
	buf, err := r.ReadFrom(path)
	if err != nil {
		return fmt.Errorf("read yaml from %s failed %v", path, v)
	}
	err = yaml.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("unmarshal yaml %s failed %v", path, err)
	}
	return nil
}

func ReadJsonFrom(r Reader, path string, v interface{}) error {
	buf, err := r.ReadFrom(path)
	if err != nil {
		return fmt.Errorf("read json from %s failed %v", path, v)
	}
	err = json.Unmarshal(buf, v)
	if err != nil {
		return fmt.Errorf("unmarshal json %s failed %v", path, err)
	}
	return nil
}

type FileReader struct {
	//TODO GA-1
}