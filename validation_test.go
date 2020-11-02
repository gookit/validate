package validate

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	"items": []string{"a"},
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
	is.Equal("name min length is 7", v.Errors.FieldOne("name"))
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
	v.StringRule("newSt", "") // will ignore
	v.StringRule("newSt", "gtField:oldSt")
	v.StringRule("newSt", "gteField:oldSt")
	v.StringRule("newSt", "neField:oldSt")
	v.StringRule("oldSt", "ltField:newSt")
	v.StringRule("oldSt", "lteField:newSt")
	is.True(v.Validate())

	v = New(M{"age": 12.34})
	v.AddRule("age", "int")
	is.False(v.Validate())
	is.Equal("age value must be an integer", v.Errors.One())
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
	is.Equal("oldSt's err msg", v.Errors.FieldOne("oldSt"))
	is.Equal("newSt's err msg", v.Errors.FieldOne("newSt"))
	// test binding
	u := struct {
		Age  int
		Name string
	}{}
	err := v.BindSafeData(&u)
	is.Nil(err)
	is.Equal(0, u.Age)

	// context validators
	is.False(v.GtField([]int{2}, "age"))
	is.False(v.GtField(2, "items"))
	is.False(v.GtField("a", "items"))

	// SetMessages, key is "field.validator"
	v = Map(mpSample)
	v.AddRule("name", "int").SetMessages(MS{
		"name.int": "HH, value must be INTEGER",
	})
	is.False(v.Validate())
	is.Equal("HH, value must be INTEGER", v.Errors.FieldOne("name"))

	// AddMessages
	v = Map(mpSample)
	v.AddMessages(MS{"isInt": "value must be INTEGER!"})
	v.AddRule("name", "isInt")
	is.False(v.Validate())
	is.Equal("value must be INTEGER!", v.Errors.FieldOne("name"))
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
	is.Equal("User Name min length is 7", v.Errors.FieldOne("Name"))
	is.Equal("oh! the UpdateAt is required", v.Errors.FieldOne("UpdateAt"))
	is.Equal("oh! the Extra is required", v.Errors.FieldOne("Extra"))
	is.Empty(v.SafeData())
	is.Empty(v.FilteredData())

	u.Name = "new name"
	u.Status = 3
	u.Extra = ExtraInfo{"xxx", 4}
	u.UpdateAt = time.Now()
	v = Struct(u)
	is.False(v.Validate())
	is.Equal("Status value must be greater the field Extra.Status1", v.Errors.One())

	u.Status = 5
	u.Extra = ExtraInfo{"xxx", 4}
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
	v.ConfigRules(MS{
		"name": "required|minLen:7",
		"age":  "required|int|range:1,99",
	})

	is.False(v.Validate())
	is.Empty(v.SafeData())

	is.Contains(v.Errors, "age")
	is.Contains(v.Errors, "name")
	is.Contains(v.Errors.String(), "name min length is 7")
	is.Contains(v.Errors.String(), "age value must be in the range 1 - 99")

	// test set
	iv, ok := v.Raw("type")
	is.False(ok)
	is.Nil(iv)
	// set new field
	err := v.Set("type", 2)
	is.Nil(err)
	iv, ok = v.Raw("type")
	is.True(ok)
	is.Equal(2, iv)

	_, err = FromJSONBytes([]byte("invalid"))
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

	val, ok := v.Raw("name")
	is.True(ok)
	is.Equal("inhere", val)
	is.False(v.Validate())
	is.Equal("name min length is 7", v.Errors.FieldOne("name"))
	is.Empty(v.SafeData())

	v = FromQuery(data).Validation(fmt.Errorf("an error"))
	is.Equal("an error", v.Errors.One())

	// change global opts
	Config(func(opt *GlobalOption) {
		opt.StopOnError = false
		opt.SkipOnEmpty = false
	})
	v = FromQuery(data).Create()
	v.StringRules(MS{
		"name": "file",
		"age":  "image",
	})

	is.False(v.Validate())
	// revert
	gOpt.StopOnError = true
	gOpt.SkipOnEmpty = true
	is.Len(v.Errors, 2)
	is.Contains(v.Errors.All(), "age")
	is.Contains(v.Errors.All(), "name")
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

	// =================== POST: form data ===================
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
	fmt.Println(v.Errors)
	val, ok := v.Safe("name")
	is.True(ok)
	is.Equal("Inhere", val)
	is.Equal(50, v.SafeVal("age"))
	is.Equal("eml@a.com", v.SafeVal("email"))
	is.Equal("Inhere", v.SafeVal("name"))
	is.Equal(true, v.SafeVal("remember"))
}

