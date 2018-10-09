package validate

import (
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestFormData_Add(t *testing.T) {
	is := assert.New(t)

	d := FromURLValues(url.Values{
		"name":   {"inhere"},
		"age":    {"30"},
		"notify": {"true"},
		"money":  {"23.4"},
	})

	is.True(d.Bool("notify"))
	is.True(d.Has("notify"))
	is.Equal(30, d.Int("age"))
	is.Equal(int64(30), d.MustInt64("age"))
	is.Equal(0, d.Int("not-exist"))
	is.Equal(23.4, d.Float("money"))
	is.Equal("inhere", d.String("name"))
	is.Equal("age=30&money=23.4&name=inhere&notify=true", d.Encode())

	d.Set("newKey", "strVal")
	is.Equal("strVal", d.String("newKey"))
	d.Set("newInt", 23)
	is.Equal(23, d.Int("newInt"))
}
