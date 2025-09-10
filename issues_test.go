package validate_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/goutil/timex"

	"github.com/gookit/validate"
	"github.com/gookit/validate/locales/zhcn"
)

func TestIssue_2(t *testing.T) {
	type Fl struct {
		A float64 `validate:"float"`
	}

	fl := Fl{123}
	v := validate.Struct(fl)
	assert.True(t, v.Validate())
	assert.Equal(t, float64(123), v.SafeVal("A"))

	val, ok := v.Raw("A")
	assert.True(t, ok)
	assert.Equal(t, float64(123), val)

	// Set value
	err := v.Set("A", float64(234))
	assert.Error(t, err)
	// field not exist
	err = v.Set("B", 234)
	assert.Error(t, err)

	// NOTICE: Must use ptr for set value
	v = validate.Struct(&fl)
	err = v.Set("A", float64(234))
	assert.Nil(t, err)

	// check new value
	val, ok = v.Raw("A")
	assert.True(t, ok)
	assert.Equal(t, float64(234), val)

	// int will convert to float
	err = v.Set("A", 23)
	assert.Nil(t, err)

	// type is error
	err = v.Set("A", "abc")
	assert.Error(t, err)
	assert.ErrMsg(t, err, `strconv.ParseFloat: parsing "abc": invalid syntax`)
}

// https://github.com/gookit/validate/issues/19
func TestIssues_19(t *testing.T) {
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

	v := validate.New(req)
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

	v = validate.New(req1)
	is.True(v.Validate())
	sd = v.SafeData()
	is.Equal("abcd", sd["CountryCode"])

	is.Equal("abcd", req1.CountryCode)
	is.Equal("13677778888", req1.Phone)
}

// https://github.com/gookit/validate/issues/20
func TestIssues_20(t *testing.T) {
	is := assert.New(t)
	type setProfileReq struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|url" filter:"trim"`
	}

	req := &setProfileReq{"123nickname111", "123"}
	v := validate.New(req)
	is.True(v.Validate())

	type setProfileReq1 struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|fullUrl" filter:"trim"`
	}
	req1 := &setProfileReq1{"123nickname111", "123"}

	validate.Config(func(opt *validate.GlobalOption) {
		opt.FieldTag = ""
	})
	v = validate.New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("Avatar must be a valid full URL address", v.Errors.One())

	validate.ResetOption()
	v = validate.New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("avatar must be a valid full URL address", v.Errors.One())
}

// https://github.com/gookit/validate/issues/22
func TestIssues_22(t *testing.T) {
	type userInfo0 struct {
		Nickname string `validate:"minLen:6" message:"OO! nickname min len is 6"`
		Avatar   string `validate:"maxLen:6" message:"OO! avatar max len is %d"`
	}

	is := assert.New(t)
	u0 := &userInfo0{
		Nickname: "tom",
		Avatar:   "https://github.com/gookit/validate/issues/22",
	}
	v := validate.Struct(u0)
	is.False(v.Validate())
	is.Equal("OO! nickname min len is 6", v.Errors.FieldOne("Nickname"))
	u0 = &userInfo0{
		Nickname: "inhere",
		Avatar:   "some url",
	}
	v = validate.Struct(u0)
	is.False(v.Validate())
	is.Equal("OO! avatar max len is 6", v.Errors.FieldOne("Avatar"))

	// multi messages
	type userInfo1 struct {
		Nickname string `validate:"required|min_len:6" message:"required:OO! nickname cannot be empty!|min_len:OO! nickname min len is %d"`
	}

	u1 := &userInfo1{Nickname: ""}
	v = validate.Struct(u1)
	is.False(v.Validate())
	is.Equal("OO! nickname cannot be empty!", v.Errors.FieldOne("Nickname"))

	u1 = &userInfo1{Nickname: "tom"}
	v = validate.Struct(u1)
	is.False(v.Validate())
	is.Equal("OO! nickname min len is 6", v.Errors.FieldOne("Nickname"))
}

// https://github.com/gookit/validate/issues/30
func TestIssues_30(t *testing.T) {
	v := validate.JSON(`{
   "cost_type": 10
}`)

	v.StringRule("cost_type", "str_num")

	assert.True(t, v.Validate())
	assert.Len(t, v.Errors, 0)
}

