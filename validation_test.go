package validate

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func ExampleStruct() {
	u := &userForm{
		Name: "inhere",
	}

	v := Struct(u)
	ok := v.Validate()

	fmt.Println(ok)
}

func TestValidation(t *testing.T) {
	is := assert.New(t)

	m := GMap{
		"name":  "inhere",
		"age":   100,
		"oldSt": 1,
		"newSt": 2,
		"email": "some@e.com",
	}

	v := New(m)

	v.AddRule("name", "required")
	v.AddRule("name", "minLen", 7)
	v.AddRule("age", "max", 99)
	v.AddRule("age", "min", 1)

	v.WithScenes(SValues{
		"create": []string{"name", "email"},
		"update": []string{"name"},
	})

	ok := v.Validate()
	is.False(ok)
	is.Equal("name value min length is 7", v.Errors.Get("name"))
}

type userForm struct {
	Name string `json:"name" validate:"required|minLen:7|customValidator"`
}

func (f userForm) CustomValidator(val string) bool {
	return false
}

func TestStruct(t *testing.T) {
	is := assert.New(t)
	u := &userForm{
		Name: "inhere",
	}

	v := Struct(u)
	ok := v.Validate()

	is.False(ok)
}