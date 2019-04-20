package validate

import (
	"bytes"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gookit/filter"
)

// Basic regular expressions for validating strings.(it is from package "asaskevich/govalidator")
const (
	Email        string = "^(((([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|((\\x22)((((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(([\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(\\([\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(((\\x20|\\x09)*(\\x0d\\x0a))?(\\x20|\\x09)+)?(\\x22)))@((([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|\\.|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])([a-zA-Z]|\\d|-|_|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*([a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
	UUID3        string = "^[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$"
	UUID4        string = "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	UUID5        string = "^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	UUID         string = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
	Int          string = "^(?:[-+]?(?:0|[1-9][0-9]*))$"
	Float        string = "^(?:[-+]?(?:[0-9]+))?(?:\\.[0-9]*)?(?:[eE][\\+\\-]?(?:[0-9]+))?$"
	RGBColor     string = "^rgb\\(\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*\\)$"
	FullWidth    string = "[^\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	HalfWidth    string = "[\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	Base64       string = "^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$"
	Latitude     string = "^[-+]?([1-8]?\\d(\\.\\d+)?|90(\\.0+)?)$"
	Longitude    string = "^[-+]?(180(\\.0+)?|((1[0-7]\\d)|([1-9]?\\d))(\\.\\d+)?)$"
	DNSName      string = `^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`
	URLSchema    string = `((ftp|tcp|udp|wss?|https?):\/\/)`
	URLUsername  string = `(\S+(:\S*)?@)`
	URLPath      string = `((\/|\?|#)[^\s]*)`
	URLPort      string = `(:(\d{1,5}))`
	URLIP        string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain string = `((www\.)|([a-zA-Z0-9]+([-_\.]?[a-zA-Z0-9])*[a-zA-Z0-9]\.[a-zA-Z0-9]+))`
	WinPath      string = `^[a-zA-Z]:\\(?:[^\\/:*?"<>|\r\n]+\\)*[^\\/:*?"<>|\r\n]*$`
	UnixPath     string = `^(/[^/\x00]*)+/?$`
)

// some string regexp. (it is from package "asaskevich/govalidator")
var (
	// rxUser           = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~.-]+$")
	// rxHostname       = regexp.MustCompile("^[^\\s]+\\.[^\\s]+$")
	// rxUserDot        = regexp.MustCompile("(^[.]{1})|([.]{1}$)|([.]{2,})")
	rxEmail          = regexp.MustCompile(Email)
	rxISBN10         = regexp.MustCompile("^(?:[0-9]{9}X|[0-9]{10})$")
	rxISBN13         = regexp.MustCompile("^(?:[0-9]{13})$")
	rxUUID3          = regexp.MustCompile(UUID3)
	rxUUID4          = regexp.MustCompile(UUID4)
	rxUUID5          = regexp.MustCompile(UUID5)
	rxUUID           = regexp.MustCompile(UUID)
	rxAlpha          = regexp.MustCompile("^[a-zA-Z]+$")
	rxAlphaNum       = regexp.MustCompile("^[a-zA-Z0-9]+$")
	rxAlphaDash      = regexp.MustCompile(`^(?:[\w-]+)$`)
	rxNumber         = regexp.MustCompile("^[0-9]+$")
	rxInt            = regexp.MustCompile(Int)
	rxFloat          = regexp.MustCompile(Float)
	rxHexadecimal    = regexp.MustCompile("^[0-9a-fA-F]+$")
	rxHexColor       = regexp.MustCompile("^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$")
	rxRGBColor       = regexp.MustCompile(RGBColor)
	rxASCII          = regexp.MustCompile("^[\x00-\x7F]+$")
	rxPrintableASCII = regexp.MustCompile("^[\x20-\x7E]+$")
	rxMultiByte      = regexp.MustCompile("[^\x00-\x7F]")
	// rxFullWidth      = regexp.MustCompile(FullWidth)
	// rxHalfWidth      = regexp.MustCompile(HalfWidth)
	rxBase64    = regexp.MustCompile(Base64)
	rxDataURI   = regexp.MustCompile(`^data:.+/(.+);base64,(?:.+)`)
	rxLatitude  = regexp.MustCompile(Latitude)
	rxLongitude = regexp.MustCompile(Longitude)
	rxDNSName   = regexp.MustCompile(DNSName)
	// rxSSN            = regexp.MustCompile(`^\d{3}[- ]?\d{2}[- ]?\d{4}$`)
	rxWinPath      = regexp.MustCompile(WinPath)
	rxUnixPath     = regexp.MustCompile(UnixPath)
	rxHasLowerCase = regexp.MustCompile(".*[[:lower:]]")
	rxHasUpperCase = regexp.MustCompile(".*[[:upper:]]")
)

// some validator alias name
var validatorAliases = map[string]string{
	// alias -> real name
	"in":    "enum",
	"num":   "number",
	"range": "between",
	// type
	"int":     "isInt",
	"integer": "isInt",
	"uint":    "isUint",
	"bool":    "isBool",
	"float":   "isFloat",
	"map":     "isMap",
	"ints":    "isInts", // []int
	"str":     "isString",
	"string":  "isString",
	"strings": "isStrings", // []string
	"arr":     "isArray",
	"array":   "isArray",
	"slice":   "isSlice",
	// val
	"regex": "regexp",
	"eq":    "isEqual",
	"equal": "isEqual",
	"intEq": "intEqual",
	"ne":    "notEqual",
	"notEq": "notEqual",
	// int
	"lte":         "max",
	"gte":         "min",
	"lessThan":    "lt",
	"greaterThan": "gt",
	// len
	"len":      "length",
	"lenEq":    "length",
	"lengthEq": "length",
	"minLen":   "minLength",
	"maxLen":   "maxLength",
	"minSize":  "minLength",
	"maxSize":  "maxLength",
	// string rune length
	"strLength":  "stringLength",
	"runeLength": "stringLength",
	// string
	"ip":        "isIP",
	"ipv4":      "isIPv4",
	"ipv6":      "isIPv6",
	"email":     "isEmail",
	"intStr":    "isIntString",
	"intString": "isIntString",
	//
	"hexadecimal":    "isHexadecimal",
	"hasWhitespace":  "hasWhitespace",
	"printableASCII": "isPrintableASCII",
	//
	"ascii":     "isASCII",
	"ASCII":     "isASCII",
	"alpha":     "isAlpha",
	"alphaNum":  "isAlphaNum",
	"alphaDash": "isAlphaDash",
	"base64":    "isBase64",
	"CIDR":      "isCIDR",
	"CIDRv4":    "isCIDRv4",
	"CIDRv6":    "isCIDRv6",
	"dnsName":   "isDNSName",
	"DNSName":   "isDNSName",
	"dataURI":   "isDataURI",
	"empty":     "isEmpty",
	"filePath":  "isFilePath",
	"hexColor":  "isHexColor",
	"isbn10":    "isISBN10",
	"ISBN10":    "isISBN10",
	"isbn13":    "isISBN13",
	"ISBN13":    "isISBN13",
	"json":      "isJSON",
	"JSON":      "isJSON",
	"lat":       "isLatitude",
	"latitude":  "isLatitude",
	"lon":       "isLongitude",
	"longitude": "isLongitude",
	"mac":       "isMAC",
	"multiByte": "isMultiByte",
	"number":    "isNumber",
	"rgbColor":  "isRGBColor",
	"RGBColor":  "isRGBColor",
	"url":       "isURL",
	"URL":       "isURL",
	"uuid":      "isUUID",
	"uuid3":     "isUUID3",
	"uuid4":     "isUUID4",
	"uuid5":     "isUUID5",
	"UUID":      "isUUID",
	"UUID3":     "isUUID3",
	"UUID4":     "isUUID4",
	"UUID5":     "isUUID5",
	"unixPath":  "isUnixPath",
	"winPath":   "isWinPath",
	// date
	"date":    "isDate",
	"gtDate":  "afterDate",
	"ltDate":  "beforeDate",
	"gteDate": "afterOrEqualDate",
	"lteDate": "beforeOrEqualDate",
	// uploaded file
	"img":       "isImage",
	"file":      "isFile",
	"image":     "isImage",
	"mimes":     "inMimeTypes",
	"mimeType":  "inMimeTypes",
	"mimeTypes": "inMimeTypes",
}

// ValidatorName get real validator name.
func ValidatorName(name string) string {
	if rName, ok := validatorAliases[name]; ok {
		return rName
	}

	return name
}

/*************************************************************
 * global validators
 *************************************************************/

// global validators. contains built-in and user custom
var (
	validators map[string]int
	// validator func meta info
	validatorMetas map[string]*funcMeta
	// validator func reflect.Value
	validatorValues = map[string]reflect.Value{
		// int value
		"lt":  reflect.ValueOf(Lt),
		"gt":  reflect.ValueOf(Gt),
		"min": reflect.ValueOf(Min),
		"max": reflect.ValueOf(Max),
		// value check
		"enum":     reflect.ValueOf(Enum),
		"notIn":    reflect.ValueOf(NotIn),
		"between":  reflect.ValueOf(Between),
		"regexp":   reflect.ValueOf(Regexp),
		"isEqual":  reflect.ValueOf(IsEqual),
		"intEqual": reflect.ValueOf(IntEqual),
		"notEqual": reflect.ValueOf(NotEqual),
		// contains
		"contains":    reflect.ValueOf(Contains),
		"notContains": reflect.ValueOf(NotContains),
		// data type check
		"isInt":     reflect.ValueOf(IsInt),
		"isMap":     reflect.ValueOf(IsMap),
		"isUint":    reflect.ValueOf(IsUint),
		"isBool":    reflect.ValueOf(IsBool),
		"isFloat":   reflect.ValueOf(IsFloat),
		"isInts":    reflect.ValueOf(IsInts),
		"isArray":   reflect.ValueOf(IsArray),
		"isSlice":   reflect.ValueOf(IsSlice),
		"isString":  reflect.ValueOf(IsString),
		"isStrings": reflect.ValueOf(IsStrings),
		// length
		"length":    reflect.ValueOf(Length),
		"minLength": reflect.ValueOf(MinLength),
		"maxLength": reflect.ValueOf(MaxLength),
		// string
		"isIntString": reflect.ValueOf(IsIntString),
		// ip
		"isIP":          reflect.ValueOf(IsIP),
		"isIPv4":        reflect.ValueOf(IsIPv4),
		"isIPv6":        reflect.ValueOf(IsIPv6),
		"isEmail":       reflect.ValueOf(IsEmail),
		"hasWhitespace": reflect.ValueOf(HasWhitespace),
		"isASCII":       reflect.ValueOf(IsASCII),
		"isAlpha":       reflect.ValueOf(IsAlpha),
		"isAlphaNum":    reflect.ValueOf(IsAlphaNum),
		"isAlphaDash":   reflect.ValueOf(IsAlphaDash),
		"isBase64":      reflect.ValueOf(IsBase64),
		"isCIDR":        reflect.ValueOf(IsCIDR),
		"isCIDRv4":      reflect.ValueOf(IsCIDRv4),
		"isCIDRv6":      reflect.ValueOf(IsCIDRv6),
		"isDNSName":     reflect.ValueOf(IsDNSName),
		"isDataURI":     reflect.ValueOf(IsDataURI),
		"isEmpty":       reflect.ValueOf(IsEmpty),
		"isFilePath":    reflect.ValueOf(IsFilePath),
		"isHexColor":    reflect.ValueOf(IsHexColor),
		"isHexadecimal": reflect.ValueOf(IsHexadecimal),
		"isISBN10":      reflect.ValueOf(IsISBN10),
		"isISBN13":      reflect.ValueOf(IsISBN13),
		"isJSON":        reflect.ValueOf(IsJSON),
		"isLatitude":    reflect.ValueOf(IsLatitude),
		"isLongitude":   reflect.ValueOf(IsLongitude),
		"isMAC":         reflect.ValueOf(IsMAC),
		"isMultiByte":   reflect.ValueOf(IsMultiByte),
		"isNumber":      reflect.ValueOf(IsNumber),
		//
		"isPrintableASCII": reflect.ValueOf(IsPrintableASCII),
		//
		"isRGBColor": reflect.ValueOf(IsRGBColor),
		"isURL":      reflect.ValueOf(IsURL),
		"isUUID":     reflect.ValueOf(IsUUID),
		"isUUID3":    reflect.ValueOf(IsUUID3),
		"isUUID4":    reflect.ValueOf(IsUUID4),
		"isUUID5":    reflect.ValueOf(IsUUID5),
		"isUnixPath": reflect.ValueOf(IsUnixPath),
		"isWinPath":  reflect.ValueOf(IsWinPath),
		// date check
		"isDate":     reflect.ValueOf(IsDate),
		"afterDate":  reflect.ValueOf(AfterDate),
		"beforeDate": reflect.ValueOf(BeforeDate),
		//
		"afterOrEqualDate":  reflect.ValueOf(AfterOrEqualDate),
		"beforeOrEqualDate": reflect.ValueOf(BeforeOrEqualDate),
	}
)

type funcMeta struct {
	fv   reflect.Value
	name string
	// readonly cache
	numIn  int
	numOut int
	// is internal built in validator
	isInternal bool
	// last arg is like "... interface{}"
	isVariadic bool
}

func (fm *funcMeta) checkArgNum(argNum int, name string) {
	// last arg is like "... interface{}"
	if fm.isVariadic {
		if argNum+1 < fm.numIn {
			panicf("not enough parameters for validator '%s'!", name)
		}
	} else if argNum != fm.numIn {
		panicf(
			"the number of parameters given does not match the required. validator '%s', want %d, given %d",
			name,
			fm.numIn,
			argNum,
		)
	}
}

func newFuncMeta(name string, isInternal bool, fv reflect.Value) *funcMeta {
	fm := &funcMeta{fv: fv, name: name, isInternal: isInternal}
	ft := fv.Type()

	fm.numIn = ft.NumIn()   // arg num of the func
	fm.numOut = ft.NumOut() // return arg num of the func
	fm.isVariadic = ft.IsVariadic()

	return fm
}

func init() {
	validators = make(map[string]int)
	validatorMetas = make(map[string]*funcMeta)

	for n, fv := range validatorValues {
		validators[n] = 1 // built in
		validatorMetas[n] = newFuncMeta(n, true, fv)
	}
}

// AddValidators to the global validators map
func AddValidators(m map[string]interface{}) {
	for name, checkFunc := range m {
		AddValidator(name, checkFunc)
	}
}

// AddValidator to the pkg. checkFunc must return a bool
func AddValidator(name string, checkFunc interface{}) {
	fv := checkValidatorFunc(name, checkFunc)

	validators[name] = 2 // custom
	validatorValues[name] = fv
	validatorMetas[name] = newFuncMeta(name, false, fv)
}

// Validators get all validator names
func Validators() map[string]int {
	return validators
}

/*************************************************************
 * context validators(TODO requiredIf, requiredUnless)
 *************************************************************/

// Required field val check
func (v *Validation) Required(field string, val interface{}) bool {
	// check file
	fd, ok := v.data.(*FormData)
	if ok && fd.HasFile(field) {
		return true
	}

	// check value
	return !IsEmpty(val)
}

// EqField value should EQ the dst field value
func (v *Validation) EqField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	// return val == dstVal
	return IsEqual(val, dstVal)
}