// https://github.com/gookit/validate/issues/34
func TestIssues_34(t *testing.T) {
	type STATUS int32
	var s1 STATUS = 1

	// use custom validator
	v := validate.New(validate.M{
		"age": s1,
	})
	v.AddValidator("checkAge", func(val any, ints ...int) bool {
		return validate.Enum(int32(val.(STATUS)), ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	assert.True(t, v.Validate())

	v = validate.New(validate.M{
		"age": s1,
	})
	v.StringRules(validate.MS{
		"age": "required|in:1,2,3,4",
	})

	assert.NotContains(t, []int{1, 2, 3, 4}, s1)
	assert.True(t, v.Validate())

	// dump.Println(validate.Enum(s1, []int{1, 2, 3, 4}), validate.Enum(int32(s1), []int{1, 2, 3, 4}))
	// dump.Println(v.Errors)

	type someMode string
	var m1 someMode = "abc"
	v = validate.New(validate.M{
		"mode": m1,
	})
	v.StringRules(validate.MS{
		"mode": "required|in:abc,def",
	})
	assert.True(t, v.Validate())
}

type issues36Form struct {
	Name  string `form:"username" json:"name" validate:"required|minLen:7"`
	Email string `form:"email" json:"email" validate:"email"`
	Age   int    `form:"age" validate:"required|int|min:18|max:150" json:"age"`
}

func (f issues36Form) Messages() map[string]string {
	return validate.MS{
		"required":      "{field}不能为空",
		"Name.minLen":   "用户名最少7位",
		"Name.required": "用户名不能为空",
		"Email.email":   "邮箱格式不正确",
		"Age.min":       "年龄最少18岁",
		"Age.max":       "年龄最大150岁",
	}
}

func (f issues36Form) Translates() map[string]string {
	return validate.MS{
		"Name":  "用户名",
		"Email": "邮箱",
		"Age":   "年龄",
	}
}

// https://github.com/gookit/validate/issues/36
func TestIssues36(t *testing.T) {
	f := issues36Form{Age: 10, Name: "i am tom", Email: "adc@xx.com"}

	v := validate.Struct(&f)
	ok := v.Validate()

	assert.False(t, ok)
	assert.Equal(t, v.Errors.One(), "年龄最少18岁")
	assert.Contains(t, v.Errors.String(), "年龄最少18岁")
}

// https://github.com/gookit/validate/issues/60
func TestIssues_60(t *testing.T) {
	is := assert.New(t)
	m := map[string]any{
		"title": "1",
	}

	v := validate.Map(m)
	v.StringRule("title", "in:2,3")
	v.AddMessages(map[string]string{
		"in": "自定义错误",
	})

	is.False(v.Validate())
	is.Equal("自定义错误", v.Errors.One())
}

// https://github.com/gookit/validate/issues/64
// see validate.Enum()
func TestPtrFieldValidation(t *testing.T) {
	type Foo struct {
		Name *string `validate:"in:henry,jim"`
	}

	name := "henry"
	v := validate.New(&Foo{Name: &name})
	assert.True(t, v.Validate())

	name = "fish"
	valid := validate.New(&Foo{Name: &name})
	assert.False(t, valid.Validate())
}

type Pet struct {
	Breed string `json:"breed" validate:"min_len:3"`
	Color string `json:"color" validate:"in:orange,black,brown"`
}
type Human struct {
	Age  int    `json:"age" validate:"gt:13"`
	Name string `json:"name" validate:"min_len:3"`
}
type Settings struct {
	*Pet   `json:",omitempty"`
	*Human `json:",omitempty"`
}

type Entity struct {
	ID       string   `json:"id" validate:"uuid4"`
	Kind     string   `json:"kind" validate:"in:pet,human"`
	Settings Settings `json:"settings"`
}

func (Entity) ConfigValidation(v *validate.Validation) {
	v.WithScenes(validate.SValues{
		"required": []string{"ID", "Kind"},
		"pet":      []string{"Settings.Pet"},
		"human":    []string{"Settings.Human"},
	})

}

func TestPtrFieldEmbeddedValid(t *testing.T) {
	// Valid case :)
	input := []byte(`{
		"id": "59fcf270-8646-4250-b4ff-d50f6121bc9d",
		"kind": "pet",
		"settings": {
		  "breed": "dog",
		  "color": "brown"
		}
	  }`)
	var entity Entity
	err := json.Unmarshal(input, &entity)
	fmt.Println(entity)
	assert.NoError(t, err)

	validate.Config(func(opt *validate.GlobalOption) {
		opt.SkipOnEmpty = false
		opt.StopOnError = false
	})

	vld := validate.Struct(&entity)
	vld.SkipOnEmpty = false
	err = vld.ValidateE("required")
	assert.Empty(t, err)
	vld.ResetResult()
	err = vld.ValidateE(entity.Kind)
	assert.Empty(t, err)

	validate.ResetOption()
}

func TestPtrFieldEmbeddedInvalid(t *testing.T) {
	// Invalid case
	input := []byte(`{
		"id": "59fcf270-8646-4250-b4ff-d50f6121bc9d",
		"kind": "pet",
		"settings": {
		}
	  }`)
	var entity Entity
	err := json.Unmarshal(input, &entity)
	fmt.Println(entity)
	assert.NoError(t, err)

	validate.Config(func(opt *validate.GlobalOption) {
		opt.SkipOnEmpty = false
		opt.StopOnError = false
	})
	vld := validate.Struct(&entity)
	err = vld.ValidateE("required")
	assert.Empty(t, err)
	vld.ResetResult()
	err = vld.ValidateE(entity.Kind)
	expected := validate.Errors{
		"breed": validate.MS{
			"min_len": "breed min length is 3",
		},
		"color": validate.MS{
			"in": "color value must be in the enum [orange black brown]",
		},
	}

	assert.Equal(t, expected, err)
	validate.ResetOption()
}

// ----- test case structs

type Org struct {
	Company string `validate:"in:A,B,C,D"`
}

type Info struct {
	Email string `validate:"email"  filter:"trim|lower"`
	Age   *int   `validate:"in:1,2,3,4"`
}

// anonymous struct nested
type User struct {
	*Info `validate:"required"`
	Org
	Name string `validate:"required|string" filter:"trim|lower"`
	Sex  string `validate:"string"`
	Time time.Time
}

// non-anonymous struct nested
type User2 struct {
	Name string `validate:"required|string" filter:"trim|lower"`
	In   Info
	Sex  string `validate:"string"`
	Time time.Time
}

type Info2 struct {
	Org
	Sub *Info
}

// gt 2 level struct nested
type User3 struct {
	In2 *Info2 `validate:"required"`
}

// https://github.com/gookit/validate/issues/58
func TestStructNested(t *testing.T) {
	// anonymous field test
	age := 3
	u := &User{
		Name: "fish",
		Info: &Info{
			Email: "fish_yww@163.com",
			Age:   &age,
		},
		Org:  Org{Company: "E"},
		Sex:  "male",
		Time: time.Now(),
	}

	// anonymous field test
	v := validate.Struct(u)
	if v.Validate() {
		assert.True(t, v.Validate())
	} else {
		// Print error msg,verify valid
		fmt.Println("--- anonymous field test\n", v.Errors)
		assert.False(t, v.Validate())
	}

	// non-anonymous field test
	age = 3
	user2 := &User2{
		Name: "fish",
		In: Info{
			Email: "fish_yww@163.com",
			Age:   &age,
		},
		Sex:  "male",
		Time: time.Now(),
	}

	v2 := validate.Struct(user2)
	if v2.Validate() {
		assert.True(t, v2.Validate())
	} else {
		// Print error msg,verify valid
		fmt.Printf("%v\n", v2.Errors)
		assert.False(t, v2.Validate())
	}
}

func TestStruct_nilPtr_field(t *testing.T) {
	u1 := &User{
		Name: "fish",
		Info: nil,
		Org:  Org{Company: "C"},
		Sex:  "male",
	}

	v := validate.Struct(u1)
	assert.False(t, v.Validate())
	assert.Contains(t, v.Errors.String(), "Info is required")
	fmt.Println(v.Errors)
}

func TestStructNested_gt2level(t *testing.T) {
	age := 3
	u := &User3{
		In2: &Info2{
			Org: Org{Company: "E"},
			Sub: &Info{
				Email: "SOME@163.com ",
				Age:   &age,
			},
		},
	}

	v := validate.Struct(u)
	ok := v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "In2.Org.Company value must be in the enum [A B C D]", v.Errors.Random())
	fmt.Println(v.Errors)

	u.In2.Org.Company = "A"
	v = validate.Struct(u)
	ok = v.Validate()
	assert.True(t, ok)
	assert.Equal(t, "some@163.com", u.In2.Sub.Email)
}

// https://github.com/gookit/validate/issues/76
func TestIssues_76(t *testing.T) {
	type CategoryReq struct {
		Name string
	}
	type Issue76 struct {
		IsPrivate  bool           `json:"is_private"`
		Categories []*CategoryReq `json:"categories" validate:"required_if:IsPrivate,false"`
	}

	v := validate.Struct(&Issue76{})
	ok := v.Validate()
	assert.False(t, ok)
	dump.Println(v.Errors)
	assert.Equal(t, "categories is required when is_private is in [false]", v.Errors.One())

	v = validate.Struct(&Issue76{Categories: []*CategoryReq{
		{Name: "book"},
	}})
	ok = v.Validate()
	assert.True(t, ok)

	// test convert to bool error
	type Issue76A struct {
		IsPrivate  bool           `json:"is_private"`
		Categories []*CategoryReq `json:"categories" validate:"required_if:IsPrivate,invalid"`
	}

	v = validate.Struct(&Issue76A{})
	ok = v.Validate()
	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/78
func TestIssue_78(t *testing.T) {
	type UserDto struct {
		Name string `validate:"required"`
		Sex  *bool  `validate:"required"`
	}

	// sex := true
	u := UserDto{
		Name: "abc",
		Sex:  nil,
	}

	// 创建 Validation 实例
	v := validate.Struct(&u)
	ok := v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "Sex is required to not be empty", v.Errors.One())

	sex := false
	u = UserDto{
		Name: "abc",
		Sex:  &sex,
	}

	v = validate.Struct(&u)
	ok = v.Validate()
	assert.True(t, ok)
}

// https://gitee.com/inhere/validate/issues/I36T2B
func TestIssues_I36T2B(t *testing.T) {
	m := map[string]any{
		"a": 0,
	}

	// 创建 Validation 实例
	v := validate.Map(m)
	v.AddRule("a", "gt", 100)

	ok := v.Validate()
	assert.True(t, ok)

	v = validate.Map(m)
	v.AddRule("a", "gt", 100).SetSkipEmpty(false)

	ok = v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "a value should be greater than 100", v.Errors.One())

	v = validate.Map(make(map[string]any))
	v.AddRule("a", "required")
	v.AddRule("a", "gt", 100)

	ok = v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "a is required to not be empty", v.Errors.One())
}

