package validate

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/testutil/assert"
)

// func TestMain(m *testing.M) {
// 	setup()
// 	code := m.Run()
// 	// shutdown()
// 	os.Exit(code)
// }
//
// func setup() {
// 	dump.Println("--------- setup ---------")
// 	StdTranslator.Reset()
// }

func ExampleStruct() {
	// UserForm struct
	type UserForm struct {
		Name      string      `validate:"required|minLen:7" message:"required:{field} is required" label:"User Name"`
		Email     string      `validate:"email" message:"email:input must be a EMAIL address"`
		CreateAt  int         `validate:"email"`
		Safe      int         `validate:"-"`
		UpdateAt  time.Time   `validate:"required"`
		Code      string      `validate:"customValidator|default:abc"`
		Status    int         `validate:"required|gtField:Extra.0.Status1"`
		Extra     []ExtraInfo `validate:"required"`
		protected string      //nolint:unused
	}

	u := &UserForm{
		Name: "inhere",
	}

	v := Struct(u)
	ok := v.Validate()

	fmt.Println(ok)
	dump.P(v.Errors, u)
}

func TestUtil_Func_valueToInt64(t *testing.T) {
	noErrTests := []struct {
		val    any
		strict bool
		want   int64
	}{
		{" 12", false, 12},
		{float32(12.23), false, 12},
		{12.23, false, 12},
	}

	for _, item := range noErrTests {
		i64, err := valueToInt64(item.val, item.strict)
		assert.NoError(t, err)
		assert.Equal(t, item.want, i64)
	}
}

func TestUtil_Func_getVariadicKind(t *testing.T) {
	noErrTests := []struct {
		val  any
		want reflect.Kind
	}{
		{"invalid", reflect.Invalid},
		{[]int{1, 2}, reflect.Int},
		{[]int8{1, 2}, reflect.Int8},
		{[]int16{1, 2}, reflect.Int16},
		{[]int32{1, 2}, reflect.Int32},
		{[]int64{1, 2}, reflect.Int64},
		{[]uint{1, 2}, reflect.Uint},
		{[]uint8{1, 2}, reflect.Uint8},
		{[]uint16{1, 2}, reflect.Uint16},
		{[]uint32{1, 2}, reflect.Uint32},
		{[]uint64{1, 2}, reflect.Uint64},
		{[]string{"a", "b"}, reflect.String},
	}

	for _, item := range noErrTests {
		eleType := getVariadicKind(reflect.TypeOf(item.val))
		assert.Equal(t, item.want, eleType)
	}
}

func TestUtil_Func_goodName(t *testing.T) {
	tests := []struct {
		give string
		want bool
	}{
		{"ab", true},
		{"1234", false},
		{"01234", false},
		{"abc123", true},
	}

	for _, item := range tests {
		assert.Equal(t, item.want, goodName(item.give))
	}
}

func Test_Util_Func_convertType(t *testing.T) {
	nVal, err := convTypeByBaseKind(23, reflect.String)
	assert.NoError(t, err)
	assert.Equal(t, "23", nVal)

	nVal, err = convTypeByBaseKind(uint(23), reflect.String)
	assert.NoError(t, err)
	assert.Equal(t, "23", nVal)

	nVal, err = convTypeByBaseKind([]byte("23"), reflect.String)
	assert.NoError(t, err)
	assert.Equal(t, "23", nVal)

	nVal, err = convTypeByBaseKind("23", reflect.Int)
	assert.NoError(t, err)
	assert.Equal(t, 23, nVal)

	// Stringer convert to string
	var val strings.Builder
	val.WriteString("23")
	nVal, err = convTypeByBaseKind(&val, reflect.String)
	assert.NoError(t, err)
	assert.Equal(t, "23", nVal)
}

func Test_IsZero(t *testing.T) {
	assert.True(t, IsZero(reflect.ValueOf([2]int{})))
	assert.True(t, IsZero(reflect.ValueOf(false)))
	assert.Panics(t, func() {
		IsZero(reflect.ValueOf(nil))
	})
}

func TestMS_String(t *testing.T) {
	ms := MS{}

	assert.Equal(t, "", ms.One())
	assert.Equal(t, "", ms.String())

	ms["key"] = "val"
	assert.Equal(t, "val", ms.One())
	assert.Equal(t, " key: val", ms.String())
}

func TestOption(t *testing.T) {
	opt := Option()

	assert.Equal(t, "json", opt.FieldTag)
	assert.Equal(t, "validate", opt.ValidateTag)

	Config(func(opt *GlobalOption) {
		opt.ValidateTag = "valid"
	})

	opt = Option()
	assert.Equal(t, "valid", opt.ValidateTag)

	ResetOption()
}