// NeField value should not equal the dst field value
func (v *Validation) NeField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	// return val != dstVal
	return !IsEqual(val, dstVal)
}

// GtField value should GT the dst field value
func (v *Validation) GtField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return valueCompare(val, dstVal, "gt")
}

// GteField value should GTE the dst field value
func (v *Validation) GteField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return valueCompare(val, dstVal, "gte")
}

// LtField value should LT the dst field value
func (v *Validation) LtField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return valueCompare(val, dstVal, "lt")
}

// LteField value should LTE the dst field value(for int, string)
func (v *Validation) LteField(val interface{}, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return valueCompare(val, dstVal, "lte")
}

/*************************************************************
 * context: file validators
 *************************************************************/

var fileValidators = "|isFile|isImage|inMimeTypes|"
var imageMimeTypes = map[string]string{
	"bmp": "image/bmp",
	"gif": "image/gif",
	"ief": "image/ief",
	"jpg": "image/jpeg",
	// "jpe":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"svg":  "image/svg+xml",
	"ico":  "image/x-icon",
	"webp": "image/webp",
}

func isFileValidator(name string) bool {
	return strings.Contains(fileValidators, "|"+name+"|")
}

// IsFile check field is uploaded file
func (v *Validation) IsFile(fd *FormData, field string) (ok bool) {
	if fh := fd.GetFile(field); fh != nil {
		_, err := fh.Open()
		if err == nil {
			return true
		}
	}
	return false
}