// https://gitee.com/inhere/validate/issues/I3B3AV
func TestIssues_I3B3AV(t *testing.T) {
	m := map[string]any{
		"a": 0.01,
		"b": float32(0.03),
	}

	v := validate.Map(m)
	v.AddRule("a", "gt", 0)
	v.AddRule("b", "gt", 0)

	assert.True(t, v.Validate())
}

// https://github.com/gookit/validate/issues/92
func TestIssues_92(t *testing.T) {
	m := map[string]any{
		"t": 1.1,
	}

	v := validate.Map(m)
	v.FilterRule("t", "float")
	ok := v.Validate()

	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/98
func TestIssues_98(t *testing.T) {
	// MenuActionResource 菜单动作关联资源对象
	type MenuActionResource struct {
		ID       uint64 `json:"id"`                         // 唯一标识
		ActionID uint64 `json:"action_id"`                  // 菜单动作ID
		Method   string `json:"method" validate:"required"` // 资源请求方式(支持正则)
		Path     string `json:"path" validate:"required"`   // 资源请求路径（支持/:id匹配）
	}

	// MenuActionResources 菜单动作关联资源管理列表
	type MenuActionResources []*MenuActionResource

	// MenuAction 菜单动作对象
	type MenuAction struct {
		Code      string              `json:"code" validate:"required"` // 动作编号
		Name      string              `json:"name" validate:"required"` // 动作名称
		Resources MenuActionResources `json:"resources,omitempty"`      // 资源列表
	}

	// MenuActions 菜单动作管理列表
	type MenuActions []*MenuAction
	type MenuCreateRequest struct {
		Name        string      `json:"name" validate:"required"`
		Sequence    int         `json:"sequence"`                  // 排序值
		Icon        string      `json:"icon"`                      // 菜单图标
		Router      string      `json:"router"`                    // 访问路由
		ParentID    uint64      `json:"parent_id"`                 // 父级ID
		IsShow      int         `json:"is_show" validate:"in:0,1"` // 是否显示(1:显示 0:隐藏)
		Status      int         `json:"status" validate:"in:0,1"`  // 状态(1:启用 0:禁用)
		Memo        string      `json:"memo"`                      // 备注
		MenuActions MenuActions `json:"actions,omitempty"`
	}

	req := &MenuCreateRequest{
		Name: "users",
		MenuActions: []*MenuAction{
			{
				Code: "code01",
				Name: "name01",
				Resources: []*MenuActionResource{
					{
						Method: "get",
						Path:   "/home",
					},
				},
			},
		},
	}

	v := validate.Struct(req)
	ok := v.Validate()

	dump.Println(v.Errors)
	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/103
func TestIssues_103(t *testing.T) {
	type Example struct {
		SomeID string `validate:"required"`
	}

	o := Example{}
	v := validate.Struct(o)
	v.Validate()
	// here we get something like {"SomeID": { /* ... */ }}
	m := v.Errors.All()
	// dump.Println(m)
	assert.Contains(t, m, "SomeID")
	assert.Contains(t, v.Errors.String(), "SomeID is required to not be empty")

	type Example2 struct {
		SomeID string `json:"some_id" validate:"required" `
	}

	e2 := Example2{}
	v2 := validate.Struct(e2)
	v2.Validate()
	dump.Println(v2.Errors)
	err2 := v2.Errors.String() // here we get something like {"some_id": { /* ... */ }}
	assert.Contains(t, err2, "some_id")
	assert.Contains(t, err2, "some_id is required to not be empty")
}

type Issue104A struct {
	ID int `json:"id" gorm:"primarykey" form:"id" validate:"int|required"`
}

type Issue104Demo struct {
	Issue104A
	Title string `json:"title" form:"title" validate:"required" example:"123456"` // 任务id
}

// GetScene 定义验证场景
// func (d Issue104Demo) GetScene() validate.SValues {
// 	return validate.SValues{
// 		"add":    []string{"ID", "Title"},
// 		"update": []string{"ID", "Title"},
// 	}
// }

// ConfigValidation 配置验证
// - 定义验证场景
func (d Issue104Demo) ConfigValidation(v *validate.Validation) {
	v.WithScenes(validate.SValues{
		"add":    []string{"Issue104A.ID", "Title"},
		"update": []string{"Issue104A.ID", "Title"},
	})
}

// https://github.com/gookit/validate/issues/104
func TestIssues_104(t *testing.T) {
	d := &Issue104Demo{
		Issue104A: Issue104A{
			ID: 0,
		},
		Title: "abc",
	}

	v := validate.Struct(d)
	ok := v.Validate()
	dump.Println(v.Errors)
	assert.False(t, ok)
	assert.Equal(t, "id is required to not be empty", v.Errors.One())

	v = validate.Struct(d, "add")
	ok = v.Validate()
	dump.Println(v.Errors, v.SceneFields())
	assert.False(t, ok)
	assert.Equal(t, "add", v.Scene())
	assert.Equal(t, "id is required to not be empty", v.Errors.One())

	// right
	d.Issue104A.ID = 34
	v = validate.Struct(d)
	ok = v.Validate()
	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/107
func TestIssues_107(t *testing.T) {
	taFilter := func(val any) int64 {
		if val != nil {
			// log.WithFields(log.Fields{"value": val}).Info("value should be other than nil")
			return int64(val.(float64))
		}

		// log.WithFields(log.Fields{"value": val}).Info("value should be nil")
		return -1
	}

	v := validate.Map(map[string]any{
		"tip_amount": float64(12),
	})
	v.AddFilter("tip_amount_filter", taFilter)
	// method 1:
	// v.FilterRule("tip_amount", "tip_amount_filter")
	// v.AddRule("tip_amount", "int")

	// method 2:
	v.StringRule("tip_amount", "int", "tip_amount_filter")

	assert.Equal(t, float64(12), v.RawVal("tip_amount"))

	// call validate
	assert.True(t, v.Validate())
	// dump.Println(v.FilteredData())
	dump.Println(v.SafeData())
	assert.Equal(t, int64(12), v.SafeVal("tip_amount"))
}

// https://github.com/gookit/validate/issues/111
func TestIssues_111(t *testing.T) {
	v := validate.New(map[string]any{
		"username":  "inhere",
		"password":  "h9i8tssx9153",
		"password2": "h9i8XYZ9153",
	})
	v.AddTranslates(map[string]string{
		"username":  "账号",
		"password":  "密码",
		"password2": "重复密码",
	})
	v.StringRule("username", "required")
	v.StringRule("password", "required|eq_field:password2|minLen:6|max_len:32")

	assert.False(t, v.Validate())
	dump.Println(v.Errors)
	assert.Contains(t, v.Errors.String(), "密码 ")
	assert.Contains(t, v.Errors.String(), " 重复密码")
}

// https://github.com/gookit/validate/issues/120
func TestIssues_120(t *testing.T) {
	type ThirdStruct struct {
		Val string `json:"val" validate:"required"`
	}
	type SecondStruct struct {
		Third []ThirdStruct `json:"third" validate:"slice"`
	}
	type MainStruct struct {
		Second []SecondStruct `json:"second" validate:"slice"`
	}

	v := validate.Struct(&MainStruct{
		Second: []SecondStruct{
			{
				Third: []ThirdStruct{
					{
						Val: "hello",
					},
				},
			},
		},
	})

	ok := v.Validate()
	assert.True(t, ok)
	// dump.Println(v.Errors)
}

// https://github.com/gookit/validate/issues/124
// Validate array/slice items #124
func TestIssue_124(t *testing.T) {
	m := map[string]any{
		"names": []string{"John", "Jane", "abc"},
		"address": []map[string]string{
			{"number": "1b", "country": "en"},
			{"number": "1", "country": "cz"},
		},
	}

	v := validate.Map(m)
	v.StopOnError = false

	// simple slices
	v.StringRule("names", "required|array|minLen:1")
	v.StringRule("names.*", "required|string|min_len:4")

	// slices of maps
	v.StringRule("address.*.number", "required|int")
	v.StringRule("address.*.country", "required|in:es,us")

	assert.False(t, v.Validate())
	assert.Error(t, v.Errors.ErrOrNil())

	assert.Equal(t, len(v.Errors.All()), 3)
	assert.Equal(t, "names.* min length is 4", v.Errors.Field("names.*")["min_len"])
	assert.Equal(t, "address.*.number value must be an integer", v.Errors.Field("address.*.number")["int"])
	assert.Equal(t, "address.*.country value must be in the enum [es us]", v.Errors.Field("address.*.country")["in"])
	// dump.Println(v.Errors)

	// TODO how to use on struct.
	// type user struct {
	// 	Tags []string `json:"tags" validate:"required|slice"`
	// }
}

// https://github.com/gookit/validate/issues/125
func TestIssue_125(t *testing.T) {
	defer validate.ResetOption()

	validate.Config(func(opt *validate.GlobalOption) {
		opt.SkipOnEmpty = false
	})
	validate.AddValidator("custom", func(value any) bool {
		// validation...
		return true
	})

	v := validate.Map(map[string]any{
		"field": nil,
	})
	v.StringRule("field", "custom")
	assert.True(t, v.Validate())

	var val *int
	v2 := validate.Map(map[string]any{
		"field": val,
	})
	v2.StringRule("field", "custom")
	assert.True(t, v2.Validate())
}

// https://github.com/gookit/validate/issues/135
func TestIssue_135(t *testing.T) {
	type SubjectCreateReq struct {
		Title       string `json:"title" validate:"required|minLen:2|maxLen:512"`           // 题目名称
		SubjectType string `json:"subject_type" validate:"required|in:radio,checkbox,bool"` // 题目类型
		Answer      []struct {
			Score float64 `json:"score" validate:"required|min:0.1"`  // 得分
			Metas string  `json:"metas" validate:"required|minLen:2"` // 元信息
		} `json:"answer"` // 答案
	}

	s := `
{
    "title":"44",
    "answer":[
        {
            "score":0.09,
            "metas":"111"
        }
    ],
    "subject_type":"radio"
}`

	r := &SubjectCreateReq{}
	err := jsonutil.Decode([]byte(s), r)
	assert.NoError(t, err)
	// dump.Println(r)

	v := validate.Struct(r)
	ok := v.Validate()
	dump.Println(v.Errors)
	assert.False(t, ok)
	assert.Equal(t, "answer.0.score min value is 0.1", v.Errors.One())
	assert.Equal(t, "answer.0.score min value is 0.1", v.Errors.OneError().Error())
}

// https://github.com/gookit/validate/issues/140
func TestIssue_140(t *testing.T) {
	type Test struct {
		Field1 string
		Field2 string `validate:"requiredIf:Field1,value"`
	}

	test := &Test{Field1: "value", Field2: ""}

	v := validate.Struct(test)
	err := v.ValidateE()
	dump.Println(err)
	assert.Error(t, err)
	assert.Equal(t, "Field2 is required when Field1 is in [value]", err.One())

	test.Field2 = "hi"
	v = validate.Struct(test)
	err = v.ValidateE()
	assert.Nil(t, err)
}

// https://github.com/gookit/validate/issues/143
func TestIssue_143(t *testing.T) {
	type Data struct {
		Name string `validate:"required"`
		Age  *int   `json:"age" validate:"required"`
	}

	v := validate.New(&Data{
		Name: "tom",
		Age:  nil,
	})

	ok := v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "age is required to not be empty", v.Errors.One())

	// use ptr
	age := 0
	v = validate.New(&Data{
		Name: "tom",
		Age:  &age,
	})

	ok = v.Validate()
	assert.True(t, ok)

	// custom not_null check.
	// TIP: validator name must start with required
	type Data1 struct {
		Name string `validate:"required"`
		Age  *int   `json:"age" validate:"required_custom" message:"required_custom:{field} should not null"`
	}

	// - use nil
	v = validate.New(&Data1{
		Name: "tom",
		Age:  nil,
	})
	v.AddValidator("required_custom", func(val any) bool {
		return !validate.IsNilObj(val)
	})

	ok = v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "age should not null", v.Errors.One())

	// - use zero value
	age = 0
	v = validate.New(&Data1{
		Name: "tom",
		Age:  &age,
	})
	v.AddValidator("required_custom", func(val any) bool {
		return !validate.IsNilObj(val)
	})

	ok = v.Validate()
	assert.True(t, ok)

	// - use other value
	age = 20
	v = validate.New(&Data1{
		Name: "tom",
		Age:  &age,
	})
	v.AddValidator("required_custom", func(val any) bool {
		return !validate.IsNilObj(val)
	})

	ok = v.Validate()
	assert.True(t, ok)

	// - with other validator
	age = 20
	v = validate.New(&Data1{
		Name: "tom",
		Age:  &age,
	})
	v.AddValidator("required_custom", func(val any) bool {
		return !validate.IsNilObj(val)
	})
	v.AddRule("age", "min", 30)

	ok = v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "age min value is 30", v.Errors.One())

	age = 30
	v = validate.New(&Data1{
		Name: "tom",
		Age:  &age,
	})
	v.AddValidator("required_custom", func(val any) bool {
		return !validate.IsNilObj(val)
	})
	v.AddRule("age", "min", 30)

	ok = v.Validate()
	dump.Println(v.Errors)
	assert.True(t, ok)
	assert.Equal(t, "", v.Errors.One())
}

