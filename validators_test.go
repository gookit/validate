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
	is.True(IsBool("1"))
	is.True(IsBool("true"))
	is.True(IsBool("false"))
	is.False(IsBool(123))

	// IsFloat
	is.True(IsFloat("3.4"))
	is.True(IsFloat("2"))
	is.False(IsFloat(""))
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

	// IsArray
	is.True(IsArray([1]int{}))
	is.True(IsArray([1]string{}))
	is.False(IsArray(nil))
	is.False(IsArray([]string{}))
	is.False(IsArray(new([]string)))

	// IsSlice
	is.True(IsSlice([]byte{'a'}))
	is.True(IsSlice([]rune{'a'}))
	is.True(IsSlice([]string{}))
	is.True(IsSlice(new([]string)))
	is.True(IsSlice(make([]string, 1)))
	is.False(IsSlice(nil))
	is.False(IsSlice([1]string{}))
	is.False(IsSlice(new(map[string]int)))

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

	// Gt
	is.True(Gt(3, 2))
	is.False(Gt(2, 3))
	is.False(Gt("invalid", 3))

	// Lt
	is.True(Lt(2, 3))
	is.False(Lt(3, 2))
	is.False(Lt("invalid", 3))

	// Between
	is.True(Between(3, 2, 5))
	is.True(Between("3", 2, 5))
	is.False(Between(6, 2, 5))
	is.False(Between("invalid", 2, 5))
}

func TestMin(t *testing.T) {
	is := assert.New(t)

	// ok
	is.True(Min(3, 2))
	is.True(Min(3, 3))
	is.True(Min(int64(3), 3))

	// fail
	is.False(Min(nil, 3))
	is.False(Min("str", 3))
	is.False(Min(3, 4))
	is.False(Min(int64(3), 4))
}

func TestMax(t *testing.T) {
	is := assert.New(t)

	// ok
	is.True(Max(3, 4))
	is.True(Max(3, 3))
	is.True(Max(int64(3), 3))
	// nil will as 0
	is.True(Max(nil, 3))

	// fail
	is.False(Max("str", 3))
	is.False(Max(3, 2))
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
	is.False(IsNumber(""))
	is.False(IsNumber("-123"))

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

	// IsISBN10
	is.True(IsISBN10("0596528310"))
	is.False(IsISBN10(""))

	// IsISBN13
	is.True(IsISBN13("9780596528317"))
	is.False(IsISBN13(""))

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

	// HasLowerCase
	is.True(HasLowerCase("abc"))
	is.True(HasLowerCase("abC"))
	is.False(HasLowerCase("123"))
	is.False(HasLowerCase("ABC"))

	// HasUpperCase
	is.True(HasUpperCase("ABC"))
	is.True(HasUpperCase("Abc"))
	is.False(HasUpperCase("abc"))
	is.False(HasUpperCase("123"))

	// Regexp
	is.True(Regexp("123", "[0-9]+"))
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

	// IsFullURL
	is.True(IsFullURL("http://a.com?p=1"))
	is.True(IsFullURL("http://www.a.com"))
	is.True(IsFullURL("https://www.a.com"))
	is.True(IsFullURL("http://a.com?p=1&c=b"))
	is.True(IsFullURL("http://a.com/ab/index"))
	is.True(IsFullURL("http://a.com/ab/index?p=1&c=b"))
	is.True(IsFullURL("http://www.a.com/ab/index?p=1&c=b"))
	is.False(IsFullURL(""))
	is.False(IsFullURL("a.com"))
	is.False(IsFullURL("a.com/ab/c"))
	is.False(IsFullURL("www.a.com"))
	is.False(IsFullURL("www.a.com?a=1"))
	is.False(IsFullURL("/users/profile/1"))

	// IsDataURI
	is.True(IsDataURI("data:image/gif;base64,AB...CD..."))
	is.False(IsDataURI(""))
}

func TestPath(t *testing.T) {
	is := assert.New(t)

	// IsWinPath
	is.True(IsWinPath(`c:\users\inhere`))
	is.False(IsWinPath(`c:/users/inhere`))

	// IsUnixPath
	is.True(IsUnixPath("/users/inhere"))

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

func TestLength(t *testing.T) {
	is := assert.New(t)

	// Length
	is.True(Length("a", 1))
	is.True(Length("ab", 2))
	is.True(Length([]int{1, 2}, 2))
	is.True(Length([]string{"a", "b"}, 2))
	is.True(Length("a中文", 7))
	is.False(Length("a中文", 3))
	is.False(Length(nil, 3))

	// ByteLength
	is.True(ByteLength("a", 1))
	is.True(ByteLength("abc", 1, 3))

	// RuneLength
	is.True(RuneLength("a", 1))
	is.True(StringLength("a中文", 3))
	is.True(StringLength("a中文", 3, 6))
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

	is.False(Enum(nil, []int{}))
	is.False(Enum('a', []int{}))
	//
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
