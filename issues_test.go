package validate

import (
	"fmt"
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

	Config(func(opt *GlobalOption) {
		opt.FieldTag = ""
	})
	v = New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("Avatar must be an valid full URL address", v.Errors.One())

	ResetOption()
	v = New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("avatar must be an valid full URL address", v.Errors.One())
}

// https://github.com/gookit/validate/issues/22
func TestIssues22(t *testing.T) {
	type userInfo0 struct {
		Nickname string `validate:"minLen:6" message:"OO! nickname min len is 6"`
		Avatar   string `validate:"maxLen:6" message:"OO! avatar max len is %d"`
	}

	is := assert.New(t)
	u0 := &userInfo0{
		Nickname: "tom",
		Avatar:   "https://github.com/gookit/validate/issues/22",
	}
	v := Struct(u0)
	is.False(v.Validate())
	is.Equal("OO! nickname min len is 6", v.Errors.FieldOne("Nickname"))
	u0 = &userInfo0{
		Nickname: "inhere",
		Avatar:   "some url",
	}
	v = Struct(u0)
	is.False(v.Validate())
	is.Equal("OO! avatar max len is 6", v.Errors.FieldOne("Avatar"))

	// multi messages
	type userInfo1 struct {
		Nickname string `validate:"required|minLen:6" message:"required:OO! nickname cannot be empty!|minLen:OO! nickname min len is %d"`
	}

	u1 := &userInfo1{Nickname: ""}
	v = Struct(u1)
	is.False(v.Validate())
	is.Equal("OO! nickname cannot be empty!", v.Errors.FieldOne("Nickname"))

	u1 = &userInfo1{Nickname: "tom"}
	v = Struct(u1)
	is.False(v.Validate())
	is.Equal("OO! nickname min len is 6", v.Errors.FieldOne("Nickname"))
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

	// v.RegisterType(func() {})

	// use custom validator
	v := New(M{
		"age": s1,
	})
	v.AddValidator("checkAge", func(val interface{}, ints ...int) bool {
		return Enum(int32(val.(STATUS)), ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	assert.True(t, v.Validate())

	// TODO refer https://golang.org/src/database/sql/driver/types.go?s=1210:1293#L29
	v = New(M{
		"age": s1,
	})
	v.StringRules(MS{
		"age": "required|in:1,2,3,4",
	})

	assert.NotContains(t, []int{1, 2, 3, 4}, s1)

	rv := reflect.ValueOf(s1)
	// iv := reflect.New()

	// sc := rv.Interface()
	// fmt.Println(rv.Type().Kind(), sc.(int32))
	fmt.Println(rv.Type().Kind())

	dump.Println(Enum(s1, []int{1, 2, 3, 4}), Enum(int32(s1), []int{1, 2, 3, 4}))

	assert.True(t, v.Validate())
	dump.Println(v.Errors)

	type someMode string
	var m1 someMode = "abc"
	v = New(M{
		"mode": m1,
	})
	v.StringRules(MS{
		"mode": "required|in:abc,def",
	})
	assert.True(t, v.Validate())

	dump.Println(v.Errors)
}

// https://github.com/gookit/validate/issues/60
func TestIssues60(t *testing.T) {
	is := assert.New(t)
	m := map[string]interface{}{
		"title": "1",
	}

	v := Map(m)
	v.StringRule("title", "in:2,3")
	v.AddMessages(map[string]string{
		"in": "自定义错误",
	})

	is.False(v.Validate())
	is.Equal("自定义错误", v.Errors.One())
}

// https://github.com/gookit/validate/issues/64
func TestPtrFieldValidation(t *testing.T) {

	type Foo struct {
		Name *string `validate:"in:henry,jim"`
	}

	name := "henry"
	v := New(&Foo{Name: &name})
	assert.True(t, v.Validate())

	name = "fish"
	valid := New(&Foo{Name: &name})
	assert.False(t, valid.Validate())
}

// https://github.com/gookit/validate/issues/58
func TestStructNested(t *testing.T) {

	type Org struct {
		Company string `validate:"in:A,B,C,D"`
	}

	type Info struct {
		Email string `validate:"email"  filter:"trim|lower"`
		Age   *int   `validate:"in:1,2,3,4"`
	}

	// anonymous struct nested
	type User struct {
		Name string `validate:"required|string" filter:"trim|lower"`
		*Info
		Org
		Sex string `validate:"string"`
	}

	//  non-anonymous struct nested
	type User2 struct {
		Name string `validate:"required|string" filter:"trim|lower"`
		In   Info
		Sex  string `validate:"string"`
	}

	//  anonymous field test
	age := 3
	u := &User{
		Name: "fish",
		Info: &Info{
			Email: "fish_yww@163.com",
			Age:   &age,
		},
		Org: Org{Company: "E"},
		Sex: "male",
	}
	//  anonymous field test
	v := Struct(u)
	if v.Validate() {
		assert.True(t, v.Validate())
	} else {
		// Print error msg,verify valid
		fmt.Println(v.Errors)
		assert.False(t, v.Validate())
	}
	//  non-anonymous field test
	age = 3
	user2 := &User2{
		Name: "fish",
		In: Info{
			Email: "fish_yww@163.com",
			Age:   &age,
		},
		Sex: "male",
	}

	v2 := Struct(user2)
	if v2.Validate() {
		assert.True(t, v2.Validate())
	} else {
		// Print error msg,verify valid
		fmt.Printf("%v\n", v2.Errors)
		assert.False(t, v2.Validate())
	}
}