// https://github.com/gookit/validate/issues/148
func TestIssues_148(t *testing.T) {
	type A struct {
		T1 string `validate:"required"`
		T2 string `validate:"required"`
	}

	a := &A{}
	goutil.PanicErr(jsonutil.DecodeString(`{"T2":"xxx"}`, a))

	v := validate.Struct(a)
	v.Validate()

	assert.True(t, v.IsFail())
	assert.False(t, v.IsSuccess())
	assert.Equal(t, "T1 is required to not be empty", v.Errors.One())

	type B struct {
		A
		T1 string `json:"T1,omitempty"` // is not required
	}

	b := &B{}
	goutil.PanicIfErr(jsonutil.DecodeString(`{"T2":"xxx"}`, b))

	// dump.Println(b)
	v = validate.Struct(b)
	v.Validate()

	assert.False(t, v.IsOK())
}

// https://github.com/gookit/validate/issues/152
func TestIssue_152(t *testing.T) {
	old := validate.CopyGlobalMessages()
	defer func() {
		validate.SetBuiltinMessages(old)
	}()

	// test required if
	type requiredIf struct {
		Type int64  `validate:"required" label:"类型"`
		Data string `validate:"required_if:Type,1" label:"数据"`
	}

	v := validate.Struct(requiredIf{
		Type: 1,
		Data: "",
	})
	zhcn.Register(v)

	assert.False(t, v.Validate())
	assert.Equal(t, `当 类型 为 [1] 时 数据 不能为空。`, v.Errors.One())

	// test required unless
	type requiredUnless struct {
		Type int64  `label:"类型"`
		Data string `validate:"required_unless:Type,1" label:"数据"`
	}

	v = validate.Struct(requiredUnless{
		Type: 0,
		Data: "",
	})
	zhcn.Register(v)

	assert.False(t, v.Validate())
	assert.Equal(t, `当 类型 不为 [1] 时 数据 不能为空。`, v.Errors.One())
}

