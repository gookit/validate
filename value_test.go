package validate_test

import (
	"testing"

	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/validate"
)

func TestVal_basic(t *testing.T) {
	err := validate.Val(nil, "required")
	assert.Error(t, err)
	assert.Equal(t, "input is required to not be empty", err.Error())

	err = validate.Val(23, "required|min:23")
	assert.NoError(t, err)

	err = validate.Var(23, "")
	assert.NoError(t, err)

	err = validate.Val(23, "||")
	assert.NoError(t, err)

	err = validate.Val(23, "required|:|min:23")
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
	assert.Equal(t, `input must match pattern \w{12,}`, err.Error())
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

func TestVal_indirect(t *testing.T) {
	foobar := "foobar"
	err := validate.Val(&foobar, "required|string")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|starts_with:foo")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|ends_with:bar")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|contains:oba")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|eq:foobar")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|ne:foo")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|len:6")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|min_len:5")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|max_len:6")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|regex:^foo")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|ascii")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|alpha")
	assert.NoError(t, err)

	err = validate.Val(&foobar, "required|alpha_num")
	assert.NoError(t, err)

	fooBar := "foo bar"
	err = validate.Val(&fooBar, "required|hasWhitespace")
	assert.NoError(t, err)

	email := "hello@email.com"
	err = validate.Val(&email, "required|email")
	assert.NoError(t, err)

	url := "https://github.com"
	err = validate.Val(&url, "required|url")
	assert.NoError(t, err)

	ip := "1.1.1.1"
	err = validate.Val(&ip, "required|ip")
	assert.NoError(t, err)

	number := 10
	err = validate.Val(&number, "required|int|number")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|int_eq:10")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|uint")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|min:0")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|max:30")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|between:0,30")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|gt:0")
	assert.NoError(t, err)

	err = validate.Val(&number, "required|lt:30")
	assert.NoError(t, err)

	float := 0.1
	err = validate.Val(&float, "required|float")
	assert.NoError(t, err)

	boolVal := true
	err = validate.Val(&boolVal, "required|bool")
	assert.NoError(t, err)

	dateVal := "2019-01-01"
	err = validate.Val(&dateVal, "required|date")
	assert.NoError(t, err)

	err = validate.Val(&dateVal, "required|gt_date:2006-01-02")
	assert.NoError(t, err)

	err = validate.Val(&dateVal, "required|lt_date:2020-01-02")
	assert.NoError(t, err)

	emptyStr := ""
	err = validate.Val(&emptyStr, "required|empty")
	assert.NoError(t, err)
}
