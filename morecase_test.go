package validate_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/validate"
)

func TestValidation_custom_type(t *testing.T) {
	type someMode string
	var m1 someMode = "abc"

	v := validate.New(validate.M{
		"mode": m1,
	})
	v.StringRules(validate.MS{
		"mode": "required|in:abc,def",
	})
	assert.True(t, v.Validate())

	dump.Println(
		reflect.ValueOf(m1).Kind().String(),
		validate.Enum(m1, []string{"abc", "def"}),
	)
}

// func TestValidation_custom_required1(t *testing.T) {
// 	type Data struct {
// 		Name string `validate:"required"`
// 		Age  int    `validate:"required" message:"age is required"`
// 	}
//
// 	v := validate.New(&Data{
// 		Name: "tom",
// 		Age:  0,
// 	})
// 	v.AddValidator("required", func(val any) bool {
// 		dump.V(val)
// 		// do something ...
// 		return false
// 	})
//
// 	err := v.ValidateE()
// 	assert.Error(t, err)
// 	assert.Equal(t, "age is required", err.One())
// }

func TestValidation_custom_required2(t *testing.T) {
	type Data struct {
		Age  int    `validate:"required_custom" message:"age is required"`
		Name string `validate:"required"`
	}

	v := validate.New(&Data{
		Name: "tom",
		Age:  0,
	})

	buf := new(bytes.Buffer)
	v.AddValidator("required_custom", func(val any) bool {
		buf.WriteString("value:")
		buf.WriteString(fmt.Sprint(val))
		return false
	})

	ok := v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "age is required", v.Errors.One())
	assert.Equal(t, "value:0", buf.String())
}