// https://github.com/gookit/validate/issues/156
func TestIssues_156(t *testing.T) {
	assert.True(t, validate.IsEmail("abc.88@qq.com"))
	assert.True(t, validate.IsEmail("xxxx.88@qq.com"))
}

// https://github.com/gookit/validate/issues/157
func TestIssues_157(t *testing.T) {
	type SubsrcSubmitReq struct {
		StockCode string `json:"stock_code" validate:"required|maxLen:20" filter:"trim"`
		UserEmail string `json:"user_email" validate:"required|regexp:^\\w{4,6}$" filter:"trim"`
	}

	req := &SubsrcSubmitReq{}
	err := jsonutil.DecodeString(`{
    "stock_code": "11111111",
    "user_email": "aaaaaaaaaaaaaaaaa"
}`, req)
	assert.NoError(t, err)

	v := validate.Struct(req)
	ok := v.Validate()

	assert.False(t, ok)
	assert.Error(t, v.Errors)
	// dump.Println(req, v.Errors)
}

// https://github.com/gookit/validate/issues/159
func TestIssues_159(t *testing.T) {
	type TestStruct struct {
		Start string `json:"start" validate:"date|minLen:10"`
		End   string `json:"end" validate:"date|minLen:10|gteField:start"`
	}

	ts := &TestStruct{
		Start: "2021-12-17",
		End:   "2020-12-16",
	}

	v := validate.Struct(ts)
	ok := v.Validate()
	// dump.Println(v.Errors)
	assert.False(t, ok)
	assert.Equal(t, "end value should be greater or equal to the field start", v.Errors.One())
}

// https://github.com/gookit/validate/issues/160
func TestIssues_160(t *testing.T) {
	type User struct {
		Email string `json:"email" validate:"required|email" message:"required:eamil必填的信息|email:邮箱验证失败的信息"`
	}

	req := &User{}
	err := jsonutil.DecodeString(`{
    "email": "aaaaaaaaaaaaaaaaa"
}`, req)
	assert.NoError(t, err)

	v := validate.Struct(req)
	v.Validate()

	// dump.Println(req, v.Errors)
	assert.False(t, v.IsOK())
	assert.Equal(t, "邮箱验证失败的信息", v.Errors.One())
}

// https://github.com/gookit/goutil/issues/60
func TestGoutilIssues_60(t *testing.T) {
	type User struct {
		Time timex.TimeX `json:"time" validate:"required"`
	}

	req := &User{}
	err := jsonutil.DecodeString(`{
    "time": "2018-10-16 12:34:01"
}`, req)
	assert.NoError(t, err)
	assert.Equal(t, "2018-10-16 12:34", req.Time.TplFormat("Y-m-d H:i"))

	// v := validate.Struct(req)
	// v.Validate()

	dump.Println(req)
}

type Foo172 struct {
	Domains []string
}

func (gs Foo172) ConfigValidation(v *validate.Validation) {
	// v.StringRule("Domains.*", "trim") // ERR: should use param 'filterRule'
	v.StringRule("Domains.*", "", "trimStrings")
}

// https://github.com/gookit/validate/issues/172
func TestIssues_172(t *testing.T) {
	f := Foo172{
		Domains: []string{"   test.com   ", "oof.com", " foobar.com"},
	}

	v := validate.Struct(&f) // nolint:varnamelen
	if !v.Validate() {
		fmt.Println(v.Errors)
	}
	dump.Println(f)

	assert.Equal(t, []string{"test.com", "oof.com", "foobar.com"}, f.Domains)
}