// IsImage check field is uploaded image file.
// Usage:
// 	v.AddRule("avatar", "image")
// 	v.AddRule("avatar", "image", "jpg", "png", "gif") // set ext limit
func (v *Validation) IsImage(fd *FormData, field string, exts ...string) (ok bool) {
	mime := fd.FileMimeType(field)
	if mime == "" {
		return
	}

	var fileExt string
	for ext, imgMime := range imageMimeTypes {
		if imgMime == mime {
			fileExt = ext
			ok = true
			break
		}
	}

	// don't limit mime type
	if len(exts) == 0 {
		return ok // only check is an image
	}

	return Enum(fileExt, exts)
}

// InMimeTypes check field is uploaded file and mime type is in the mimeTypes.
// Usage:
// 	v.AddRule("video", "mimeTypes", "video/avi", "video/mpeg", "video/quicktime")
func (v *Validation) InMimeTypes(fd *FormData, field, mimeType string, moreTypes ...string) bool {
	mime := fd.FileMimeType(field)
	if mime == "" {
		return false
	}

	mimeTypes := append(moreTypes, mimeType)
	return Enum(mime, mimeTypes)
}

/*************************************************************
 * global: basic validators
 *************************************************************/

// IsEmpty of the value
func IsEmpty(val interface{}) bool {
	if val == nil {
		return true
	}

	if s, ok := val.(string); ok {
		return s == ""
	}
	return ValueIsEmpty(reflect.ValueOf(val))
}