func TestFromRequest_FileForm(t *testing.T) {
	is := assert.New(t)

	// =================== POST: file data form ===================
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	w, err := mw.CreateFormFile("file", "test.jpg")
	if is.NoError(err) {
		// write file content, this is jpg file start content
		_, _ = w.Write([]byte("\xFF\xD8\xFF"))
	}
	_ = mw.WriteField("age", "24")
	_ = mw.WriteField("name", "inhere")
	_ = mw.Close()

	r, _ := http.NewRequest("POST", "/users", buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	// - create data
	d, err := FromRequest(r, defaultMaxMemory)
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
	v := d.Validation()
	v.StringRules(MS{
		"age":  "min:1",
		"file": "required|mimeTypes:image/jpeg,image/jpg",
	})
	v.AddRule("name", "alpha")
	v.AddRule("file", "file")
	v.AddRule("file", "image", "jpg", "jpeg")
	v.AddRule("file", "mimeTypes", "image/jpeg")
	v.Validate()
	is.True(v.IsOK())

	ok = v.IsFormImage(fd, "file")
	is.True(ok)

	ok = v.InMimeTypes(fd, "not-exist", "image/jpg")
	is.False(ok)

	// clear rules
	v.Reset()
	v.AddRule("file", "mimes")
	is.PanicsWithValue("validate: not enough parameters for validator 'mimes'!", func() {
		v.Validate()
	})
}

func TestFromRequest_JSON(t *testing.T) {
	// =================== POST: JSON body ===================
	body := `{
		"name": " inhere ",
		"age": 100
	}`

	tests := []struct {
		name    string
		header  string
		body    string
		failure bool
	}{
		{
			name:   "valid JSON content type #1",
			header: "application/json",
			body:   body,
		},
		{
			name:   "valid JSON content type #2",
			header: "application/activity+json",
			body:   body,
		},
		{
			name:   "valid JSON content type #3",
			header: "application/geo+json-seq",
			body:   body,
		},
		{
			name:   "valid JSON content type #4",
			header: "application/json-patch+json",
			body:   body,
		},
		{
			name:   "valid JSON content type #5",
			header: "application/vnd.api+json",
			body:   body,
		},
		{
			name:   "valid JSON content type #6",
			header: "application/vnd.capasystems-pg+json",
			body:   body,
		},
		{
			name:   "valid JSON content type #7",
			header: "application/vnd.ims.lti.v2.toolconsumerprofile+json",
			body:   body,
		},
		{
			name:   "invalid JSON content type #1",
			header: "foo/bar+json-seq",
			body:   "",
		},
		{
			name:   "invalid JSON content type #2",
			header: "application/xml",
			body:   "",
		},
		{
			name:   "invalid JSON content type #3",
			header: "application/invalidjson",
			body:   "",
		},
		{
			name:   "invalid JSON content type #4",
			header: "application/invalid-json-seq",
			body:   "",
		},
		{
			name:   "invalid JSON content type #5",
			header: "application/+json",
			body:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			is := assert.New(t)

			r, _ := http.NewRequest("POST", "/users", strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.header)

			// - create data
			d, err := FromRequest(r)

			if test.body != "" {
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
				v := d.Create()
				v.StringRule("name", "-", "trim|upper")
				v.Validate() // validate
				is.True(v.IsOK())
				err = v.BindSafeData(user)
				is.NoError(err)
				is.Equal("INHERE", user.Name)
			} else {
				is.Nil(d)
				is.Error(err)
			}
		})
	}
}

func TestFieldCompare(t *testing.T) {
	is := assert.New(t)
	v := Map(mpSample)
	v.StopOnError = false
	v.StringRules(MS{
		"oldSt": "gteField:notExist|gtField:notExist",
		"newSt": "lteField:notExist|ltField:notExist",
		"name":  "neField:notExist",
		"age":   "eqField:notExist",
	})
	v.Validate()
	is.False(v.IsOK())
	emp := v.Errors.All()
	is.Len(emp, 4)
	is.Contains(emp, "age")
	is.Contains(emp, "name")
	is.Contains(emp, "oldSt")
	is.Contains(emp, "newSt")
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
	is.Equal("", v.Errors.FieldOne("age"))
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

func TestValidation_ValidateData(t *testing.T) {
	d := FromMap(M{
		"name": "inhere",
		"json": `{"k": "v"}`,
	})

	v := NewEmpty()
	v.StringRules(MS{
		"json": "json",
	})

	ok := v.ValidateData(d)
	assert.True(t, ok)

	v.Reset()
	assert.Len(t, v.Validators(false), 0)
}

func TestIssue2(t *testing.T) {
	type Fl struct {
		A float64 `validate:"float"`
	}

	fl := Fl{123}
	v := Struct(fl)
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
	v = Struct(&fl)
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
	assert.Equal(t, errConvertFail.Error(), err.Error())
}

func TestGetSetOnNilData(t *testing.T) {
	ris := assert.New(t)

	// custom new
	v := &Validation{}

	// Get
	val, ok := v.Get("age")
	ris.Nil(val)
	ris.False(ok)

	// Safe
	val, ok = v.Safe("age")
	ris.Nil(val)
	ris.False(ok)

	// Raw
	val, ok = v.Raw("age")
	ris.Nil(val)
	ris.False(ok)

	// Set
	err := v.Set("age", 12)
	ris.Error(err)
}

func TestBuiltInValidators(t *testing.T) {
	ris := assert.New(t)
	v := New(M{"age": "12"})
	v.StringRule("age", "isNumber")
	ris.True(v.Validate())
	v.Reset()

	v.StringRule("age", "isInt", "int")
	ris.True(v.Validate())
	ris.Equal(12, v.SafeVal("age"))
	v.Reset()

	ris.Panics(func() {
		v.StringRule("age", "not-exist")
		v.Validate()
	})
}