// https://github.com/gookit/validate/issues/192
// v1.4.6 change has broken existing behaviour #192
func TestIssues_192(t *testing.T) {
	type Request struct {
		Main1 *struct {
			Child11 int `json:"child11" validate:"required"`
			Child12 int `json:"child12" validate:"required"`
		} `json:"main1" validate:"optional"` // optional - will not validate Main1.* on Main1 is nil

		Main2 *struct {
			Child21 int `json:"child21" validate:"required"`
			Child22 int `json:"child22" validate:"required"`
		} `json:"main2"`
	}

	jsonStr := `{
  "main1": {
    "child12": 2
  },
  "main2": {
    "child21": 5,
    "child22": 6
  }
}`

	req := &Request{}
	err := jsonutil.DecodeString(jsonStr, req)
	assert.NoErr(t, err)

	t.Run("with child data", func(t *testing.T) {
		v := validate.Struct(req)
		assert.False(t, v.Validate())
		assert.StrContains(t, v.Errors.String(), "main1.child11 is required to not be empty")
	})

	// set main1 = nil, should not validate Main1.*
	t.Run("set main1 to nil", func(t *testing.T) {
		req.Main1 = nil // fields under Main1 will not be validated
		v := validate.Struct(req)
		assert.True(t, v.Validate())
	})
}

// https://github.com/gookit/validate/issues/206
func TestIssues_206(t *testing.T) {
	m := map[string]any{
		"name":  "inhere",
		"age":   99,
		"oldSt": 10,
		"newSt": 10,
		"email": "some@email.com",
		"tags":  []string{"go", "php", "java"},
		"origins": []map[string]string{
			{"name": "test", "url": "https://test.com"},
			{"name": "test", "url": "https://test.com"},
		},
	}

	v := validate.Map(m)
	v.StopOnError = false
	v.SkipOnEmpty = false
	// can also
	v.StringRule("age", "required|int|min:1|max:99")
	v.StringRule("name", "required|minLen:2")
	v.StringRule("tags", "required|slice|minlen:1")
	// feat: support check sub-item in slice
	v.StringRule("tags.*", "required|string|min_len:1")
	v.StringRule("origins", "required|slice|minlen:1")
	v.StringRule("origins.*.name", "required|full_url")

	assert.False(t, v.Validate())
	// fmt.Println(v.Errors)
	assert.Len(t, v.Errors, 1)
	assert.StrContains(t, v.Errors.String(), "origins.*.name must be a valid full URL address")
}

// https://github.com/gookit/validate/issues/209
func TestIssues_209(t *testing.T) {
	mp := make(map[string]any)
	err := jsonutil.DecodeString(`{
    "em": {
        "code": "001",
        "encounter_uid": 1,
        "work_item_uid": 2, 
        "billing_provider": "Test provider",
        "resident_provider": "Test Resident Provider"
    },
    "cpt": [
        {
            "code": "001",
            "billing_provider": "Test provider",
            "resident_provider": "Test Resident Provider"
        },
        {
            "code": "OBS01",
            "billing_provider": "Test provider",
            "resident_provider": "Test Resident Provider"
        },
        {
            "code": "SU002",
            "resident_provider": "Test Resident Provider"
        }
    ]
}`, &mp)
	assert.NoErr(t, err)

	v := validate.Map(mp)
	v.StopOnError = false
	v.StringRule("cpt.*.encounter", "required|int")
	v.StringRule("cpt.*.billing_provider", "required")

	assert.False(t, v.Validate())
	// fmt.Println(v.Errors)
	assert.StrContains(t, v.Errors.String(), "cpt.*.encounter is required to not be empty")
	assert.StrContains(t, v.Errors.String(), "cpt.*.billing_provider is required to not be empty")
}

// https://github.com/gookit/validate/issues/213
func TestIssues_213(t *testing.T) {
	type Person struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age" validate:"required"`
	}
	type Form struct {
		Data []Person `validate:"required"`
	}

	f := &Form{}
	v := validate.Struct(f) // nolint:varnamelen
	assert.False(t, v.Validate())
	// fmt.Println(v.Errors)
	assert.StrContains(t, v.Errors.String(), "Data is required to not be empty")

	f = &Form{
		Data: []Person{
			{Name: "tome"},
		},
	}
	v = validate.Struct(f) // nolint:varnamelen
	assert.False(t, v.Validate())
	// fmt.Println(v.Errors)
	assert.StrContains(t, v.Errors.String(), "age is required to not be empty")
}

// https://github.com/gookit/validate/issues/217
func TestIssues_217(t *testing.T) {
	type Sample struct {
		Val *bool `validate:"required"`
	}

	type Nested struct {
		Samples []Sample `validate:"slice"`
	}

	val, val2 := false, true
	data := Sample{Val: &val}
	data2 := Sample{Val: &val2}
	data3 := Sample{Val: nil}

	data4 := Nested{Samples: []Sample{data, data2}}

	v4 := validate.Struct(data4)
	ok4 := v4.Validate()
	fmt.Println(ok4, v4.Errors) // TODO Should not fail and fails. required check bool value

	v1 := validate.Struct(data)
	ok1 := v1.Validate()
	assert.True(t, ok1) // Should not fail and does not fail.

	v2 := validate.Struct(data2)
	ok2 := v2.Validate()
	assert.True(t, ok2) // Should not fail and does not fail.

	v3 := validate.Struct(data3)
	ok3 := v3.Validate()
	assert.False(t, ok3) // Should fail and fails.

}

// https://github.com/gookit/validate/issues/221
func TestIssues_221(t *testing.T) {
	m := map[string]any{
		"clinics": []map[string]any{
			{
				"clinic_id": "1",
				"doctors": []map[string]any{
					{
						"doctor_id": 1,
						"dates": []map[string]string{
							{
								"date": "2023-01-01",
							},
						},
					},
				},
			},
		},
	}

	dump.Println(maputil.GetByPath("clinics.*.doctors.*.dates.*.date", m))

	v := validate.Map(m)

	v.StringRule("clinics", "required|array")
	v.StringRule("clinics.*.clinic_id", "required|string")
	v.StringRule("clinics.*.doctors", "required|array")
	v.StringRule("clinics.*.doctors.*.doctor_id", "required")
	v.StringRule("clinics.*.doctors.*.dates", "required|array")
	v.StringRule("clinics.*.doctors.*.dates.*.date", "required|string")

	if assert.True(t, v.Validate()) { // validate ok
		safeData := v.SafeData()

		fmt.Println("Validation OK:")
		dump.Println(safeData)
	} else {
		fmt.Println("Validation Fail:")
		fmt.Println(v.Errors) // all error messages
	}
}

