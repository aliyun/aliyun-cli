package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/stretchr/testify/assert"

	"testing"
)

type write struct {

}

func (w *write) Write(p []byte)(nn int, err error) {
	return len(p), nil
}

func Test_main(t *testing.T) {
	w := &write{}
	ctx := cli.NewCommandContext(w)
	cmd := config.NewConfigureSetCommand()
   ctx.EnterCommand(cmd)
	c := Commando{
		library: NewLibrary(w, "test"),
			}
	err := c.main(ctx, nil)
	assert.Nil(t, err)

	//args := []string{"arg1"}
	//os.Setenv("REGION", "cn-hangzhou")
	//err = c.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Contains(t, err.Error(),"'arg1' is not a valid command or product. See `aliyun help`.")

	//args = []string{"arg1", "arg2"}
	//err = c.main(ctx, args)
	//assert.NotNil(t, err)
	//assert.Contains(t, err.Error(),"'arg1' is not a valid command or product. See `aliyun help`.")
}

func TestFormatJson(t *testing.T) {
	out := "test format json"
	out = FormatJson(out)
	assert.Equal(t, "test format json", out)

	out = `{"requestid":"test","name":"json"}`
	out = FormatJson(out)
	assert.Equal(t, "{\n\t\"requestid\": \"test\",\n\t\"name\": \"json\"\n}", out)
}

