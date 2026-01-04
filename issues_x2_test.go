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