// https://github.com/gookit/validate/issues/223
func TestIssues_223(t *testing.T) {
	m := map[string]any{
		"clinics": []map[string]any{
			{
				"clinic_id": nil,
				"doctors": []map[string]any{
					{
						"duration": nil,
					},
				},
			},
		},
	}

	v := validate.Map(m)
	v.StringRule("clinics", "required|array")
	v.StringRule("clinics.*.clinic_id", "string")

	if assert.False(t, v.Validate()) {
		fmt.Println("Validation Fail:")
		fmt.Println(string(v.Errors.JSON())) // all error messages
		assert.StrContains(t, v.Errors.String(), "clinics.*.clinic_id value must be a string")
	} else {
		safeData := v.SafeData()
		fmt.Println("Validation OK:")
		dump.Println(safeData)
	}

	// test field in sub slice
	v = validate.Map(m)

	v.StringRule("clinics", "required|array")
	v.StringRule("clinics.*.doctors.*.duration", "int")
	assert.False(t, v.Validate())
	s := v.Errors.String()
	fmt.Println(s)
	assert.StrContains(t, s, "clinics.*.doctors.*.duration value must be an integer")
}

// https://github.com/gookit/validate/issues/242
// requiredWithoutAll tag is not work with struct validation
func TestIssues_242(t *testing.T) {
	type RequiredWithoutAll struct {
		ID    string `validate:"requiredWithoutAll:NewID|uuid4"`
		NewID string `validate:"requiredWithoutAll:OldID|uuid4"`
		OldID string `validate:"requiredWithoutAll:NewID|string"`
	}

	h := RequiredWithoutAll{}
	v := validate.Struct(h)
	assert.False(t, v.Validate())
	s := v.Errors.String()
	fmt.Println(v.Errors)
	assert.StrContains(t, s, "ID field is required when none of [NewID] are present")
}

// https://github.com/gookit/validate/issues/245
// required_if 中，当指定的 anotherField 为 uint8 类型时，校验失效
func TestIssues_245(t *testing.T) {
	type MyStruct struct {
		ID     int64    `json:"id" validate:"required"`
		Type   uint8    `json:"ip_type" validate:"in:0,1"`
		IpList []string `json:"ip_list" validate:"required|validateIP"`
		Field  int      `json:"field" validate:"required_if:Type,1"`
	}

	ms := &MyStruct{Type: 1}
	v := validate.Struct(ms)
	assert.False(t, v.Validate())
	assert.ErrSubMsg(t, v.Errors, "id is required to not be empty")
	// fmt.Println(v.Errors)
}

// issues#246 类似这样的结构体和规则
type CertStore struct {
	Type      string   `form:"type" json:"type"`
	Domains   []string `form:"domains" json:"domains"`
	AutoRenew bool     `form:"auto_renew" json:"auto_renew"`
	UserID    uint     `form:"user_id" json:"user_id" filter:"uint"`
	DNSID     uint     `form:"dns_id" json:"dns_id" filter:"uint"`
	WebsiteID uint     `form:"website_id" json:"website_id" filter:"uint"`
}

func (r *CertStore) Rules() map[string]string {
	return map[string]string{
		"type":       "required|in:P256,P384,2048,4096",
		"domains":    "required|array",
		"auto_renew": "required|bool",
		"user_id":    "required|uint|exists:cert_users,id",
		"dns_id":     "uint",
		"website_id": "uint",
	}
}

// https://github.com/gookit/validate/issues/246
// 使用 uint filter时，传 null 值会报错 convert value type error
func TestIssues_246(t *testing.T) {
	jsonStr := `{
    "domains": [],
    "dns_id": null,
    "type": "P256",
    "user_id": null,
    "website_id": null,
    "auto_renew": true
}`
	cs := &CertStore{}
	err := jsonutil.DecodeString(jsonStr, cs)
	assert.NoErr(t, err)
	dump.P(cs)

	v := validate.Struct(cs)
	assert.True(t, v.Validate())
	dump.P(cs)
}

type mockUUID [16]byte

type TestUser struct {
	Name             string    `validate:"required|minLen:2|maxLen:100" filter:"trim|lower"`
	Email            string    `validate:"required|email" filter:"lower"`
	PtrString        *string   `validate:"required|minLen:2|maxLen:100" filter:"trim|lower"`
	OptionalString   *string   `validate:"required|minLen:2|maxLen:100" filter:"trimPtr|lowerPtr"`
	TestOptionalUUID *mockUUID // **Problem LINE**
}

// https://github.com/gookit/validate/issues/247
// Pointer of UUID will break the Validate() function
func TestIssues_247(t *testing.T) {
	str := "    Optional String   "
	str1 := "    Optional String1   "
	tu := &TestUser{
		Name:           "  John Doe  ",
		Email:          "JOHNDOE@EXAMPLE.COM",
		PtrString:      &str,
		OptionalString: &str1,
	}

	v := validate.Struct(tu)
	v.AddFilter("trimPtr", func(val any) any {
		if strPtr, ok := val.(*string); ok && strPtr != nil {
			trimmed := strings.TrimSpace(*strPtr)
			return &trimmed
		}
		return val
	})
	v.AddFilter("lowerPtr", func(val any) any {
		if strPtr, ok := val.(*string); ok && strPtr != nil {
			lowered := strings.ToLower(*strPtr)
			return &lowered
		}
		return val
	})
	assert.True(t, v.Validate())
	dump.P(tu)
	fmt.Println(v.Errors)
}

// https://github.com/gookit/validate/issues/250
// Does not work with *int ?
func TestIssues_250(t *testing.T) {
	type Salary struct {
		MinSalary   *int `json:"MinSalary" validate:"int|min:10000" message:"The minimum salary is 10.000 EUR"`
		OkSalary    *int `json:"OkSalary" validate:"int|min:10000" message:"The ok salary is larger than 10.000 EUR"`
		HappySalary *int `json:"HappySalary" validate:"int|min:10000" message:"The happy salary is larger than 10.000 EUR"`
	}

	// works: optional + right value
	iv1 := 30000
	st := &Salary{OkSalary: &iv1}
	v := validate.Struct(st)
	assert.True(t, v.Validate())

	// works
	iv := 3
	st = &Salary{MinSalary: &iv}
	v = validate.Struct(st)
	assert.False(t, v.Validate())
	assert.ErrMsg(t, v.Errors, "min: The minimum salary is 10.000 EUR")

	// works: use zero value
	iv2 := 0
	st = &Salary{MinSalary: &iv2}
	v = validate.Struct(st)
	assert.False(t, v.Validate())
	assert.ErrMsg(t, v.Errors, "min: The minimum salary is 10.000 EUR")

	// case 2
	type A struct {
		A1 *int `validate:"int|min:200"`
		A2 *int `validate:"int|min:200"`
	}

	// works: optional + right value
	i := 300
	pa := A{A1: &i, A2: nil}
	v = validate.Struct(pa)
	assert.True(t, v.Validate())

	// works: optional + error value
	i2 := 3
	pa = A{A1: &i2, A2: nil}
	v = validate.Struct(pa)
	assert.False(t, v.Validate())
	assert.StrContains(t, v.Errors.String(), "A1 min value is 200")

	// works
	i3 := 0
	pa = A{A1: &i3, A2: nil}
	v = validate.Struct(pa)
	assert.False(t, v.Validate())
	assert.StrContains(t, v.Errors.String(), "A1 min value is 200")
}

