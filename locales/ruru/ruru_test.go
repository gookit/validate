package ruru

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

	Register(v)

	v.AddRule("age", "max", 1)

	is.False(v.Validate())
	is.Equal(v.Errors.One(), "Максимальное значение age равно 1")
}

func TestRegisterGlobal(t *testing.T) {
	RegisterGlobal()

	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age": 23,
	})

	v.AddRule("age", "max", 1)
	is.False(v.Validate())
	is.Equal(v.Errors.One(), "Максимальное значение age равно 1")
}
