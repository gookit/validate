package validate

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
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

var mpSample = M{
	"age":   100,
	"name":  "inhere",
	"oldSt": 1,
	"newSt": 2,
	"email": "some@e.com",
}

func TestMap(t *testing.T) {
	is := assert.New(t)

	v := New(mpSample)
	v.AddRule("name", "required")
	v.AddRule("name", "minLen", 7)
	v.AddRule("age", "max", 99)
	v.AddRule("age", "min", 1)

	ok := v.Validate()
	is.False(ok)
	is.Equal("name min length is 7", v.Errors.Get("name"))
	is.Empty(v.SafeData())

	v = New(nil)
	is.Contains(v.Errors.String(), "invalid input data")
	is.False(v.Validate())

	// test panic
	v1 := New(mpSample)
	is.Panics(func() {
		// Max(val, max) only one arg
		v1.AddRule("age", "max", 99, 34)
		v1.Validate()
	})

	v = New(mpSample)
	// invalid args
	v.AddRule("age", "max", nil)
	// v.AddRule("age", "max", []string{"a"})
	is.False(v.Validate())
	is.Contains(v.Errors.String(), "cannot convert invalid to int64")

	v = New(mpSample)
	v.StringRule("newSt", "gtField:oldSt")
	v.StringRule("newSt", "gteField:oldSt")
	v.StringRule("newSt", "neField:oldSt")
	v.StringRule("oldSt", "ltField:newSt")
	v.StringRule("oldSt", "lteField:newSt")
	is.True(v.Validate())
}

func TestValidation_StringRule(t *testing.T) {
	is := assert.New(t)

	v := Map(mpSample)
	v.StringRules(MS{
		"name":  "string|len:6|minLen:2|maxLen:10",
		"oldSt": "lt:5|gt:0|in:1,2,3|notIn:4,5",
		"age":   "max:100",
	})
	v.StringRule("newSt", "required|int:1|gtField:oldSt")
	ok := v.Validate()
	is.True(ok)

	v = Map(M{
		"oldSt": 2,
		"newSt": 2,
	})
	v.StringRule("oldSt", "eqField:newSt")
	v.StringRule("newSt", "required|int:1,5")
	is.True(v.Validate())
	is.Equal("", v.Errors.One())
}

func TestErrorMessages(t *testing.T) {
	is := assert.New(t)

	v := Map(mpSample)
	v.AddRule("name", "minLen", 8).SetMessage("custom msg0")
	is.False(v.Validate())
	is.Equal("custom msg0", v.Errors.One())

	v = Map(mpSample)
	v.StopOnError = false
	v.AddRule("oldSt, newSt", "min", 3).SetMessages(MS{
		"oldSt": "oldSt's err msg",
		"newSt": "newSt's err msg",
	})

	is.False(v.Validate())
	is.Equal("oldSt's err msg", v.Errors.Get("oldSt"))
	is.Equal("newSt's err msg", v.Errors.Get("newSt"))
}

// UserForm struct
type UserForm struct {
	Name      string    `validate:"required|minLen:7"`
	Email     string    `validate:"email"`
	CreateAt  int       `validate:"email"`
	Safe      int       `validate:"-"`
	UpdateAt  time.Time `validate:"required"`
	Code      string    `validate:"customValidator"`
	Status    int       `validate:"required|gtField:Extra.Status1"`
	Extra     ExtraInfo `validate:"required"`
	protected string
}

// ExtraInfo data
type ExtraInfo struct {
	Github  string `validate:"required|url"` // tags is invalid
	Status1 int    `validate:"required|int"`
}

// custom validator in the source struct.
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

func (f UserForm) ConfigValidation(v *Validation) {
	v.AddTranslates(MS{
		"Safe": "Safe-Name",
	})
}

