package validate_test

import (
	"mime/multipart"
	"net/url"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/validate"
)

// https://github.com/gookit/validate/issues/227
// 一个 key 包含多个上传文件时，除了第一个文件，其他文件被丢弃，导致 BindSafeData 行为非预期
func Test_Issue227(t *testing.T) {
	type UserForm struct {
		Name string
		File []*multipart.FileHeader
	}

	d := validate.FromURLValues(url.Values{
		"name": {"inhere"},
		"age":  {"30"},
	})
	// add files
	d.AddFile("File", &multipart.FileHeader{Filename: "test1.txt"}, &multipart.FileHeader{Filename: "test2.txt"})
	v := d.Create()
	v.AddRule("File", "min_len", 1)

	assert.True(t, v.Validate())
	dump.P(v.Errors)
	assert.Nil(t, v.Errors.ErrOrNil())

	u := &UserForm{}
	err := v.BindStruct(u)
	assert.NoError(t, err)
	dump.P(u)
}

// https://github.com/gookit/validate/issues/259 Embedded structs are not validated properly #259

// https://github.com/gookit/validate/issues/272 eqField对于指针类型数据无法正确校验
func Test_Issue272(t *testing.T) {
	type T272 struct {
		FieldA *string `validate:"required"`
		FieldB *string `validate:"required|eqField:FieldA"`
	}

	// test eqField
	var str = "abc"
	var str1 = "bcd"
	v := validate.Struct(&T272{
		FieldA: &str,
		FieldB: &str1,
	})
	assert.False(t, v.Validate())
	assert.Len(t, v.Errors, 1)
	assert.ErrSubMsg(t, v.Errors, "FieldB value must be equal the field FieldA")

	var str2 = "abc"
	v = validate.Struct(&T272{
		FieldA: &str,
		FieldB: &str2,
	})
	assert.True(t, v.Validate())
	assert.Nil(t, v.Errors.ErrOrNil())

	// nil value
	v = validate.Struct(&T272{
		FieldA: nil,
		FieldB: nil,
	})
	assert.False(t, v.Validate())
	assert.Len(t, v.Errors, 1)
	assert.ErrSubMsg(t, v.Errors, "FieldA is required")

}

// https://github.com/gookit/validate/issues/316
// The int validator failed to validate a number exceeds the range of int64
func Test_Issue316(t *testing.T) {
	data := []byte(`{"value": 9223372036854775807}`)

	t.Run("not use filter", func(t *testing.T) {
		dataFace, err := validate.FromJSONBytes(data)
		assert.NoErr(t, err)

		v := dataFace.Create()
		v.StringRule("value", "int")
		assert.False(t, v.Validate())
		dump.P(v.Errors)
		assert.Err(t, v.Errors.ErrOrNil())
		assert.Equal(t, "value value must be an integer", v.Errors.One())
	})

	t.Run("use filter", func(t *testing.T) {
		dataFace, err := validate.FromJSONBytes(data)
		assert.NoErr(t, err)

		v := dataFace.Create()
		v.FilterRule("value", "int64")
		v.StringRule("value", "int")
		assert.True(t, v.Validate())
		assert.Nil(t, v.Errors.ErrOrNil())
		dump.P(v.SafeData())
	})
}
