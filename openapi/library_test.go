package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
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

func TestLibrary_PrintProducts(t *testing.T) {
	writer := cli.DefaultWriter()
	library := NewLibrary(writer, "en")

	_, isexist := library.GetApi("aos", "v1.0", "describe")
	assert.False(t, isexist)

	products := library.GetProducts()
	assert.NotNil(t, products)

	product := meta.Product{
		Code: "ecs",
		Version:"v1.0",
		Name: map[string]string{"zh":"test"},
	}
	library.printProduct(product)

	library.builtinRepo.Products = []meta.Product{
		{
			Code: "ecs",
		},
	}
	library.PrintProducts()
}

func TestLibrary_PrintProductUsage(t *testing.T) {
	writer := cli.DefaultWriter()
	library := NewLibrary(writer, "en")
	content := `{"products":[{"code":"ecs","api_style":"rpc","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err := library.PrintProductUsage("aos", true)
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)

	content = `{"products":[{"code":"ecs","api_style":"restful","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err = library.PrintProductUsage("ecs", true)
	assert.Nil(t, err)
}

func TestLibrary_PrintApiUsage(t *testing.T) {
	writer := cli.DefaultWriter()
	library := NewLibrary(writer, "en")
	content := `{"products":[{"code":"ecs","api_style":"rpc","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err := library.PrintApiUsage("aos", "DescribeRegions")
	assert.Equal(t, "'aos' is not a valid command or product. See `aliyun help`.", err.Error())

	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)

	content = `{"products":[{"code":"ecs","api_style":"restful","apis":["DescribeRegions"]}]}`
	library.builtinRepo = getRepository(content)
	err = library.PrintApiUsage("ecs", "DescribeRegions")
	assert.Nil(t, err)
}

func Test_printParameters(t *testing.T){
	writer := cli.DefaultWriter()
	params := []meta.Parameter{
		{
			Hidden: true,
		},
		{
			Position: "Domain",
		},
		{
			Type: "RepeatList",
			Required: true,
		},
		{
			Required: false,
		},
		{
			SubParameters: []meta.Parameter{
				{
					Name: "test",
				},
			},
		},
	}
	printParameters(writer, params, "")
}

func getRepository(content string) (*meta.Repository){
	reader := &reader_test{
		content: content,
	}
	repository := meta.LoadRepository(reader)
	return repository
}