// Messages you can custom define validator error messages.
func (f UserForm) Messages() map[string]string {
	return MS{
		"required":      "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}

// Translates you can custom field translates.
func (f UserForm) Translates() map[string]string {
	return MS{
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
	// test trans
	v.Trans().AddMessage("custom", "message0")
	is.True(v.Trans().HasMessage("custom"))
	is.Contains(v.Trans().FieldMap(), "Name")

	// validate
	v.StopOnError = false
	ok := v.Validate()
	is.True(v.IsFail())
	is.False(ok)
	is.Equal("User Name min length is 7", v.Errors.Get("Name"))
	is.Equal("oh! the UpdateAt is required", v.Errors.Get("UpdateAt"))
	is.Equal("oh! the Extra is required", v.Errors.Get("Extra"))
	is.Empty(v.SafeData())
	is.Empty(v.FilteredData())
	// fmt.Println(v.Errors)

	u.Name = "new name"
	u.Status = 5
	u.Extra = ExtraInfo{"xxx", 4}
	u.UpdateAt = time.Now()
	v = Struct(u)
	v.Validate()

	is.True(v.IsOK())
	is.True(v.Validate())
}

func TestJSON(t *testing.T) {
	is := assert.New(t)

	jsonStr := `{
	"name": "inhere",
	"age": 100
}`
	v := JSON(jsonStr)

	v.StopOnError = false
	v.StringRules(MS{
		"name": "required|minLen:7",
		"age":  "required|int|range:1,99",
	})

	is.False(v.Validate())
	is.Empty(v.SafeData())

	is.Contains(v.Errors, "age")
	is.Contains(v.Errors, "name")
	is.Contains(v.Errors.String(), "name min length is 7")
	is.Contains(v.Errors.String(), "age value must be in the range 1 - 99")

	_, err := FromJSONBytes([]byte("invalid"))
	is.Error(err)
	d, err := FromJSONBytes([]byte(jsonStr))
	is.Nil(err)
	v = d.Create()
	v.StringRules(MS{
		"name": "string:6,12",
		"age":  "range:1,100",
	})
	// float to int
	fl := v.FilterRule("age", "int")
	is.Equal([]string{"age"}, fl.Fields())
	is.True(v.Validate())
	is.Equal(100, v.SafeData()["age"])
	is.Equal("inhere", v.SafeData()["name"])
}

func TestFromQuery(t *testing.T) {
	is := assert.New(t)

	data := url.Values{
		"name": []string{"inhere"},
		"age":  []string{"10"},
	}

	v := FromQuery(data).Create()
	v.StopOnError = false
	v.FilterRule("age", "int")
	v.StringRules(MS{
		"name": "required|minLen:7",
		"age":  "required|int|min:10",
	})

	is.False(v.Validate())
	is.Equal("name min length is 7", v.Errors.Field("name")[0])
	is.Empty(v.SafeData())

	v = FromQuery(data).Validation(fmt.Errorf("an error"))
	is.Equal("an error", v.Errors.One())
}

func TestRequest(t *testing.T) {
	is := assert.New(t)

	// =================== GET query data ===================
	r, _ := http.NewRequest("GET", "/users?page=1&size=10&name= inhere ", nil)
	v := Request(r)
	v.StringRule("page", "required|min:1", "int")
	// v.StringRule("status", "required|min:1")
	v.StringRule("size", "required|min:1", "int")
	v.StringRule("status", "min:1", "int")
	v.StringRule("name", "minLen:5", "trim|upper")
	v.Validate()
	is.True(v.IsOK())
	is.Equal(10, v.SafeVal("size"))
	is.Equal(1, v.SafeVal("page"))
	is.Equal(nil, v.SafeVal("status"))
	is.Equal("INHERE", v.SafeVal("name"))

	// POST =================== form data ===================
	body := strings.NewReader("name= inhere &age=50&remember=yes&email=eml@a.com")
	r, _ = http.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// create data
	d, err := FromRequest(r)
	is.Nil(err)
	// create validation
	v = d.Validation()
	v.FilterRules(MS{
		"age":      "trim|int",
		"name":     "trim|ucFirst",
		"remember": "bool",
	})
	v.StringRules(MS{
		"name":     "string|minLen:5",
		"age":      "int|min:1",
		"email":    "email",
		"remember": "-", // mark is safe. don't validate, collect to safe data.
	})
	v.Validate() // validate
	// fmt.Println(v.Errors, v.safeData)
	is.True(v.IsOK())

	val, ok := v.Safe("name")
	is.True(ok)
	is.Equal("Inhere", val)
	is.Equal(50, v.SafeVal("age"))
	is.Equal("eml@a.com", v.SafeVal("email"))
	is.Equal("Inhere", v.SafeVal("name"))
	is.Equal(true, v.SafeVal("remember"))

	// =================== POST file data form ===================
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	w, err := mw.CreateFormFile("file", "test.jpg")
	if is.NoError(err) {
		w.Write([]byte("\xFF\xD8\xFF")) // write file content
	}
	mw.WriteField("age", "24")
	mw.WriteField("name", "inhere")
	mw.Close()

	r, _ = http.NewRequest("POST", "/users", buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	// - create data
	d, err = FromRequest(r)
	is.NoError(err)
	fd, ok := d.(*FormData)
	is.True(ok)
	is.True(fd.HasFile("file"))
	is.Equal("inhere", fd.String("name"))
	is.Equal(24, fd.Int("age"))

	bts, err := fd.FileBytes("file")
	is.NoError(err)
	is.Equal([]byte("\xFF\xD8\xFF"), bts)
	bts, err = fd.FileBytes("not-exist")
	is.Nil(bts)
	is.NoError(err)
	is.Equal("", fd.FileMimeType("not-exist"))
	is.Equal("image/jpeg", fd.FileMimeType("file"))
	// - create validation
	v = d.Validation()
	v.StringRules(MS{
		"age":  "min:1",
		"file": "required|mimeTypes:image/jpeg,image/jpg",
	})
	v.AddRule("name", "alpha")
	v.AddRule("file", "file")
	v.AddRule("file", "image", "jpg", "jpeg")
	v.Validate()
	is.True(v.IsOK())
	fmt.Println(v.Errors)

	// =================== POST JSON body ===================
	body = strings.NewReader(`{
	"name": " inhere ",
	"age": 100
}`)
	r, _ = http.NewRequest("POST", "/users", body)
	r.Header.Set("Content-Type", "application/json")
	// - create data
	d, err = FromRequest(r)
	is.Nil(err)
	user := &struct {
		Age  int
		Name string
	}{}
	md, ok := d.(*MapData)
	is.True(ok)
	err = md.BindJSON(user)
	is.Nil(err)
	is.Equal(100, user.Age)
	is.Equal(" inhere ", user.Name)
	// - create validation
	v = d.Create()
	v.StringRule("name", "-", "trim|upper")
	v.Validate()
	is.True(v.IsOK())
	v.BindSafeData(user)
	is.Equal("INHERE", user.Name)
}

func TestValidationScene(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name": "inhere",
		"age":  100,
	}

	v := Map(mp)
	v.StopOnError = false
	v.StringRules(MS{
		"name": "minLen:7",
		"age":  "min:101",
	})
	v.WithScenarios(SValues{
		"create": []string{"name", "age"},
		"update": []string{"name"},
	})

	// on scene "create"
	ok := v.Validate("create")
	is.False(ok)
	is.False(v.Errors.Empty())
	is.Equal("create", v.Scene())
	is.Contains(v.Errors.Error(), "age")
	is.Contains(v.Errors.Error(), "name")

	// on scene "update"
	v.ResetResult()
	v.InScene("update")
	is.Equal("update", v.Scene())
	is.Equal([]string{"name"}, v.SceneFields())
	ok = v.Validate()
	is.False(ok)
	is.Contains(v.Errors, "name")
	is.NotContains(v.Errors, "age")
	is.Equal("", v.Errors.Get("age"))
	is.Equal("name min length is 7", v.Errors.One())
}

func TestAddValidator(t *testing.T) {
	is := assert.New(t)

	is.Panics(func() {
		AddValidator("myCheck", "invalid")
	})
	is.Panics(func() {
		AddValidator("bad-name", func() {})
	})
	is.Panics(func() {
		AddValidator("", func() {})
	})
	is.Panics(func() {
		AddValidator("myCheck", func() {})
	})
	is.Panics(func() {
		AddValidator("myCheck", func() bool { return false })
	})
	is.Panics(func() {
		AddValidator("myCheck", func(val interface{}) {})
	})

	is.Contains(Validators(), "min")

	AddValidator("myCheck0", func(val interface{}) bool {
		return true
	})
	AddValidators(M{
		"myCheck1": func(val interface{}) bool {
			return true
		},
	})

	v := Map(mpSample)
	is.True(v.HasValidator("int"))
	is.True(v.HasValidator("min"))
	is.True(v.HasValidator("myCheck0"))
	is.True(v.HasValidator("myCheck1"))
	is.False(v.HasValidator("myCheck"))

	is.Panics(func() {
		v.AddValidator("myFunc2", func() {})
	})

	v.AddValidator("myFunc3", func(val interface{}) bool {
		return true
	})
	v.AddValidators(M{
		"myFunc4": func(val interface{}) bool {
			return true
		},
	})
	is.True(v.HasValidator("myFunc3"))
	is.True(v.HasValidator("myFunc4"))
	is.False(v.HasValidator("myFunc2"))

	is.Contains(v.Validators(false), "gtField")
	is.Contains(v.Validators(false), "myFunc4")
	is.NotContains(v.Validators(false), "min")
	is.Contains(v.Validators(true), "min")
}
