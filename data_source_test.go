package validate

import (
	"fmt"
	"mime/multipart"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/gookit/goutil/dump"
	"github.com/stretchr/testify/assert"
)

func TestData(t *testing.T) {
	is := assert.New(t)
	// MapData
	d := FromMap(M{
		"age": 45,
	})

	nval, err := d.Set("name", "inhere")
	is.Nil(err)
	is.Equal("inhere", nval)

	val, ok := d.Get("name")
	is.True(ok)
	is.Equal("inhere", val)
	is.Nil(d.BindJSON(nil))

	// mp := map[string]interface{}{"age": "45"}
	// d = FromMap(&mp)

	// StructData
	sd, err := FromStruct(&UserForm{Name: "abc"})
	is.Nil(err)
	val, ok = sd.Get("name")
	is.True(ok)
	is.Equal("abc", val)

	// 上面的 &UserForm 必须使用地址，下面的set才能成功
	nval, err = sd.Set("name", "def")
	is.Nil(err)
	is.Equal("def", nval)
	val, ok = sd.Get("name")
	is.True(ok)
	is.Equal("def", val)

	_, err = sd.Set("notExist", "val")
	is.Error(err)
}

func TestFormData(t *testing.T) {
	is := assert.New(t)
	d := FromURLValues(url.Values{
		"name":   {"inhere"},
		"age":    {"30"},
		"notify": {"true"},
		"money":  {"23.4"},
	})

	is.True(d.Has("notify"))
	is.True(d.HasField("notify"))
	is.False(d.Has("not-exist"))
	is.False(d.HasFile("file"))
	is.False(d.HasField("file"))
	is.True(d.Bool("notify"))
	is.False(d.Bool("not-exist"))
	is.Equal(30, d.Int("age"))
	is.Equal([]string{"30"}, d.Strings("age"))
	is.Equal(int64(30), d.Int64("age"))
	is.Equal(int64(0), d.Int64("not-exist"))
	is.Equal(0, d.Int("not-exist"))
	is.Equal(23.4, d.Float("money"))
	is.Equal(float64(0), d.Float("not-exist"))
	is.Equal("inhere", d.String("name"))
	is.Equal("age=30&money=23.4&name=inhere&notify=true", d.Encode())

	val, exist, zero := d.TryGet("name")
	is.True(exist)
	is.False(zero)
	is.Equal("inhere", val)

	val, exist = d.Get("name")
	is.True(exist)
	is.Equal("inhere", val)

	nval, err := d.Set("newKey", "strVal")
	is.NoError(err)
	is.Equal("strVal", nval)
	is.Equal("strVal", d.String("newKey"))
	_, err = d.Set("newInt", 23)
	is.NoError(err)
	is.Equal(23, d.Int("newInt"))
	_, err = d.Set("invalid", []int{2})
	is.Error(err)
	is.Equal("set value failure for field: invalid", err.Error())

	// form
	d.Add("newKey1", "new val1")
	is.NotEmpty(d.Src())
	is.Equal("new val1", d.String("newKey1"))
	d.Del("newKey1")
	is.Equal("", d.String("newKey1"))
	d.AddValues(url.Values{
		"newKey2": {"v2"},
		"newKey3": {"v3"},
	})
	is.Equal("v3", d.String("newKey3"))

	// file
	d.AddFile("file", &multipart.FileHeader{Filename: "test.txt"})
	is.True(d.Has("file"))
	is.True(d.HasFile("file"))
	is.NotEmpty(d.GetFile("file"))
	d.DelFile("file")
	is.False(d.HasFile("file"))
}

func TestStructData_Create(t *testing.T) {
	is := assert.New(t)
	_, err := FromStruct(time.Now())
	is.Error(err)
	_, err = FromStruct("invalid")
	is.Error(err)

	u := &UserForm{
		Name:      "new name",
		Status:    3,
		UpdateAt:  time.Now(),
		protected: "text",
		Extra: []ExtraInfo{
			{"xxx", 2},
		},
	}

	d, err := FromStruct(u)
	is.Nil(err)

	v := New(d, "test")
	is.Equal("test", v.Scene())

	// create validation
	v = d.Create(fmt.Errorf("a error"))
	is.False(v.Validate())
	is.Equal("a error", v.Errors.One())

	d.ValidateTag = ""
	v = d.Validation()
	is.True(v.Validate())

	// get field value
	str, ok := d.Get("Name")
	is.True(ok)
	is.Equal("new name", str)

	str, ok = d.Get("Name.NotExist")
	is.False(ok)
	is.Nil(str)

	iVal, ok := d.Get("Extra.0.Status1")
	is.True(ok)
	is.Equal(2, iVal)

	// not exist
	ret, ok := d.Get("NotExist")
	is.False(ok)
	is.Nil(ret)

	ret, ok = d.Get("NotExist.SubKey")
	is.False(ok)
	is.Nil(ret)

	ret, ok = d.Get("Extra.0.NotExist")
	is.False(ok)
	is.Nil(ret)

	// set value
	_, err = d.Set("protected", "new text")
	is.Error(err)
	nval, err := d.Set("Name", "inhere")
	is.Nil(err)
	is.Equal("inhere", nval)
	str, ok = d.Get("Name")
	is.True(ok)
	is.Equal("inhere", str)
}

func TestStructData_Set(t *testing.T) {
	is := assert.New(t)
	u := &UserForm{
		Name:      "new name",
		Status:    3,
		UpdateAt:  time.Now(),
		protected: "text",
		Extra: []ExtraInfo{
			{"xxx", 2},
		},
	}

	d, err := FromStruct(u)
	is.Nil(err)

	v := d.Validation()
	is.True(v.Errors.Empty())

	dump.P(d.fieldNames)

	_, err = d.Set("Extra.0.Github", "new url")
	is.NoError(err)
	val, ok := d.Get("Extra.0.Github")
	is.True(ok)
	is.Equal("new url", val)
}

func TestStructData_Get_ptrVal(t *testing.T) {
	type Struct1 struct {
		Name string `validate:"required"`
		Age  *int   `json:"age" validate:"required"`
	}

	age := 0
	st := &Struct1{
		Name: "tom",
		Age:  &age,
	}

	assert.False(t, reflect.ValueOf(&age).IsZero())

	d, err := FromStruct(st)
	assert.NoError(t, err)
	assert.NotEmpty(t, d.Src())

	val, ok := d.Get("Age")
	assert.True(t, ok)
	assert.Equal(t, "*int", fmt.Sprintf("%T", val))
	assert.Equal(t, 0, *val.(*int))
}

func privateFieldValidation(t *testing.T, validatePrivate, expect bool) {
	type foo struct {
		Field1 int `validate:"required|min:5|max:20" message:"Field1 outside of range"`
	}
	type bar struct {
		foo
		Field2 int `validate:"required|int" message:"Field2 outside of range"`
	}

	fooInt := 4
	barInt := 25

	myFoo := foo{Field1: fooInt}
	barz := &bar{
		foo:    myFoo,
		Field2: barInt,
	}

	Config(func(opt *GlobalOption) {
		opt.ValidatePrivateFields = validatePrivate
	})

	v := Struct(barz)
	v.Validate()

	assert.Equal(t, v.hasError, expect)
}

func TestValidatePrivateFieldsWhenTrue(t *testing.T) {
	privateFieldValidation(true, true, t)
}

func TestValidatePrivateFieldsWhenFalse(t *testing.T) {
	privateFieldValidation(false, false, t)
}
