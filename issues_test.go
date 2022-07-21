package validate_test

import (
	"fmt"
	"github.com/gookit/validate/locales/zhcn"
	"testing"
	"time"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/validate"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, validate.ErrConvertFail.Error(), err.Error())
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
	is.Equal("Avatar must be an valid full URL address", v.Errors.One())

	validate.ResetOption()
	v = validate.New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("avatar must be an valid full URL address", v.Errors.One())
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
		Nickname string `validate:"required|minLen:6" message:"required:OO! nickname cannot be empty!|minLen:OO! nickname min len is %d"`
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
	v.AddValidator("checkAge", func(val interface{}, ints ...int) bool {
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
	m := map[string]interface{}{
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
	assert.Equal(t, "categories is required when is_private is [false]", v.Errors.One())

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
	assert.Equal(t, "Sex is required and not empty", v.Errors.One())

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
	m := map[string]interface{}{
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
	assert.Equal(t, "a value should greater the 100", v.Errors.One())

	v = validate.Map(m)
	v.AddRule("a", "required")
	v.AddRule("a", "gt", 100)

	ok = v.Validate()
	assert.False(t, ok)
	assert.Equal(t, "a is required and not empty", v.Errors.One())
}

// https://gitee.com/inhere/validate/issues/I3B3AV
func TestIssues_I3B3AV(t *testing.T) {
	m := map[string]interface{}{
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
	m := map[string]interface{}{
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
	assert.Contains(t, v.Errors.String(), "SomeID is required and not empty")

	type Example2 struct {
		SomeID string `json:"some_id" validate:"required" `
	}

	e2 := Example2{}
	v2 := validate.Struct(e2)
	v2.Validate()
	dump.Println(v2.Errors)
	err2 := v2.Errors.String() // here we get something like {"some_id": { /* ... */ }}
	assert.Contains(t, err2, "some_id")
	assert.Contains(t, err2, "some_id is required and not empty")
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
	assert.Equal(t, "id is required and not empty", v.Errors.One())

	v = validate.Struct(d, "add")
	ok = v.Validate()
	dump.Println(v.Errors, v.SceneFields())
	assert.False(t, ok)
	assert.Equal(t, "add", v.Scene())
	assert.Equal(t, "id is required and not empty", v.Errors.One())

	// right
	d.Issue104A.ID = 34
	v = validate.Struct(d)
	ok = v.Validate()
	assert.True(t, ok)
}

// https://github.com/gookit/validate/issues/107
func TestIssues_107(t *testing.T) {
	taFilter := func(val interface{}) int64 {
		if val != nil {
			// log.WithFields(log.Fields{"value": val}).Info("value should be other than nil")
			return int64(val.(float64))
		} else {
			// log.WithFields(log.Fields{"value": val}).Info("value should be nil")
			return -1
		}
	}

	v := validate.Map(map[string]interface{}{
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
	v := validate.New(map[string]interface{}{
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
func TestIssue_124(t *testing.T) {
	m := map[string]interface{}{
		"names": []string{"John", "Jane", "abc"},
	}

	v := validate.Map(m)
	v.StringRule("names", "required|array|minLen:1")
	v.StringRule("names.*", "required|string|min_len:4")

	assert.False(t, v.Validate())
	assert.Error(t, v.Errors.ErrOrNil())
	assert.Equal(t, "names.* min length is 4", v.Errors.One())
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
	validate.AddValidator("custom", func(value interface{}) bool {
		// validation...
		return true
	})

	v := validate.Map(map[string]interface{}{
		"field": nil,
	})
	v.StringRule("field", "custom")
	assert.True(t, v.Validate())

	var val *int
	v2 := validate.Map(map[string]interface{}{
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
	assert.Equal(t, "score min value is 0.1", v.Errors.One())
	assert.Equal(t, "score min value is 0.1", v.Errors.OneError().Error())
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
	assert.Equal(t, "Field2 is required when Field1 is [value]", err.One())

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
	assert.Equal(t, "age is required and not empty", v.Errors.One())

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
	v.AddValidator("required_custom", func(val interface{}) bool {
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
	v.AddValidator("required_custom", func(val interface{}) bool {
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
	v.AddValidator("required_custom", func(val interface{}) bool {
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
	v.AddValidator("required_custom", func(val interface{}) bool {
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
	v.AddValidator("required_custom", func(val interface{}) bool {
		return !validate.IsNilObj(val)
	})
	v.AddRule("age", "min", 30)

	ok = v.Validate()
	dump.Println(v.Errors)
	assert.False(t, ok)
	assert.Equal(t, "age min value is 30", v.Errors.One())
}

func TestIssue_152(t *testing.T) {
	zhcn.RegisterGlobal()

	// test required if
	type requiredIf struct {
		Type int64  `validate:"required" label:"类型"`
		Data string `validate:"required_if:Type,1" label:"数据"`
	}

	v := validate.Struct(requiredIf{
		Type: 1,
		Data: "",
	})

	v.Validate()

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

	v.Validate()

	assert.Equal(t, `当 类型 不为 [1] 时 数据 不能为空。`, v.Errors.One())
}
