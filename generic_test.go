package validate

import (
	"fmt"
	"mime/multipart"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValueLen(t *testing.T) {
	is := assert.New(t)
	tests := []interface{}{
		"abc",
		123,
		int8(123), int16(123), int32(123), int64(123),
		uint8(123), uint16(123), uint32(123), uint64(123),
		float32(123), float64(123),
		[]int{1, 2, 3}, []string{"a", "b", "c"},
		map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"},
	}

	for _, sample := range tests {
		is.Equal(3, ValueLen(reflect.ValueOf(sample)))
	}

	is.Equal(-1, ValueLen(reflect.ValueOf(nil)))
}

func TestCallByValue(t *testing.T) {
	is := assert.New(t)
	is.Panics(func() {
		CallByValue(reflect.ValueOf("invalid"))
	})
	is.Panics(func() {
		CallByValue(reflect.ValueOf(IsJSON), "age0", "age1")
	})
}

func TestData(t *testing.T) {
	is := assert.New(t)
	// MapData
	d := FromMap(M{
		"age": 45,
	})

	err := d.Set("name", "inhere")
	is.Nil(err)

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
	err = sd.Set("name", "def")
	is.Nil(err)
	val, ok = sd.Get("name")
	is.True(ok)
	is.Equal("def", val)

	is.Error(sd.Set("notExist", "val"))
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

	err := d.Set("newKey", "strVal")
	is.NoError(err)
	is.Equal("strVal", d.String("newKey"))
	err = d.Set("newInt", 23)
	is.NoError(err)
	is.Equal(23, d.Int("newInt"))
	is.Error(d.Set("invalid", []int{2}))

	// form
	d.Add("newKey1", "new val1")
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
		Extra:     ExtraInfo{"xxx", 2},
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
	v = d.Create()
	is.True(v.Validate())

	// get field value
	str, ok := d.Get("Name")
	is.True(ok)
	is.Equal("new name", str)

	iVal, ok := d.Get("Extra.Status1")
	is.True(ok)
	is.Equal(2, iVal)

	// not exist
	ret, ok := d.Get("NotExist")
	is.False(ok)
	is.Nil(ret)

	ret, ok = d.Get("NotExist.SubKey")
	is.False(ok)
	is.Nil(ret)

	ret, ok = d.Get("Extra.NotExist")
	is.False(ok)
	is.Nil(ret)

	// set value
	err = d.Set("protected", "new text")
	is.Error(err)
	err = d.Set("Name", "inhere")
	is.Nil(err)
	str, ok = d.Get("Name")
	is.True(ok)
	is.Equal("inhere", str)
}

func TestAddFilter(t *testing.T) {
	is := assert.New(t)
	is.Panics(func() {
		AddFilter("myFilter", "invalid")
	})
	is.Panics(func() {
		AddFilter("myFilter", func() {})
	})
	is.Panics(func() {
		AddFilter("bad-name", func() {})
	})
	is.Panics(func() {
		AddFilter("", func() {})
	})
	is.Panics(func() {
		AddFilter("myFilter", func(v string) (bool, int) { return false, 0 })
	})
	is.Panics(func() {
		AddFilter("myFilter", func() interface{} { return nil })
	})

	AddFilters(M{
		"myFilter0": func(val interface{}) string { return "myFilter0" },
	})
	AddFilter("myFilter1", func(val interface{}) string { return "myFilter1" })

	v := New(map[string]interface{}{
		"name": " inhere ",
		"age":  " 50 ",
		"key0": "val0",
		"key1": "val1",
		"tags": "go,php",
	})
	v.AddFilters(M{
		"myFilter2": func(val interface{}, a, b string) (string, error) { return "myFilter2:" + a + b, nil },
	})
	v.FilterRule("key0", "myFilter0")
	v.FilterRules(MS{
		"key1": "myFilter2:a,b",
		"name": "trim|upper",
		"tags": "str2arr:,",
		//
		"age, not-exist": "trim|int",
	})

	is.Panics(func() {
		v.FilterRule("", "")
	})

	v.Sanitize() // do filtering
	v.Sanitize() // repeat call
	is.True(v.IsOK())
	is.Equal(50, v.Filtered("age"))
	is.Equal("INHERE", v.Filtered("name"))
	is.Equal("myFilter0", v.Filtered("key0"))
	is.Equal("myFilter2:ab", v.Filtered("key1"))
	is.Contains(fmt.Sprint(v.FilteredData()), "key0:myFilter0")

	v.Trans().AddMessage("new-key", "msg text")
	is.True(v.Trans().HasMessage("new-key"))
	is.Equal("msg text", v.Trans().Message("new-key", "some"))
	is.Equal("some did not pass validate", v.Trans().Message("not-exist", "some"))
	v.Trans().Reset()
	is.False(v.Trans().HasMessage("new-key"))

	// filter fail
	v = New(SValues{
		"name": {"inhere"},
	})
	v.AddFilter("myFilter3", func(s string) (string, error) {
		return s, fmt.Errorf("report a error")
	})
	v.FilterRules(MS{
		"name": "invalid|int",
	})
	v.Filtering()
	is.True(v.IsFail())
	is.Contains(v.Errors, "_filter")

	v = New(url.Values{
		"age": {"invalid"},
	})
	v.AddFilter("myFilter3", func(s string) (string, error) {
		return s, fmt.Errorf("report a error")
	})
	v.FilterRules(MS{
		"age": "myFilter3",
	})
	v.Filtering()
	is.True(v.IsFail())
	is.Equal("report a error", v.Errors.Get("_filter"))
}

func TestRule(t *testing.T) {
	is := assert.New(t)
	data := url.Values{
		"name": []string{"inhere"},
		"age":  []string{"10"},
		"key0": []string{"val0"},
	}

	v := New(data)
	// new rule
	r := NewRule("name", "minLen", 6)
	r.SetScene("test") // only validate on scene "test"
	r.SetFilterFunc(func(val interface{}) (interface{}, error) {
		return val.(string) + "-HI", nil
	})
	r.SetBeforeFunc(func(field string, v *Validation) bool {
		return true
	})

	is.Equal([]string{"name"}, r.Fields())
	v.AppendRule(r)
	v.AddRule("field0", "required").SetOptional(true)
	v.AddRule("key0", "inRule").SetCheckFunc(func(s string) bool {
		return s == "val0"
	})
	v.AddRule("name", "gtField", "key0")

	// validate. will skip validate field "name"
	v.Validate()
	is.True(v.IsOK())
	is.Equal("val0", v.SafeVal("key0"))
	is.Equal(nil, v.SafeVal("not-exist"))

	// validate on "test". will validate field "name"
	v.ResetResult()
	v.Validate("test")
	is.True(v.IsOK())
	is.Equal("val0", v.SafeVal("key0"))
	is.Equal("inhere-HI", v.SafeVal("name"))
}
