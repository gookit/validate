package zhtw

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/validate"
)

func TestRegister(t *testing.T) {
	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age":  23,
		"name": "inhere",
	})

	Register(v)

	v.AddRule("name", "min_len", 7)

	is.False(v.Validate())
	is.Equal(v.Errors.One(), "name 的最小長度是 7")
}

func TestRegisterGlobal(t *testing.T) {
	old := validate.CopyGlobalMessages()
	defer func() {
		validate.SetBuiltinMessages(old)
	}()

	RegisterGlobal()

	is := assert.New(t)
	v := validate.Map(map[string]interface{}{
		"age":  23,
		"name": "inhere",
	})

	v.AddRule("name", "min_len", 7)

	is.False(v.Validate())
	is.Equal(v.Errors.One(), "name 的最小長度是 7")
}
