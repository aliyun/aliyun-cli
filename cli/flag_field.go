package cli

import (
	"github.com/aliyun/aliyun-cli/i18n"
)

type Field struct {
	Key string			// if key is null, it can appear with
	Required bool
	Repeatable bool
	DefaultValue string
	Short i18n.Text

	assigned bool
	value string
	values []string
}

func (f *Field) assign(v string) {
	f.assigned = true
	f.value = v
	f.values = append(f.values, v)
}

func (f *Field) getValue() (string, bool) {
	if f.assigned {
		return f.value, true
	} else if f.DefaultValue != "" {
		return f.DefaultValue, false
	} else {
		return "", false
	}
}
