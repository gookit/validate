package validate

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	is := assert.New(t)
	tests := []interface{}{
		"",
		nil,
		0,
		false,
		int8(0), int16(0), int32(0), int64(0),
		uint8(0), uint16(0), uint32(0), uint64(0),
		float32(0), float64(0),
		[]int{}, []string{},
		map[string]string{},
	}

	for _, val := range tests {
		is.True(IsEmpty(val))
	}

	is.True(ValueIsEmpty(reflect.ValueOf(nil)))
	is.True(ValueIsEmpty(reflect.ValueOf("")))

	type T struct{ v interface{} }
	rv := reflect.ValueOf(T{}).Field(0)
	is.True(ValueIsEmpty(rv))
}

func TestContains(t *testing.T) {
	is := assert.New(t)

	// Contains
	is.True(Contains("abc", "a"))
	is.True(Contains([]string{"a", "b", "c"}, "a"))
	is.True(Contains(map[int]string{1: "a", 2: "b", 3: "c"}, 2))
	is.False(Contains(345, "a"))

	// NotContains
	is.True(NotContains("abc", "d"))
	is.True(NotContains([]string{"a", "b", "c"}, "d"))
	is.True(NotContains(map[int]string{1: "a", 2: "b", 3: "c"}, 4))
}

// ------------------ type validator ------------------

func TestIntCheck(t *testing.T) {
	is := assert.New(t)

	// type check
	tests := []interface{}{
		2, -2,
		int8(2), int16(2), int32(2), int64(2),
		uint(2), uint8(2), uint16(2), uint32(2), uint64(2),
	}
	for _, item := range tests {
		is.True(IsInt(item))
	}
	is.False(IsInt(nil))
	is.False(IsInt("str"))
	is.False(IsInt(2.3))
	is.False(IsInt(float32(2.3)))
	is.False(IsInt(-2.3))
	is.False(IsInt([]int{}))
	is.False(IsInt([]int{2}))
	is.False(IsInt(map[string]int{"key": 2}))

	// with min and max value
	is.True(IsInt(5, 5))
	is.True(IsInt(5, 4))
	is.True(IsInt(5, 4, 6))
	is.False(IsInt(nil, 4, 6))
	is.False(IsInt("str", 4, 6))

	// IsUint
	cases := []interface{}{
		2,
		int8(2), int16(2), int32(2), int64(2),
		uint(2), uint8(2), uint16(2), uint32(2), uint64(2),
	}
	for _, item := range cases {
		is.True(IsUint(item))
	}
	is.True(IsUint("0"))
	is.True(IsUint("2"))
	is.False(IsUint("-2"))
	is.False(IsUint("2a"))
	is.False(IsUint([]int{2}))
}

func TestTypeCheck(t *testing.T) {
	is := assert.New(t)

	// IsBool
	is.True(IsBool(true))
	is.True(IsBool("1"))
	is.True(IsBool("true"))
	is.True(IsBool("false"))
	is.False(IsBool(123))

	// IsFloat
	is.True(IsFloat("3.4"))
	is.True(IsFloat("2"))
	is.False(IsFloat(""))
	is.False(IsFloat(2))
	is.False(IsFloat("ab"))
	is.False(IsFloat(nil))

	// IsString
	is.True(IsString("str"))
	is.False(IsString(nil))
	is.False(IsString(2))

	is.True(IsString("str", 3))
	is.True(IsString("str", 3, 5))
	is.False(IsString(nil, 4))
	is.False(IsString(3, 4))
	is.False(IsString("str", 4))
	is.False(IsString("str", 1, 2))

	// IsMap
	is.True(IsMap(map[string]int{}))
	is.True(IsMap(new(map[string]int)))
	is.True(IsMap(make(map[string]int)))
	is.True(IsMap(map[string]int{"key": 1}))
	is.False(IsMap(nil))
	is.False(IsMap([]string{}))

	// IsInts
	is.True(IsInts([]int{}))
	is.True(IsInts([]int{1}))
	is.True(IsInts([]int8{}))
	is.False(IsInts(nil))
	is.False(IsInts(map[string]int{}))

	// IsStrings
	is.True(IsStrings([]string{}))
	is.True(IsStrings([]string{"a"}))
	is.False(IsStrings(nil))
	is.False(IsStrings([]int{}))
	is.False(IsStrings(map[string]int{}))
}

