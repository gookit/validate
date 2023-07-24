package ruru

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/validate"
)

func TestRegister(t *testing.T) {
	is := assert.New(t)
	v := validate.Map(map[string]any{
		"age": 23,
	})

	Register(v)

	v.AddRule("age", "max", 1)

	is.False(v.Validate())
	is.Equal(v.Errors.One(), "Максимальное значение age равно 1")
}

func TestRegisterGlobal(t *testing.T) {
	old := validate.CopyGlobalMessages()
	defer func() {
		validate.SetBuiltinMessages(old)
	}()

	RegisterGlobal()

	is := assert.New(t)
	v := validate.Map(map[string]any{
		"age": 23,
	})

	v.AddRule("age", "max", 1)
	is.False(v.Validate())
	is.Equal(v.Errors.One(), "Максимальное значение age равно 1")
}
