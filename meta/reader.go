package meta

import (
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
		return fmt.Errorf("unmarshal %s failed %v", path, err)
	}
	return nil
}