// ------------------ value compare ------------------

func TestIsArray_IsSlice(t *testing.T) {
	is := assert.New(t)

	// IsArray
	is.True(IsArray([1]int{}))
	is.True(IsArray([1]string{}))
	is.False(IsArray(nil))
	is.True(IsArray([]string{}))
	is.True(IsArray(new([]string)))
	is.False(IsArray([]string{}, true))
	is.False(IsArray(new([]string), true))

	// IsSlice
	is.True(IsSlice([]byte{'a'}))
	is.True(IsSlice([]rune{'a'}))
	is.True(IsSlice([]string{}))
	is.True(IsSlice(new([]string)))
	is.True(IsSlice(make([]string, 1)))
	is.False(IsSlice(nil))
	is.False(IsSlice([1]string{}))
	is.False(IsSlice(new(map[string]int)))

}

// ------------------ value compare ------------------

func TestValueCompare(t *testing.T) {
	is := assert.New(t)

	// IsEqual
	tests := []interface{}{
		2,
		int8(2), int16(2), int32(2), int64(2),
		uint8(2), uint16(2), uint32(2), uint64(2),
	}
	for _, item := range tests {
		is.True(IsEqual(item, 2))
	}

	is.True(IsEqual(true, true))
	is.True(IsEqual(uint(2), uint64(2)))
	is.True(IsEqual(2, uint64(2)))
	is.True(IsEqual(float32(2), float64(2)))
	is.True(IsEqual(nil, nil))
	is.True(IsEqual([]byte("abc"), []byte("abc")))

	// -- array, slice, map ...
	is.True(IsEqual([1]int{1}, [1]int{1}))
	is.True(IsEqual([]int{1}, []int{1}))
	is.True(IsEqual([]byte(`abc`), []byte(`abc`)))
	is.True(IsEqual([]string{"a"}, []string{"a"}))
	is.True(IsEqual([]interface{}{"a"}, []interface{}{"a"}))
	is.True(IsEqual(map[string]string{"a": "v0"}, map[string]string{"a": "v0"}))

	is.False(IsEqual(2, "2"))
	is.False(IsEqual(2, nil))
	is.False(IsEqual(nil, 2))
	is.False(IsEqual(func() {}, func() {}))
	is.False(IsEqual(2, func() {}))
	is.False(IsEqual([]byte(`abc`), "abc"))
	is.False(IsEqual(complex64(1+2i), "abc"))
	is.False(IsEqual(complex128(1+2), complex128(1+1)))

	// NotEqual
	is.True(NotEqual(2, nil))
	is.False(NotEqual(2, 2))

	// IntEqual
	is.True(IntEqual(2, 2))
	is.True(IntEqual("2", 2))
	is.False(IntEqual("a", 97))
	is.False(IntEqual("invalid", 2))

	// Between
	is.True(Between(3, 2, 5))
	is.True(Between("3", 2, 5))
	is.False(Between(6, 2, 5))
	is.False(Between("invalid", 2, 5))
}

func TestLtGt(t *testing.T) {
	is := assert.New(t)

	// Lt
	is.True(Lt(2, 3))
	is.True(Lt(2.1, 3))
	is.True(Lt(0.1, 0.3))
	is.False(Lt(3, 2))
	is.False(Lt(0.1, "invalid"))
	is.False(Lt(float32(0.1), "invalid"))
	is.False(Lt("invalid", 3))

	// Gt
	is.True(Gt(3, 2))
	is.True(Gt(0.3, 0.2))
	is.True(Gt(2.1, 2))
	is.False(Gt(2, 3))
	is.False(Gt([]int{23}, 3))
}

