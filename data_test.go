package validate

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
	"net/url"
	"testing"
	"time"
)

func TestFormData_Add(t *testing.T) {
	is := assert.New(t)

	d := FromURLValues(url.Values{
		"name":   {"inhere"},
		"age":    {"30"},
		"notify": {"true"},
		"money":  {"23.4"},
	})

	is.True(d.Has("notify"))
	is.False(d.Has("not-exist"))
	is.False(d.HasFile("file"))
	is.True(d.Bool("notify"))
	is.False(d.Bool("not-exist"))
	is.Equal(30, d.Int("age"))
	is.Equal(int64(30), d.MustInt64("age"))
	is.Equal(0, d.Int("not-exist"))
	is.Equal(23.4, d.Float("money"))
	is.Equal(float64(0), d.Float("not-exist"))
	is.Equal("inhere", d.String("name"))
	is.Equal("age=30&money=23.4&name=inhere&notify=true", d.Encode())

	d.Set("newKey", "strVal")
	is.Equal("strVal", d.String("newKey"))
	d.Set("newInt", 23)
	is.Equal(23, d.Int("newInt"))

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
	_, err := newStructData(time.Now())
	is.Error(err)
	_, err = newStructData("invalid")
	is.Error(err)

	u := &UserForm{
		Name:     "new name",
		Status:   3,
		UpdateAt: time.Now(),
		Extra:    ExtraInfo{"xxx", 2},
	}

	d, err := newStructData(u)
	is.Nil(err)

	// create validation
	v := d.Create(fmt.Errorf("a error"))
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
	err = d.Set("Name", "inhere")
	is.Nil(err)
	str, ok = d.Get("Name")
	is.True(ok)
	is.Equal("inhere", str)
}
