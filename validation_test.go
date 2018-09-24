package validate

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
	"time"
)

func ExampleStruct() {
	u := &UserForm{
		Name: "inhere",
	}

	v := Struct(u)
	ok := v.Validate()

	fmt.Println(ok)
}

func TestMap(t *testing.T) {
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
	is.Equal("name min length is 7", v.Errors.Get("name"))
}

// UserForm struct
type UserForm struct {
	Name     string    `json:"name" validate:"required|minLen:7"`
	Email    string    `json:"email" validate:"email"`
	CreateAt int       `json:"createAt" validate:"email"`
	Safe     int       `json:"safe" validate:"-"`
	UpdateAt time.Time `json:"updateAt" validate:"required"`
	Code     string    `json:"code" validate:"customValidator"`
	Status   int       `json:"status" validate:"required|gtField:Extra.Status1"`
	Extra    ExtraInfo `validate:"required"`
}

// ExtraInfo data
type ExtraInfo struct {
	Github  string `validate:"required|url"`
	Status1 int    `validate:"required|int"`
}

// custom validator in the source struct.
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

func (f UserForm) ConfigValidation(v *Validation) {
	v.AddTranslates(SMap{
		"Safe": "Safe-Name",
	})
}

// Messages you can custom define validator error messages.
func (f UserForm) Messages() map[string]string {
	return SMap{
		"required":      "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}

// Translates you can custom field translates.
func (f UserForm) Translates() map[string]string {
	return SMap{
		"Name":  "User Name",
		"Email": "User Email",
	}
}

func TestStruct(t *testing.T) {
	is := assert.New(t)

	u := &UserForm{
		Name: "inhere",
	}
	v := Struct(u)

	// check trans data
	is.True(v.Trans().HasField("Name"))
	is.True(v.Trans().HasField("Safe"))
	is.True(v.Trans().HasMessage("Name.required"))

	// validate
	ok := v.Validate()
	is.False(ok)
	is.Equal("User Name min length is 7", v.Errors.Field("Name")[0])
}

func TestRequest(t *testing.T) {
	// r := http.NewRequest()
}

func TestFromQuery(t *testing.T) {
	is := assert.New(t)

	data := url.Values{
		"name": []string{"inhere"},
		"age":  []string{"10"},
	}

	v := FromQuery(data).Create()
	v.StopOnError = false
	v.StringRules(SMap{
		// "name": "required|minLen:7",
		"age": "int",
		// "age":  "required|int|min:20",
	})

	// v.AddRule("age", )

	is.False(v.Validate())
	fmt.Println(v.Errors)
}
