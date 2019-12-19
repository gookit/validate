package validate

import (
	"net/url"
	"testing"

	"github.com/gookit/filter"
	"github.com/stretchr/testify/assert"
)

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
	r.SetBeforeFunc(func(v *Validation) bool {
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

func TestRule_SetBeforeFunc(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name":   "inhere",
		"avatar": "/some/file",
	}

	v := Map(mp)
	v.AddRule("avatar", "isFile")
	is.False(v.Validate())
	is.Equal("avatar must be an uploaded file", v.Errors.One())

	// use SetBeforeFunc
	v = Map(mp)
	v.
		AddRule("avatar", "isFile").
		SetBeforeFunc(func(v *Validation) bool {
			// return false for skip validate
			return false
		})

	v.Validate()
	is.True(v.IsOK())
}

func TestRule_SetFilterFunc(t *testing.T) {
	is := assert.New(t)
	v := Map(M{
		"name": "inhere",
		"age":  "abc",
	})

	v.
		AddRule("age", "int", 1, 100).
		SetFilterFunc(func(val interface{}) (i interface{}, e error) {
			return filter.Int(val)
		})

	is.False(v.Validate())
	is.Equal(`strconv.Atoi: parsing "abc": invalid syntax`, v.Errors.One())
}

func TestRule_SetSkipEmpty(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name": "inhere",
		"age":  0,
	}

	v := Map(mp)
	v.AddRule("age", "int", 1)
	v.AddRule("name", "string", 1, 10)
	is.True(v.Validate())
	sd := v.SafeData()
	is.Contains(sd, "name")
	is.NotContains(sd, "age")
	is.Equal("inhere", v.GetSafe("name"))
	is.Equal(nil, v.GetSafe("age"))

	v = Map(mp)
	v.AddRule("age", "int", 1).SetSkipEmpty(false)
	v.AddRule("name", "string", 1, 10)
	is.False(v.Validate())
	is.Equal("age value must be an integer and mix value is 1", v.Errors.One())
}