func TestStruct_nilPtr_field2(t *testing.T) {
	type UserDto struct {
		Name string `validate:"required"`
		Sex  *bool  `validate:"required" json:"sex"`
	}

	sex := true
	u := UserDto{
		Name: "abc",
		Sex:  nil,
	}

	v := Struct(&u)
	assert.False(t, v.Validate())
	assert.True(t, v.Errors.HasField("sex"))
	assert.Contains(t, v.Errors.FieldOne("sex"), "sex is required")
	dump.Println(v.Errors)

	u.Sex = &sex
	v = Struct(&u)
	assert.True(t, v.Validate())
}

func TestStruct_nexted_anonymity_struct(t *testing.T) {
	type UserDto struct {
		Name    string `validate:"required"`
		Sex     *bool  `validate:"required" json:"sex"`
		ExtInfo struct {
			Homepage string `validate:"required"`
			CityName string
		}
	}

	sex := true
	u := &UserDto{
		Name: "abc",
		Sex:  &sex,
	}

	v := Struct(u)
	assert.False(t, v.Validate())
	dump.Println(v.Errors)
	assert.True(t, v.Errors.HasField("ExtInfo.Homepage"))
	assert.Contains(t, v.Errors, "ExtInfo.Homepage")
	assert.Equal(t, "ExtInfo.Homepage is required to not be empty", v.Errors.FieldOne("ExtInfo.Homepage"))

	u.ExtInfo.Homepage = "https://github.com/inhere"
	v = Struct(u)
	assert.True(t, v.Validate())
}

func TestStruct_nexted_field_name_tag(t *testing.T) {
	type UserDto struct {
		Name    string `validate:"required" label:"Display-Name"`
		Sex     *bool  `validate:"required" json:"sex"`
		ExtInfo struct {
			Homepage string `validate:"required" json:"home_page"`
			CityName string
		} `json:"ext_info" label:"info"`
	}

	sex := true
	u := &UserDto{
		Name: "",
		Sex:  &sex,
	}
	v := Struct(u)
	v.StopOnError = false
	assert.False(t, v.Validate())

	dump.Println(v.Errors)
	assert.True(t, v.Errors.HasField("Name"))
	assert.True(t, v.Errors.HasField("ext_info.home_page"))
	assert.Contains(t, v.Errors, "Name")
	assert.Contains(t, v.Errors, "ext_info.home_page")

	nameErrStr := v.Errors["Name"]["required"]
	extHomeErrStr := v.Errors["ext_info.home_page"]["required"]
	assert.True(t, strings.HasPrefix(nameErrStr, "Display-Name"))
	assert.True(t, strings.HasPrefix(extHomeErrStr, "ext_info.home_page"))
}

func TestStruct_create_error(t *testing.T) {
	v := Struct(nil)
	assert.NotEmpty(t, v.Errors)
	assert.Equal(t, "invalid input data", v.Errors.One())
	assert.False(t, v.Validate())
}

func TestStruct_json_tag_name_parsing(t *testing.T) {
	// Ensure that the JSON tag after comma is ignored.
	type Thing struct {
		Field string `json:"test,omitempty" validate:"email"`
	}

	th := Thing{Field: "a"}

	v := Struct(th)
	assert.False(t, v.Validate())

	dump.Println(v.Errors)
	assert.True(t, v.Errors.HasField("test"))

	errStr := v.Errors["test"]["email"]
	assert.True(t, strings.HasPrefix(errStr, "test "))

	// Ensure that the field name is used if the JSON tag name is empty.
	type Thing2 struct {
		Field string `json:",omitempty" validate:"email"`
	}

	th2 := Thing2{Field: "a"}

	v = Struct(th2)
	assert.False(t, v.Validate())

	dump.Println(v.Errors)
	assert.True(t, v.Errors.HasField("Field"))

	errStr = v.Errors["Field"]["email"]
	assert.True(t, strings.HasPrefix(errStr, "Field "))
}

func TestValidation_RestoreRequestBody(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"test": "data"}`))
	assert.Nil(t, err)
	request.Header.Set("Content-Type", "application/json")

	data, err := FromRequest(request)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	bs, err := io.ReadAll(request.Body)
	assert.Nil(t, err)
	assert.Empty(t, bs)

	// restore body
	Config(func(opt *GlobalOption) {
		opt.RestoreRequestBody = true
	})

	request, err = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"test": "data"}`))
	assert.Nil(t, err)
	request.Header.Set("Content-Type", "application/json")

	data, err = FromRequest(request)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	bs, err = io.ReadAll(request.Body)
	assert.Nil(t, err)
	assert.Equal(t, `{"test": "data"}`, string(bs))

}
