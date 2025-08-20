package validate

import (
	"fmt"
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

type TestPermission struct {
	TestUserData `json:",inline" validate:"required_if:Type,give"`
	Type         string `json:"type" validate:"required|in:give,remove"`
	Access       string `json:"access" validate:"required_if:Type,remove"`
}

type TestUserData struct {
	TestNameField   `json:",inline"`
	TestBranchField `json:",inline"`
}

type TestNameField struct {
	Name string `json:"name" validate:"required|max_len:5000"`
}

type TestBranchField struct {
	Branch string `json:"branch" validate:"required|min_len:32|max_len:32"`
}

func TestEmbeddedStructRequiredIf(t *testing.T) {
	// Test case 1: Type is "give", should validate UserData fields
	perm1 := TestPermission{
		TestUserData: TestUserData{},
		Type:         "give",
	}

	v1 := Struct(perm1)
	v1.StopOnError = false
	assert.False(t, v1.Validate())
	fmt.Println("perm1 errors (expected to fail):", v1.Errors.All())
	
	// Should have errors for UserData and its nested fields
	assert.True(t, v1.Errors.HasField("TestUserData"))

	// Test case 2: Type is "remove", should NOT validate UserData fields
	perm2 := TestPermission{
		Type:   "remove",
		Access: "change_types",
	}
	v2 := Struct(perm2)
	v2.StopOnError = false
	if !v2.Validate() {
		fmt.Println("perm2 errors (should be empty but currently fails):", v2.Errors.All())
		// This was the bug - it should validate successfully but doesn't
		t.Errorf("perm2 should validate successfully when Type=remove, but got errors: %v", v2.Errors.All())
	} else {
		fmt.Println("perm2: No errors (expected)")
	}
	// This should now pass with our fix
	assert.True(t, v2.Validate())

	// Test case 3: Type is "give" with valid UserData, should pass
	perm3 := TestPermission{
		TestUserData: TestUserData{
			TestNameField:   TestNameField{Name: "test"},
			TestBranchField: TestBranchField{Branch: "12345678901234567890123456789012"},
		},
		Type: "give",
	}
	v3 := Struct(perm3)
	v3.StopOnError = false
	if !v3.Validate() {
		fmt.Println("perm3 errors (unexpected):", v3.Errors.All())
	} else {
		fmt.Println("perm3: No errors (expected)")
	}
	assert.True(t, v3.Validate())
}

// Test edge cases for embedded struct conditional validation
func TestEmbeddedStructRequiredIfEdgeCases(t *testing.T) {
	// Test case: required_unless
	type TestStruct struct {
		UserData2 TestUserData `validate:"required_unless:Mode,skip"`
		Mode      string       `validate:"required"`
	}

	// Mode is "skip", so UserData2 should not be required
	test1 := TestStruct{
		Mode: "skip",
	}
	v1 := Struct(test1)
	assert.True(t, v1.Validate(), "Should pass when Mode=skip")

	// Mode is "process", so UserData2 should be required
	test2 := TestStruct{
		Mode: "process",
	}
	v2 := Struct(test2)
	assert.False(t, v2.Validate(), "Should fail when Mode=process and UserData2 is empty")
}
