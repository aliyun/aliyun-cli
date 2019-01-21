package meta

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type reader_test struct {
	content   string
}

func (r *reader_test)ReadFrom(path string) ([]byte, error){
	if path == "" || r.content == ""{
		return nil, fmt.Errorf("Please insert a valid path.")
	}
	return []byte(r.content), nil
}

func (r *reader_test)setcontent(contenth string){
	r.content = contenth
}

func TestReadYamlFrom(t *testing.T) {
	r := &reader_test{}
	path := ""
	err := ReadYamlFrom(r, path, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "read yaml from  failed <nil>", err.Error())

	api := &Api{}
	path = `{"name":"api""protocol":"http"}`
	r.content = path
	err = ReadYamlFrom(r, path, api)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal yaml {\"name\":\"api\"\"protocol\":\"http\"} failed yaml: did not find expected ',' or '}'", err.Error())

	r.content = `{"name":"api","protocol":"http"}`
	err = ReadYamlFrom(r, path, api)
	assert.Nil(t, err)
}

func TestReadJsonFrom(t *testing.T) {
	r := &reader_test{}
	path := ""
	err := ReadJsonFrom(r, path, nil)
	assert.NotNil(t, err)
	assert.Equal(t, "read json from  failed <nil>", err.Error())

	api := &Api{}
	path = `{"name":"api""protocol":"http"}`
	r.content = path
	err = ReadJsonFrom(r, path, api)
	assert.NotNil(t, err)
	assert.Equal(t, "unmarshal json {\"name\":\"api\"\"protocol\":\"http\"} failed invalid character '\"' after object key:value pair", err.Error())

	r.content = `{"name":"api","protocol":"http"}`
	err = ReadJsonFrom(r, path, api)
	assert.Nil(t, err)
}