package locales

import (
	"github.com/gookit/validate"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegister(t *testing.T) {
	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age": 23,
	})

	ok := Register(v, "zh-CN")
	is.True(ok)
	is.False(Register(v, "not-exist"))
}