// Contains check that the specified string, list(array, slice) or map contains the
// specified substring or element.
//
// Notice: list check value exist. map check key exist.
func Contains(s, sub interface{}) bool {
	ok, found := includeElement(s, sub)

	// ok == false: 's' could not be applied builtin len()
	// found == false: 's' does not contain 'sub'
	return ok && found
}

// NotContains check that the specified string, list(array, slice) or map does NOT contain the
// specified substring or element.
//
// Notice: list check value exist. map check key exist.
func NotContains(s, sub interface{}) bool {
	ok, found := includeElement(s, sub)

	// ok == false: could not be applied builtin len()
	// found == true: 's' contain 'sub'
	return ok && !found
}

/*************************************************************
 * global: type validators
 *************************************************************/

// IsUint check, allow: intX, uintX, string
func IsUint(val interface{}) bool {
	switch typVal := val.(type) {
	case int:
		return typVal >= 0
	case int8:
		return typVal >= 0
	case int16:
		return typVal >= 0
	case int32:
		return typVal >= 0
	case int64:
		return typVal >= 0
	case uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseUint(typVal, 10, 32)
		return err == nil
	}
	return false
}

// IsBool check. allow: bool, string.
func IsBool(val interface{}) bool {
	if _, ok := val.(bool); ok {
		return true
	}

	if typVal, ok := val.(string); ok {
		_, err := filter.Bool(typVal)
		return err == nil
	}
	return false
}