func TestMin(t *testing.T) {
	is := assert.New(t)

	// ok
	tests := []struct{ val, min interface{} }{
		{val: 3, min: 2},
		{val: 3, min: 3},
		{val: int64(3), min: 3},
		{val: 3.2, min: 3.1},
		{val: float32(3.2), min: 3.1},
		{val: 3.2, min: 3.2},
		{val: 3.2, min: "3.2"},
		{val: 3, min: 3.2},
		{val: 0.02, min: 0.01},
		{val: 0.02, min: 0.02},
	}
	for _, e := range tests {
		is.True(Min(e.val, e.min), "error: %#v should >= %#v", e.val, e.min)
	}

	// fail
	tests = []struct{ val, min interface{} }{
		{val: 3.1, min: 3.2},
		{nil, 3},
		{"abc", "def"},
		{3, nil},
		{3, 4},
		{3, "abc"},
		{int64(3), 4},
	}
	for _, e := range tests {
		is.False(Min(e.val, e.min), "error: %#v should not >= %#v", e.val, e.min)
	}
}

func TestMax(t *testing.T) {
	is := assert.New(t)

	// ok
	is.True(Max(3, 4))
	is.True(Max(3, 3))
	is.True(Max(3.2, 3.2))
	is.True(Max(3.1, 3.2))
	is.True(Max(int64(3), 3))
	is.True(Lte(int64(3), 3))

	// fail
	is.False(Max("str", 3))
	is.False(Max(3, 2))
	// since 1.3.2+ Max, Min input nil will always return FALSE.
	is.False(Max(nil, 3))
	is.False(Max(3, nil))
	is.False(Max(int64(3), 2))
}

// ------------------ string check ------------------

