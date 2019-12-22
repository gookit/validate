package validate

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/stretchr/testify/assert"
)

// https://github.com/gookit/validate/issues/19
func TestIssues19(t *testing.T) {
	is := assert.New(t)

	// use tag name: country_code
	type smsReq struct {
		CountryCode string `json:"country_code" validate:"required" filter:"trim|lower"`
		Phone       string `json:"phone" validate:"required" filter:"trim"`
		Type        string `json:"type" validate:"required|in:register,forget_password,set_pay_password,reset_pay_password,reset_password" filter:"trim"`
	}

	req := &smsReq{
		" ABcd   ", "13677778888  ", "register",
	}

	v := New(req)
	is.True(v.Validate())
	sd := v.SafeData()
	is.Equal("abcd", sd["CountryCode"])
	is.Equal("13677778888", sd["Phone"])

	// Notice: since 1.2, filtered value will update to struct
	// err := v.BindSafeData(req)
	// is.NoError(err)
	is.Equal("abcd", req.CountryCode)
	is.Equal("13677778888", req.Phone)

	// use tag name: countrycode
	type smsReq1 struct {
		// CountryCode string `json:"countryCode" validate:"required" filter:"trim|lower"`
		CountryCode string `json:"countrycode" validate:"required" filter:"trim|lower"`
		Phone       string `json:"phone" validate:"required" filter:"trim"`
		Type        string `json:"type" validate:"required|in:register,forget_password,set_pay_password,reset_pay_password,reset_password" filter:"trim"`
	}

	req1 := &smsReq1{
		" ABcd   ", "13677778888  ", "register",
	}

	v = New(req1)
	is.True(v.Validate())
	sd = v.SafeData()
	is.Equal("abcd", sd["CountryCode"])

	is.Equal("abcd", req1.CountryCode)
	is.Equal("13677778888", req1.Phone)
}

// https://github.com/gookit/validate/issues/20
func TestIssues20(t *testing.T) {
	is := assert.New(t)
	type setProfileReq struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|url" filter:"trim"`
	}

	req := &setProfileReq{"123nickname111", "123"}
	v := New(req)
	is.True(v.Validate())

	type setProfileReq1 struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|fullUrl" filter:"trim"`
	}
	req1 := &setProfileReq1{"123nickname111", "123"}

	v = New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("Avatar must be an valid full URL address", v.Errors.One())
}

// https://github.com/gookit/validate/issues/30
func TestIssues30(t *testing.T) {
	v := JSON(`{
   "cost_type": 10
}`)

	v.StringRule("cost_type", "str_num")

	assert.True(t, v.Validate())
	assert.Len(t, v.Errors, 0)
}

// https://github.com/gookit/validate/issues/34
func TestIssues34(t *testing.T) {
	type STATUS int32
	var s1 STATUS = 1

	rv := reflect.ValueOf(s1)
	dump.Println(rv.Type().Kind())

	// use custom validator
	v := New(M{
		"age": s1,
	})
	v.StringRules(MS{
		// "age": "required|in:1,2,3,4",
		"age": "enum_int:1,2,3,4",
	})
	v.AddValidator("inIntegers", func(val interface{}, ints ...int) bool {
		return Contains(ints, val)
	})
	v.Validate()
	dump.Println(v.Errors)

	return
	v = New(M{
		"age": s1,
	})
	v.AddValidator("enum_int", func(val int, ints ...int) bool {
		return Enum(val, ints)
	})
	v.Validate()
	dump.Println(v.Errors)

	v = New(M{
		"age": s1,
	})
	v.AddRule("age", "contains", []STATUS{1, 2, 3, 4})
	v.Validate()
	dump.Println(v.Errors)

	type someMode string
	var m1 someMode = "abc"
	v = New(M{
		"mode": m1,
	})
	v.StringRules(MS{
		"mode": "required|in:abc,def",
	})
	v.Validate()
	dump.Println(v.Errors)
}

type issues36Form struct{
	Name string `form:"username" json:"name" validate:"required|minLen:7"`
	Email string `form:"email" json:"email" validate:"email"`
	Age int `form:"age" validate:"required|int|min:18|max:150" json:"age"`
}

func (f issues36Form) Messages() map[string]string {
	return MS{
		"required": "{field}不能为空",
		"Name.minLen":"用户名最少7位",
		"Name.required": "用户名不能为空",
		"Email.email":"邮箱格式不正确",
		"Age.min":"年龄最少18岁",
		"Age.max":"年龄最大150岁",
	}
}

func (f issues36Form) Translates() map[string]string {
	return MS{
		"Name": "用户名",
		"Email": "邮箱",
		"Age":"年龄",
	}
}

// https://github.com/gookit/validate/issues/36
func TestIssues36(t *testing.T) {
	f := issues36Form{Age: 10, Name: "i am tom", Email: "adc@xx.com"}

	v := Struct(&f)
	ok := v.Validate()

	assert.False(t, ok)
	assert.Equal(t, v.Errors.One(), "年龄最少18岁")
	assert.Contains(t, v.Errors.String(), "年龄最少18岁")
}
