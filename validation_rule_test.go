package validate

import (
	"net/url"
	"testing"

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

func TestRule_SetBeforeFunc(t *testing.T) {

}