func TestStringCheck(t *testing.T) {
	is := assert.New(t)

	// IsAlpha
	is.True(IsAlpha("abc"))
	is.False(IsAlpha("abc123"))
	is.False(IsAlpha(""))

	// IsASCII
	is.True(IsASCII("abc"))
	is.True(IsASCII("#$"))
	is.False(IsASCII(""))
	is.False(IsASCII("中文"))

	// IsPrintableASCII
	is.True(IsPrintableASCII("abc"))
	is.False(IsPrintableASCII(""))
	is.False(IsPrintableASCII("中文"))

	// IsEmail
	is.True(IsEmail("some@abc.com"))
	is.False(IsEmail(""))
	is.False(IsEmail("some.abc.com"))
	// Add for issue#21
	is.False(IsEmail("ab@a1wa.c_m"))
	is.False(IsEmail("a@sina.c"))
	is.False(IsEmail("aaaa@qq.com."))

	// IsIP
	is.True(IsIP("127.0.0.1"))
	is.True(IsIP("1.1.1.1"))
	is.False(IsIP(""))
	is.False(IsIP("1.1.1.1.1"))

	// IsIPv4
	is.True(IsIPv4("127.0.0.1"))
	is.True(IsIPv4("1.1.1.1"))
	is.False(IsIPv4(""))
	is.False(IsIPv4("1.1.1.1.1"))

	// IsIPv6
	is.False(IsIPv6(""))
	is.False(IsIPv6("1.1.1.1"))

	// IsAlpha
	is.True(IsAlpha("abc"))
	is.True(IsAlpha("Abc"))
	is.False(IsAlpha(""))
	is.False(IsAlpha("#$"))
	is.False(IsAlpha("a bc"))
	is.False(IsAlpha("1232"))
	is.False(IsAlpha("1ab"))

	// IsAlphaNum
	is.True(IsAlphaNum("123abc"))
	is.True(IsAlphaNum("abc123"))
	is.True(IsAlphaNum("123"))
	is.True(IsAlphaNum("abc"))
	is.False(IsAlphaNum(""))
	is.False(IsAlphaNum("#$"))
	is.False(IsAlphaNum("123 abc"))

	// IsAlphaDash
	is.True(IsAlphaDash("abc"))
	is.False(IsAlphaDash(""))
	is.False(IsAlphaDash("123 abc"))

	// IsNumber
	is.True(IsNumber("0"))
	is.True(IsNumber("123"))
	is.False(IsNumber(nil))
	is.False(IsNumber([]int{2}))
	is.False(IsNumber(""))
	is.False(IsNumber("-123"))
	is.False(IsNumber("-123"))

	// IsNumeric
	is.True(IsNumeric("0"))
	is.True(IsNumeric("123"))
	is.False(IsNumeric(nil))
	is.False(IsNumeric([]int{2}))
	is.False(IsNumeric(""))
	is.False(IsNumeric("-123"))
	is.False(IsNumeric("-123"))

	// IsMultiByte
	is.True(IsMultiByte("你好"))
	is.False(IsMultiByte("hello"))

	// IsBase64
	is.True(IsBase64("dGhpcyBpcyBhIGV4YW1wbGU=")) // -> "this is a example"
	is.False(IsBase64("="))

	// IsDNSName
	is.True(IsDNSName("8.8.8.8"))
	is.False(IsDNSName(""))

	// IsMAC
	is.True(IsMAC("01:23:45:67:89:ab"))
	is.False(IsMAC(""))
	is.False(IsMAC("123 abc"))

	// IsCIDR
	is.True(IsCIDR("192.0.2.0/24"))
	is.True(IsCIDR("2001:db8::/32"))
	is.False(IsCIDR(""))

	// IsCIDRv4
	is.True(IsCIDRv4("192.0.2.0/24"))
	is.False(IsCIDRv4(""))

	// IsCIDRv6
	is.True(IsCIDRv6("2001:db8::/32"))
	is.False(IsCIDRv6(""))

	// HasWhitespace
	is.True(HasWhitespace("a bc"))
	is.False(HasWhitespace(""))
	is.False(HasWhitespace("abc"))

	// IsHexadecimal
	is.True(IsHexadecimal("0a23"))
	is.False(IsHexadecimal(""))

	// IsCnMobile
	is.True(IsCnMobile("13677778888"))
	is.False(IsCnMobile("136777888"))

	// IsHexColor
	is.True(IsHexColor("ccc"))
	is.True(IsHexColor("#ccc"))
	is.True(IsHexColor("ababab"))
	is.True(IsHexColor("#ababab"))
	is.False(IsHexColor(""))

	// IsRGBColor
	is.True(IsRGBColor("rgb(23,123,255)"))
	is.False(IsRGBColor(""))
	is.False(IsRGBColor("rgb(23,123,355)"))

	// UUID
	is.True(IsUUID("fd2fff4c-cc39-11e8-a8d5-f2801f1b9fd1"))
	is.False(IsUUID(""))

	// UUID3
	is.True(IsUUID("e0f98f02-6703-365c-9a42-4a0749f76068"))
	is.True(IsUUID3("e0f98f02-6703-365c-9a42-4a0749f76068"))
	is.False(IsUUID3(""))

	// UUID4
	is.True(IsUUID("8098f6fb-1557-4633-b82b-40e1b26137bf"))
	is.True(IsUUID4("8098f6fb-1557-4633-b82b-40e1b26137bf"))
	is.False(IsUUID4("fd2fff4c-cc39-11e8-a8d5-f2801f1b9fd1")) // uuid 1
	is.False(IsUUID4(""))

	// UUID5
	is.True(IsUUID("f6785639-778b-5db8-b1b3-60962fb4f38d"))
	is.True(IsUUID5("f6785639-778b-5db8-b1b3-60962fb4f38d"))
	is.False(IsUUID5(""))

	// IsLatitude
	is.True(IsLatitude("29.8431681298"))
	is.False(IsLatitude(""))

	// IsLongitude
	is.True(IsLongitude("102.3908204650"))
	is.False(IsLongitude(""))

	// IsIntString
	is.True(IsIntString("123"))
	is.False(IsIntString(""))
	is.False(IsIntString("a123"))

	// Regexp
	is.True(Regexp("123", "[0-9]+"))
}

func TestStringCheck_Case(t *testing.T) {
	is := assert.New(t)

	// HasLowerCase
	is.True(HasLowerCase("abc"))
	is.True(HasLowerCase("abC"))
	is.False(HasLowerCase(""))
	is.False(HasLowerCase("123"))
	is.False(HasLowerCase("ABC"))

	// HasUpperCase
	is.True(HasUpperCase("ABC"))
	is.True(HasUpperCase("Abc"))
	is.False(HasUpperCase("abc"))
	is.False(HasUpperCase(""))
	is.False(HasUpperCase("123"))

}

