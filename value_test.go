package validate_test

import (
	"testing"

	"github.com/gookit/validate"
	"github.com/stretchr/testify/assert"
)

func TestVal_basic(t *testing.T) {
	err := validate.Val(nil, "required")
	assert.Error(t, err)
	assert.Equal(t, "input is required and not empty", err.Error())

	err = validate.Val(23, "required|min:23")
	assert.NoError(t, err)

	err = validate.Var(23, "")
	assert.NoError(t, err)

	err = validate.Val(23, "||")
	assert.NoError(t, err)

	err = validate.Val(23, "required||min:23")
	assert.NoError(t, err)

	err = validate.Val(22, "required|min:23")
	assert.Error(t, err)
	assert.Equal(t, "input min value is 23", err.Error())
}

func TestVal_regexp(t *testing.T) {
	err := validate.Val("inhere", `required|regexp:\w+`)
	assert.NoError(t, err)

	err = validate.Val("inhere", `required|regexp:\w{12,}`)
	assert.Error(t, err)
	assert.Equal(t, `input must be match pattern \w{12,}`, err.Error())
}

func TestVal_enum(t *testing.T) {
	err := validate.Val("php", "required|in:go,php")
	assert.NoError(t, err)

	err = validate.Val("java", "required|not_in:go,php")
	assert.NoError(t, err)

	err = validate.Var("java", "required|in:go,php")
	assert.Error(t, err)
	assert.Equal(t, "input value must be in the enum [go php]", err.Error())
}
