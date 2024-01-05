package validate

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/gookit/goutil/testutil/assert"
)

func TestFilterOnStruct(t *testing.T) {
	ris := assert.New(t)
	u := &struct {
		// Age  int    `filter:"uint" validate:"int"`
		Tres string `filter:"upper" validate:"required|in:ONE,TWO,THREE" `
		Name string `filter:"upper" validate:"string"`
	}{
		"one", "inhere",
	}

	v := New(u)
	ris.True(v.Validate())

	// since 1.1.4 filtered value will update to source
	ris.Equal("ONE", u.Tres)
	ris.Equal("INHERE", u.Name)

	// bind filtering and validated data to struct
	err := v.BindSafeData(u)
	ris.Nil(err)
	ris.Equal("ONE", u.Tres)
	ris.Equal("INHERE", u.Name)
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
		AddFilter("myFilter", func() any { return nil })
	})

	AddFilters(M{
		"myFilter0": func(val any) string { return "myFilter0" },
	})
	AddFilter("myFilter1", func(val any) string { return "myFilter1" })

	v := New(map[string]any{
		"name": " inhere ",
		"age":  " 50 ",
		"key0": "val0",
		"key1": "val1",
		"tags": "go,php",
	})
	v.AddFilters(M{
		"myFilter2": func(val any, a, b string) (string, error) { return "myFilter2:" + a + b, nil },
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
	is.Equal("SOME field did not pass validation", v.Trans().Message("not-exist", "SOME"))
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
	is.Equal("age: report a error", v.Errors.FieldOne("_filter"))
}

// check panic caused nil value with custom filter
func TestFilterRuleNilValue(t *testing.T) {
	AddFilter("X", func(in any) any {
		return in
	})

	v := Map(map[string]any{
		"bad": nil,
	})
	v.FilterRule("bad", "X")

	assert.NotEmpty(t, v.data.Src())
	assert.NotPanics(t, func() {
		v.Validate()
	})
}