func TestStringCheck_ISBN(t *testing.T) {
	is := assert.New(t)

	// IsISBN10
	is.True(IsISBN10("0596528310"))
	is.False(IsISBN10(""))

	// IsISBN13
	is.True(IsISBN13("9780596528317"))
	is.False(IsISBN13(""))
}

func TestStringContains(t *testing.T) {
	// StringContains
	assert.True(t, StringContains("abc123", "123"))
	assert.False(t, StringContains("", "1234"))
	assert.False(t, StringContains("abc123", "1234"))

	// StartsWith
	assert.True(t, StartsWith("abc123", "abc"))
	assert.False(t, StartsWith("", "123"))
	assert.False(t, StartsWith("abc123", "123"))

	// EndsWith
	assert.True(t, EndsWith("abc123", "123"))
	assert.False(t, EndsWith("", "abc"))
	assert.False(t, EndsWith("abc123", "abc"))
}

func TestURLString(t *testing.T) {
	is := assert.New(t)

	// HasURLSchema
	is.True(HasURLSchema("http://a.com"))
	is.False(HasURLSchema("abd://a.com"))
	is.False(HasURLSchema("/ab/cd"))

	// IsURL
	is.True(IsURL("a.com?p=1"))
	is.True(IsURL("http://a.com?p=1"))
	is.True(IsURL("/users/profile/1"))
	is.True(IsURL("123"))
	is.False(IsURL(""))

	// IsDataURI
	is.True(IsDataURI("data:image/gif;base64,AB...CD..."))
	is.False(IsDataURI(""))
}

func TestIsFullURL(t *testing.T) {
	is := assert.New(t)

	okTests := []string{
		"http://a.com?p=1",
		"http://a.com?p=1&c=b",
		"http://a.com/ab/index",
		"http://a.com/ab/index?p=1&c=b",
		"http://www.a.com",
		"https://www.a.com",
		"http://www.a.com/ab/index?p=1&c=b",
		"https://www.google.com/testme",
		"https://www.google.com/test-me",
		"https://www.google.com/test_me",
		"https://www.google.com/test%",
		"http://www.google.com/test?a=2%25a&b=c",
	}
	for _, str := range okTests {
		is.True(IsFullURL(str), str)
	}

	failTests := []string{
		"",
		"a.com",
		"a.com/ab/c",
		"www.a.com",
		"www.a.com?a=1",
		"/users/profile/1",
		"www.google.com/test%",
	}
	for _, str := range failTests {
		is.False(IsFullURL(str))
	}
}

func TestPath(t *testing.T) {
	is := assert.New(t)

	// IsWinPath
	is.True(IsWinPath(`c:\users\inhere`))
	is.False(IsWinPath(`c:/users/inhere`))

	// IsUnixPath
	is.True(IsUnixPath("/users/inhere"))

	// IsDirPath
	is.True(IsDirPath("./"))
	is.False(IsDirPath("./testdata/test.txt"))

	is.True(PathExists("./testdata/test.txt"))

	// IsFilePath
	is.True(IsFilePath("./testdata/test.txt"))
	is.False(IsFilePath("./testdata/not-exist.txt"))
	is.False(IsFilePath(""))
}

func TestIsJSON(t *testing.T) {
	is := assert.New(t)

	// IsJSON
	is.True(IsJSON(`{"key": "value"}`))
	is.True(IsJSON(`["a", "b"]`))
	is.False(IsJSON("string"))
	is.False(IsJSON(""))
}

func TestCalcLength(t *testing.T) {
	is := assert.New(t)

	tests := []struct {
		sample string
		want   int
	}{
		{"a", 1},
		{"ab", 2},
		{"12ab", 4},
		{"+-ab", 4},
		{"ab你好", 4},
	}
	for _, item := range tests {
		is.Equal(CalcLength(item.sample), item.want)
	}
	is.Equal(CalcLength(nil), -1)

	ptrStr := "abc"
	is.Equal(CalcLength(&ptrStr), 3)
	ptrStr = "ab你好"
	is.Equal(CalcLength(&ptrStr), 4)
}

