package validate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/stretchr/testify/assert"
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

func TestUtil_Func_valueToInt64(t *testing.T) {
	noErrTests := []struct {
		val    interface{}
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
		val  interface{}
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
		vt := reflect.TypeOf(item.val)
		eleType := getVariadicKind(vt.String())
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
	nVal, err := convTypeByBaseKind(23, intKind, reflect.String)
	assert.NoError(t, err)
	assert.Equal(t, "23", nVal)

	nVal, err = convTypeByBaseKind(uint(23), uintKind, reflect.String)
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
	assert.Equal(t, "ExtInfo.Homepage is required and not empty", v.Errors.FieldOne("ExtInfo.Homepage"))

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
