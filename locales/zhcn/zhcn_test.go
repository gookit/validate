package zhcn

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

	Register(v)

	v.AddRule("age", "max", 1)

	is.False(v.Validate())
	is.Equal(v.Errors.One(), "age 的最大值是 1")
}

func TestRegisterGlobal(t *testing.T) {
	old := validate.CopyGlobalMessages()
	defer func() {
		validate.SetBuiltinMessages(old)
	}()

	RegisterGlobal()

	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age": 23,
	})

	v.AddRule("age", "max", 1)
	is.False(v.Validate())
	is.Equal(v.Errors.One(), "age 的最大值是 1")
}