// IsFloat check. allow: floatX, string
func IsFloat(val interface{}) bool {
	if val == nil {
		return false
	}

	switch rv := val.(type) {
	case float32, float64:
		return true
	case string:
		return rv != "" && rxFloat.MatchString(rv)
	}
	return false
}

// IsArray check
func IsArray(val interface{}) (ok bool) {
	if val == nil {
		return false
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	return rv.Kind() == reflect.Array
}

// IsSlice check
func IsSlice(val interface{}) (ok bool) {
	if val == nil {
		return false
	}

	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	return rv.Kind() == reflect.Slice
}

// IsInts is int slice check
func IsInts(val interface{}) bool {
	if val == nil {
		return false
	}

	switch val.(type) {
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64:
		return true
	}
	return false
}

// IsStrings is string slice check
func IsStrings(val interface{}) (ok bool) {
	if val == nil {
		return false
	}

	_, ok = val.([]string)
	return
}

// IsMap check
func IsMap(val interface{}) (ok bool) {
	if val == nil {
		return false
	}

	var rv reflect.Value
	if rv, ok = val.(reflect.Value); !ok {
		rv = reflect.ValueOf(val)
	}

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	return rv.Kind() == reflect.Map
}

// IsInt check, and support length check
func IsInt(val interface{}, minAndMax ...int64) (ok bool) {
	if val == nil {
		return false
	}

	intVal, err := valueToInt64(val, true)
	if err != nil {
		return false
	}

	argLn := len(minAndMax)
	if argLn == 0 { // only check type
		return true
	}

	// value check
	minVal := minAndMax[0]
	if argLn == 1 { // only min length check.
		return intVal >= minVal
	}

	maxVal := minAndMax[1]

	// min and max length check
	return intVal >= minVal && intVal <= maxVal
}

// IsString check, and support length check.
// Usage:
// 	ok := IsString(val)
// 	ok := IsString(val, 5) // with min len check
// 	ok := IsString(val, 5, 12) // with min and max len check
func IsString(val interface{}, minAndMaxLen ...int) (ok bool) {
	if val == nil {
		return false
	}

	argLn := len(minAndMaxLen)
	str, isStr := val.(string)

	// only check type
	if argLn == 0 {
		return isStr
	}

	if !isStr {
		return false
	}

	// length check
	strLen := len(str)
	minLen := minAndMaxLen[0]

	// only min length check.
	if argLn == 1 {
		return strLen >= minLen
	}

	// min and max length check
	maxLen := minAndMaxLen[1]
	return strLen >= minLen && strLen <= maxLen
}

/*************************************************************
 * global: string validators
 *************************************************************/

// HasWhitespace check. eg "10"
func HasWhitespace(s string) bool {
	return s != "" && strings.ContainsRune(s, ' ')
}

// IsIntString check. eg "10"
func IsIntString(s string) bool {
	return s != "" && rxInt.MatchString(s)
}

// IsASCII string.
func IsASCII(s string) bool {
	return s != "" && rxASCII.MatchString(s)
}

// IsPrintableASCII string.
func IsPrintableASCII(s string) bool {
	return s != "" && rxPrintableASCII.MatchString(s)
}

// IsBase64 string.
func IsBase64(s string) bool {
	return s != "" && rxBase64.MatchString(s)
}

// IsLatitude string.
func IsLatitude(s string) bool {
	return s != "" && rxLatitude.MatchString(s)
}

// IsLongitude string.
func IsLongitude(s string) bool {
	return s != "" && rxLongitude.MatchString(s)
}

// IsDNSName string.
func IsDNSName(s string) bool {
	return s != "" && rxDNSName.MatchString(s)
}

// IsURL string.
func IsURL(s string) bool {
	if s == "" {
		return false
	}

	_, err := url.Parse(s)
	return err == nil
}

// IsDataURI string.
// data:[<mime type>] ( [;charset=<charset>] ) [;base64],码内容
// eg. "data:image/gif;base64,R0lGODlhA..."
func IsDataURI(s string) bool {
	return s != "" && rxDataURI.MatchString(s)
}

// IsMultiByte string.
func IsMultiByte(s string) bool {
	return s != "" && rxMultiByte.MatchString(s)
}

// IsISBN10 string.
func IsISBN10(s string) bool {
	return s != "" && rxISBN10.MatchString(s)
}

// IsISBN13 string.
func IsISBN13(s string) bool {
	return s != "" && rxISBN13.MatchString(s)
}

// IsHexadecimal string.
func IsHexadecimal(s string) bool {
	return s != "" && rxHexadecimal.MatchString(s)
}

// IsHexColor string.
func IsHexColor(s string) bool {
	return s != "" && rxHexColor.MatchString(s)
}

// IsRGBColor string.
func IsRGBColor(s string) bool {
	return s != "" && rxRGBColor.MatchString(s)
}

// IsAlpha string.
func IsAlpha(s string) bool {
	return s != "" && rxAlpha.MatchString(s)
}

// IsAlphaNum string.
func IsAlphaNum(s string) bool {
	return s != "" && rxAlphaNum.MatchString(s)
}

// IsAlphaDash string.
func IsAlphaDash(s string) bool {
	return s != "" && rxAlphaDash.MatchString(s)
}

// IsNumber string. should >= 0
func IsNumber(s string) bool {
	return s != "" && rxNumber.MatchString(s)
}

// IsFilePath string
func IsFilePath(str string) bool {
	if str == "" {
		return false
	}

	if _, err := os.Stat(str); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// IsWinPath string
func IsWinPath(s string) bool {
	return s != "" && rxWinPath.MatchString(s)
}

// IsUnixPath string
func IsUnixPath(s string) bool {
	return s != "" && rxUnixPath.MatchString(s)
}

// IsEmail check
func IsEmail(s string) bool {
	return s != "" && rxEmail.MatchString(s)
}

// IsUUID string
func IsUUID(s string) bool {
	return s != "" && rxUUID.MatchString(s)
}

// IsUUID3 string
func IsUUID3(s string) bool {
	return s != "" && rxUUID3.MatchString(s)
}

// IsUUID4 string
func IsUUID4(s string) bool {
	return s != "" && rxUUID4.MatchString(s)
}

// IsUUID5 string
func IsUUID5(s string) bool {
	return s != "" && rxUUID5.MatchString(s)
}

// IsIP is the validation function for validating if the field's value is a valid v4 or v6 IP address.
func IsIP(s string) bool {
	// ip := net.ParseIP(s)
	return s != "" && net.ParseIP(s) != nil
}

// IsIPv4 is the validation function for validating if a value is a valid v4 IP address.
func IsIPv4(s string) bool {
	if s == "" {
		return false
	}

	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

// IsIPv6 is the validation function for validating if the field's value is a valid v6 IP address.
func IsIPv6(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() == nil
}

// IsMAC is the validation function for validating if the field's value is a valid MAC address.
func IsMAC(s string) bool {
	if s == "" {
		return false
	}
	_, err := net.ParseMAC(s)
	return err == nil
}

// IsCIDRv4 is the validation function for validating if the field's value is a valid v4 CIDR address.
func IsCIDRv4(s string) bool {
	if s == "" {
		return false
	}
	ip, _, err := net.ParseCIDR(s)
	return err == nil && ip.To4() != nil
}

// IsCIDRv6 is the validation function for validating if the field's value is a valid v6 CIDR address.
func IsCIDRv6(s string) bool {
	if s == "" {
		return false
	}

	ip, _, err := net.ParseCIDR(s)
	return err == nil && ip.To4() == nil
}

// IsCIDR is the validation function for validating if the field's value is a valid v4 or v6 CIDR address.
func IsCIDR(s string) bool {
	if s == "" {
		return false
	}

	_, _, err := net.ParseCIDR(s)
	return err == nil
}

// IsJSON check if the string is valid JSON (note: uses json.Unmarshal).
func IsJSON(s string) bool {
	if s == "" {
		return false
	}

	var js json.RawMessage
	return Unmarshal([]byte(s), &js) == nil
}

// HasLowerCase check string has lower case
func HasLowerCase(s string) bool {
	if s == "" {
		return false
	}

	return rxHasLowerCase.MatchString(s)
}

// HasUpperCase check string has upper case
func HasUpperCase(s string) bool {
	if s == "" {
		return false
	}

	return rxHasUpperCase.MatchString(s)
}

// Regexp match value string
func Regexp(str string, pattern string) bool {
	ok, _ := regexp.MatchString(pattern, str)
	return ok
}

/*************************************************************
 * global: compare validators
 *************************************************************/

// IsEqual check two value is equals.
// Support: bool, int(X), uint(X), string, float(X) AND slice, array, map
func IsEqual(val, wantVal interface{}) bool {
	// check is nil
	if val == nil || wantVal == nil {
		return val == wantVal
	}

	sv := reflect.ValueOf(val)
	wv := reflect.ValueOf(wantVal)

	// don't compare func, struct
	if sv.Kind() == reflect.Func || sv.Kind() == reflect.Struct {
		return false
	}
	if wv.Kind() == reflect.Func || wv.Kind() == reflect.Struct {
		return false
	}

	// compare basic type: bool, int(X), uint(X), string, float(X)
	equal, err := eq(sv, wv)

	// is not an basic type, eg: slice, array, map ...
	if err != nil {
		expBt, ok := val.([]byte)
		if !ok {
			return reflect.DeepEqual(val, wantVal)
		}

		actBt, ok := wantVal.([]byte)
		if !ok {
			return false
		}
		if expBt == nil || actBt == nil {
			return expBt == nil && actBt == nil
		}

		return bytes.Equal(expBt, actBt)
	}

	return equal
}

// NotEqual check
func NotEqual(val, wantVal interface{}) bool {
	return !IsEqual(val, wantVal)
}

// IntEqual check
func IntEqual(val interface{}, wantVal int64) bool {
	// intVal, isInt := IntVal(val)
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal == wantVal
}

// Gt check value greater dst value. only check for: int(X), uint(X), float(X)
func Gt(val interface{}, dstVal int64) bool {
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal > dstVal
}

// Min check value greater or equal dst value, alias `Gte`.
// only check for: int(X), uint(X), float(X).
func Min(val interface{}, min int64) bool {
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal >= min
}

// Lt less than dst value. only check for: int(X), uint(X), float(X).
func Lt(val interface{}, dstVal int64) bool {
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal < dstVal
}

// Max less than or equal dst value, alias `Lte`. check for: int(X), uint(X), float(X).
func Max(val interface{}, max int64) bool {
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal <= max
}

// Between int value in the given range.
func Between(val interface{}, min, max int64) bool {
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	return intVal >= min && intVal <= max
}

/*************************************************************
 * global: array, slice, map validators
 *************************************************************/

// Enum value(int(X),string) should be in the given enum(strings, ints, uints).
func Enum(val, enum interface{}) bool {
	if val == nil || enum == nil {
		return false
	}

	// if is string value
	if strVal, ok := val.(string); ok {
		if ss, ok := enum.([]string); ok {
			for _, strItem := range ss {
				if strVal == strItem { // exists
					return true
				}
			}
		}

		return false
	}

	// as int value
	intVal, err := filter.Int64(val)
	if err != nil {
		return false
	}

	if int64s, ok := toInt64Slice(enum); ok {
		for _, i64 := range int64s {
			if intVal == i64 { // exists
				return true
			}
		}
	}
	return false
}

// NotIn value should be not in the given enum(strings, ints, uints).
func NotIn(val, enum interface{}) bool {
	return false == Enum(val, enum)
}

/*************************************************************
 * global: length validators
 *************************************************************/

// Length equal check for string, array, slice, map
func Length(val interface{}, wantLen int) bool {
	ln := CalcLength(val)
	if ln == -1 {
		return false
	}

	return ln == wantLen
}

// MinLength check for string, array, slice, map
func MinLength(val interface{}, minLen int) bool {
	ln := CalcLength(val)
	if ln == -1 {
		return false
	}

	return ln >= minLen
}

// MaxLength check for string, array, slice, map
func MaxLength(val interface{}, maxLen int) bool {
	ln := CalcLength(val)
	if ln == -1 {
		return false
	}

	return ln <= maxLen
}

// ByteLength check string's length
func ByteLength(str string, minLen int, maxLen ...int) bool {
	strLen := len(str)

	// only min length check.
	if len(maxLen) == 0 {
		return strLen >= minLen
	}

	// min and max length check
	return strLen >= minLen && strLen <= maxLen[0]
}

// RuneLength check string's length (including multi byte strings)
func RuneLength(str string, minLen int, maxLen ...int) bool {
	// strLen := len([]rune(str))
	strLen := utf8.RuneCountInString(str)

	// only min length check.
	if len(maxLen) == 0 {
		return strLen >= minLen
	}

	// min and max length check
	return strLen >= minLen && strLen <= maxLen[0]
}

// StringLength check string's length (including multi byte strings)
func StringLength(str string, minLen int, maxLen ...int) bool {
	return RuneLength(str, minLen, maxLen...)
}

/*************************************************************
 * global: date/time validators
 *************************************************************/

// IsDate check value is an date string.
func IsDate(srcDate string) bool {
	_, err := filter.StrToTime(srcDate)
	return err == nil
}

// DateFormat check
func DateFormat(s string, layout string) bool {
	_, err := time.Parse(layout, s)
	return err == nil
}

// DateEquals check.
// Usage:
// 	DateEquals(val, "2017-05-12")
// func DateEquals(srcDate, dstDate string) bool {
// 	return false
// }

// BeforeDate check
func BeforeDate(srcDate, dstDate string) bool {
	st, err := filter.StrToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := filter.StrToTime(dstDate)
	if err != nil {
		return false
	}

	return st.Before(dt)
}

// BeforeOrEqualDate check
func BeforeOrEqualDate(srcDate, dstDate string) bool {
	st, err := filter.StrToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := filter.StrToTime(dstDate)
	if err != nil {
		return false
	}

	return st.Before(dt) || st.Equal(dt)
}

// AfterOrEqualDate check
func AfterOrEqualDate(srcDate, dstDate string) bool {
	st, err := filter.StrToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := filter.StrToTime(dstDate)
	if err != nil {
		return false
	}

	return st.After(dt) || st.Equal(dt)
}

// AfterDate check
func AfterDate(srcDate, dstDate string) bool {
	st, err := filter.StrToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := filter.StrToTime(dstDate)
	if err != nil {
		return false
	}

	return st.After(dt)
}
