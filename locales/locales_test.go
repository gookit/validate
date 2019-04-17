package locales

import (
	"testing"

	"github.com/gookit/validate"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age": 23,
	})

	ok := Register(v, "zh-CN")
	is.True(ok)
	is.False(Register(v, "not-exist"))

	v.AddRule("age", "max", 1)
	v.AddTranslates(validate.MS{
		"age": "年龄",
	})
	v.Validate()
	is.Equal(v.Errors.One(), "年龄 的最大值是 1")
}