// DataIssues252 struct
type DataIssues252 struct {
	PtrStrings *[]string `validate:"strings"`
}

// https://github.com/gookit/validate/issues/252
// Panic when validating struct with nil pointer to a slice field
func TestIssues_252(t *testing.T) {
	v := validate.New(&DataIssues252{})

	ok := v.Validate()
	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/255
// Error when using custom filter with pointer to slice of strings #255
func TestIssues_255(t *testing.T) {
	type Request struct {
		PtrStringSlice *[]string `json:"string_slice" filter:"ptrTrimStrings" validate:"strings"`
	}

	t.Run("return pointer", func(t *testing.T) {
		ss1 := []string{" foobar "}
		req := &Request{PtrStringSlice: &ss1}
		v := validate.New(req)

		v.AddFilters(validate.M{
			"ptrTrimStrings": func(val *[]string) *[]string {
				trimmedSlice := make([]string, len(*val))
				for i, str := range *val {
					trimmedSlice[i] = strings.TrimSpace(str)
				}
				return &trimmedSlice
			},
		})

		assert.True(t, v.Validate())
		assert.Equal(t, []string{"foobar"}, *req.PtrStringSlice)
	})

	t.Run("return value", func(t *testing.T) {
		ss1 := []string{" foobar "}
		req := &Request{PtrStringSlice: &ss1}
		v := validate.New(req)

		// return value []string
		v.AddFilters(validate.M{
			"ptrTrimStrings": func(val *[]string) []string {
				trimmedSlice := make([]string, len(*val))
				for i, str := range *val {
					trimmedSlice[i] = strings.TrimSpace(str)
				}
				return trimmedSlice
			},
		})

		assert.True(t, v.Validate())
		assert.Equal(t, []string{"foobar"}, *req.PtrStringSlice)
	})
}

// https://github.com/gookit/validate/issues/276
// Struct validation: invalid memory address or nil pointer dereference #276
func TestIssues_276(t *testing.T) {
	type user struct {
		Name string `validate:"required"`
		Age  int    `validate:"required_if:Name,lee"`
	}

	u := &user{Name: "lee"}
	v := validate.Struct(u)

	assert.False(t, v.Validate())
	fmt.Println(v.Errors)              // all error messages
	fmt.Println(v.Errors.One())        // returns a random error message text
	fmt.Println(v.Errors.OneError())   // returns a random error
	fmt.Println(v.Errors.Field("Age")) // returns error messages of the field
}

// https://github.com/gookit/validate/issues/280 can not validate uuid field
func TestIssues_280(t *testing.T) {
	type TestData struct {
		ID mockUUID
	}

	data := TestData{}
	v := validate.New(data)
	assert.Nil(t, v.ValidateErr())
}

type WithArray struct {
	Extras []ExtraInfo `validate:"required|minLen:2"`
}

type WithArrayJSON struct {
	Extras []ExtraInfoJSON `json:"extras" validate:"required|minLen:2"`
}

type ExtraInfo struct {
	Github  string `validate:"required"`
	Status1 int    `validate:"required|int"`
}

type ExtraInfoJSON struct {
	Github  string `json:"github" validate:"required"`
	Status1 int    `json:"status1" validate:"required|int"`
}

// http://github.com/gookit/validate/issues/297 Nested JSON tag paths are not respected
func TestIssue_297(t *testing.T) {
	v := validate.New(WithArray{
		Extras: []ExtraInfo{
			{
				Github:  "",
				Status1: 0,
			},
			{
				Github:  "",
				Status1: 0,
			},
		},
	})
	assert.False(t, v.Validate())
	var firstKey string
	for key := range v.Errors {
		firstKey = key
		break
	}
	assert.Equal(t, "Extras.0.Github", firstKey)

	v = validate.New(WithArrayJSON{
		Extras: []ExtraInfoJSON{
			{
				Github:  "",
				Status1: 0,
			},
			{
				Github:  "",
				Status1: 0,
			},
		},
	})
	assert.False(t, v.Validate())
	var firstKeyJSON string
	for key := range v.Errors {
		firstKeyJSON = key
		break
	}
	assert.Equal(t, "extras.0.github", firstKeyJSON)
}

// http://github.com/gookit/validate/issues/272 eqField does not correctly validate pointer type data
func TestIssue_272(t *testing.T) {
	type Test struct {
		FieldA *string `validate:"required"`
		FieldB *string `validate:"eqField:FieldA"`
	}

	var (
		fieldA = "A"
		fieldB = "B"
	)
	t.Run("equal", func(t *testing.T) {
		v := validate.Struct(Test{
			FieldA: &fieldA,
			FieldB: &fieldA,
		})
		assert.True(t, v.Validate())
	})

	t.Run("not equal", func(t *testing.T) {
		v := validate.Struct(Test{
			FieldA: &fieldA,
			FieldB: &fieldB,
		})
		assert.False(t, v.Validate())
		assert.Equal(t, "FieldB value must be equal the field FieldA", v.Errors.One())
	})
}

// http://github.com/gookit/validate/issues/301 Regex pipe escape
func TestIssue_301(t *testing.T) {
	v := validate.Map(map[string]any{
		"field":   "abc.json",
		"other":   "123",
		"escape":  "a|b",
		"escape2": "a\\|b",
	})
	v.StringRule("field", "required|regex:^[a-zA-Z0-9_.-]+\\.(yaml\\|yml\\|json)$|maxLen:50")
	v.StringRule("other", "required|regex:^\\d{3}$")
	v.StringRule("escape", "in: a\\|b")
	v.StringRule("escape2", "in: a\\\\|b")
	assert.True(t, v.Validate())
}

// http://github.com/gookit/validate/issues/302 The required rule fails for int/uint values when the value is 0.
func TestIssue_302(t *testing.T) {
	v := validate.Map(map[string]any{
		"required":             0,
		"required_if":          0,
		"required_unless":      0,
		"required_with":        0,
		"required_with_all":    0,
		"required_without":     0,
		"required_without_all": 0,
	})
	v.StringRules(map[string]string{
		"required":             "required|uint|in:0,1,2",
		"required_if":          "required_if:required,0",
		"required_unless":      "required_unless:required,1",
		"required_with":        "required_with:required",
		"required_with_all":    "required_with_all:required,required_if",
		"required_without":     "required_without:nonexist",
		"required_without_all": "required_without_all:nonexist,required",
	})
	assert.True(t, v.Validate())
	assert.Empty(t, v.Errors.One())
}