func TestLength(t *testing.T) {
	is := assert.New(t)

	// Length
	is.True(Length("a", 1))
	is.True(Length("ab", 2))
	is.True(Length([]int{1, 2}, 2))
	is.True(Length([]string{"a", "b"}, 2))
	is.True(Length("a中文", 3))
	is.False(Length("a中文", 7))
	is.False(Length(nil, 3))

	// ByteLength
	is.True(ByteLength("a", 1))
	is.True(ByteLength("abc", 1, 3))

	// RuneLength
	is.True(RuneLength("a", 1))
	is.True(StringLength("a中文", 3))
	is.True(StringLength("a中文", 3, 6))
	is.False(RuneLength(23, 2))
	// fmt.Println(len([]rune("a中文")))

	// MinLength
	is.True(MinLength("abc", 3))
	is.False(MinLength(nil, 3))

	// MaxLength
	is.True(MaxLength("abc", 5))
	is.False(MaxLength(nil, 5))
}

func TestEnumAndNotIn(t *testing.T) {
	is := assert.New(t)
	tests := map[interface{}]interface{}{
		1:   []int{1, 2, 3},
		2:   []int8{1, 2, 3},
		3:   []int16{1, 2, 3},
		4:   []int32{4, 2, 3},
		5:   []int64{5, 2, 3},
		6:   []uint{6, 2, 3},
		7:   []uint8{7, 2, 3},
		8:   []uint16{8, 2, 3},
		9:   []uint32{9, 2, 3},
		10:  []uint64{10, 3},
		11:  []string{"11", "3"},
		'a': []int64{97},
		'b': []rune{'a', 'b'},
		'c': []byte{'a', 'b', 'c'}, // byte -> uint8
		"a": []string{"a", "b", "c"},
	}

	for val, list := range tests {
		is.True(Enum(val, list))
		is.False(NotIn(val, list))
	}

	is.True(Enum(uint(2), []int{2, 3}))
	val := "a"
	is.True(Enum(&val, []string{"a", "b"}))

	is.False(Enum(nil, []int{}))
	is.False(Enum('a', []int{}))
	is.False(Enum([]int{2}, []int{2, 3}))
	is.False(Enum(12, []string{"a", "b"}))
	is.False(Enum(12, nil))
	is.False(Enum(12, map[int]int{2: 3}))

	tests1 := map[interface{}]interface{}{
		2:   []int{1, 3},
		"a": []string{"b", "c"},
	}

	for val, list := range tests1 {
		is.True(NotIn(val, list))
		is.False(Enum(val, list))
	}
}

func TestDateCheck(t *testing.T) {
	is := assert.New(t)
	// Date
	is.True(IsDate("2018-10-25"))

	// DateFormat
	is.True(DateFormat("2018-10-25", "2006-01-02"))
	is.True(DateFormat("2018-10-25 23:34:45", "2006-01-02 15:04:05"))

	// BeforeDate
	is.True(BeforeDate("2018-10-25", "2018-10-26"))
	is.False(BeforeDate("2018-10-26", "2018-10-26"))
	is.False(BeforeDate("2018-10-26", "invalid"))
	is.False(BeforeDate("invalid", "2018-10-26"))

	// BeforeOrEqualDate
	is.True(BeforeOrEqualDate("2018-10-25", "2018-10-26"))
	is.True(BeforeOrEqualDate("2018-10-26", "2018-10-26"))
	is.False(BeforeOrEqualDate("2018-10-27", "2018-10-26"))
	is.False(BeforeOrEqualDate("2018-10-27", "invalid"))
	is.False(BeforeOrEqualDate("invalid", "2018-10-26"))

	// AfterDate
	is.True(AfterDate("2018-10-26", "2018-10-25"))
	is.False(AfterDate("2018-10-26", "2018-10-26"))
	is.False(AfterDate("invalid", "2018-10-26"))
	is.False(AfterDate("2018-10-26", "invalid"))

	// AfterOrEqualDate
	is.True(AfterOrEqualDate("2018-10-27", "2018-10-26"))
	is.True(AfterOrEqualDate("2018-10-26", "2018-10-26"))
	is.False(AfterOrEqualDate("2018-10-25", "2018-10-26"))
	is.False(AfterOrEqualDate("invalid", "2018-10-26"))
	is.False(AfterOrEqualDate("2018-10-25", "invalid"))
}
