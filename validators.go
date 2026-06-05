package validate

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/gookit/validate/v2/internal/reflectx"
)

// Basic regular expressions for validating strings.
// (there are from package "asaskevich/govalidator")
const (
	Email        = `^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`
	UUID3        = "^[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$"
	UUID4        = "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	UUID5        = "^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	UUID         = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
	Int          = "^(?:[-+]?(?:0|[1-9][0-9]*))$"
	Float        = "^(?:[-+]?(?:[0-9]+))?(?:\\.[0-9]*)?(?:[eE][\\+\\-]?(?:[0-9]+))?$"
	RGBColor     = "^rgb\\(\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*\\)$"
	FullWidth    = "[^\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	HalfWidth    = "[\u0020-\u007E\uFF61-\uFF9F\uFFA0-\uFFDC\uFFE8-\uFFEE0-9a-zA-Z]"
	Base64       = "^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$"
	Latitude     = "^[-+]?([1-8]?\\d(\\.\\d+)?|90(\\.0+)?)$"
	Longitude    = "^[-+]?(180(\\.0+)?|((1[0-7]\\d)|([1-9]?\\d))(\\.\\d+)?)$"
	DNSName      = `^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`
	// FullURL requires scheme + a structured host (domain.tld or IPv4), with an
	// optional port and path/query/fragment. The host structure is what rejects
	// the over-permissive cases of the old `[\w./#=?&-_%]+` blob (#138): a missing
	// TLD ("https://www"), invalid host chars ("https://not%23"), or an underscore
	// in the host ("https://www.googl_?e.com/...").
	FullURL      = `^(?:ftp|tcp|udp|wss?|https?)://(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}|(?:\d{1,3}\.){3}\d{1,3})(?::\d{1,5})?(?:[/?#]\S*)?$`
	URLSchema    = `((ftp|tcp|udp|wss?|https?):\/\/)`
	URLUsername  = `(\S+(:\S*)?@)`
	URLPath      = `((\/|\?|#)[^\s]*)`
	URLPort      = `(:(\d{1,5}))`
	URLIP        = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain = `((www\.)|([a-zA-Z0-9]+([-_\.]?[a-zA-Z0-9])*[a-zA-Z0-9]\.[a-zA-Z0-9]+))`
	WinPath      = `^[a-zA-Z]:\\(?:[^\\/:*?"<>|\r\n]+\\)*[^\\/:*?"<>|\r\n]*$`
	UnixPath     = `^(/[^/\x00]*)+/?$`
)

// some string regexp. (it is from package "asaskevich/govalidator")
var (
	// rxUser           = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~.-]+$")
	// rxHostname       = regexp.MustCompile("^[^\\s]+\\.[^\\s]+$")
	// rxUserDot        = regexp.MustCompile("(^[.]{1})|([.]{1}$)|([.]{2,})")
	rxEmail     = regexp.MustCompile(Email)
	rxISBN10    = regexp.MustCompile(`^(?:\d{9}X|\d{10})$`)
	rxISBN13    = regexp.MustCompile(`^\d{13}$`)
	rxUUID3     = regexp.MustCompile(UUID3)
	rxUUID4     = regexp.MustCompile(UUID4)
	rxUUID5     = regexp.MustCompile(UUID5)
	rxUUID      = regexp.MustCompile(UUID)
	rxAlpha     = regexp.MustCompile("^[a-zA-Z]+$")
	rxAlphaNum  = regexp.MustCompile("^[a-zA-Z0-9]+$")
	rxAlphaDash = regexp.MustCompile(`^(?:[\w-]+)$`)
	rxNumber    = regexp.MustCompile("^[0-9]+$")
	rxInt       = regexp.MustCompile(Int)
	rxFloat     = regexp.MustCompile(Float)
	rxCnMobile  = regexp.MustCompile(`^1\d{10}$`)
	rxHexColor  = regexp.MustCompile(`^#?([\da-fA-F]{3}|[\da-fA-F]{6})$`)
	rxRGBColor  = regexp.MustCompile(RGBColor)
	rxASCII     = regexp.MustCompile("^[\x00-\x7F]+$")
	// --
	rxHexadecimal    = regexp.MustCompile(`^[\da-fA-F]+$`)
	rxPrintableASCII = regexp.MustCompile("^[\x20-\x7E]+$")
	rxMultiByte      = regexp.MustCompile("[^\x00-\x7F]")
	// rxFullWidth = regexp.MustCompile(FullWidth)
	// rxHalfWidth = regexp.MustCompile(HalfWidth)
	rxBase64    = regexp.MustCompile(Base64)
	rxDataURI   = regexp.MustCompile(`^data:.+/(.+);base64,(?:.+)`)
	rxLatitude  = regexp.MustCompile(Latitude)
	rxLongitude = regexp.MustCompile(Longitude)
	rxDNSName   = regexp.MustCompile(DNSName)
	rxFullURL   = regexp.MustCompile(FullURL)
	rxURLSchema = regexp.MustCompile(URLSchema)
	// rxSSN            = regexp.MustCompile(`^\d{3}[- ]?\d{2}[- ]?\d{4}$`)
	rxWinPath  = regexp.MustCompile(WinPath)
	rxUnixPath = regexp.MustCompile(UnixPath)
	// --
	rxHasLowerCase = regexp.MustCompile(".*[[:lower:]]")
	rxHasUpperCase = regexp.MustCompile(".*[[:upper:]]")
)

/*************************************************************
 * global validators
 *************************************************************/

type funcMeta struct {
	fv reflect.Value
	// validator name
	name string
	// readonly cache
	numIn  int
	numOut int
	// is an internal built-in validator
	builtin bool
	// the last arg is variadic param. like "... any"
	isVariadic bool
}

func (fm *funcMeta) checkArgNum(argNum int, name string) {
	// last arg is like "... any"
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

func newFuncMeta(name string, builtin bool, fv reflect.Value) *funcMeta {
	fm := &funcMeta{fv: fv, name: name, builtin: builtin}
	ft := fv.Type()

	fm.numIn = ft.NumIn()   // arg num of the func
	fm.numOut = ft.NumOut() // return arg num of the func
	fm.isVariadic = ft.IsVariadic()

	return fm
}

// ValidatorName get real validator name.
func ValidatorName(name string) string {
	if rName, ok := validatorAliases[name]; ok {
		return rName
	}
	return name
}

// AddValidators to the global validators map
func AddValidators(m map[string]any) {
	for name, checkFunc := range m {
		AddValidator(name, checkFunc)
	}
}

// AddValidator to the pkg. checkFunc must return a bool
//
// Usage:
//
//	v.AddValidator("myFunc", func(val any) bool {
//		// do validate val ...
//		return true
//	})
func AddValidator(name string, checkFunc any) {
	fv := checkValidatorFunc(name, checkFunc)

	validators[name] = validatorTypeCustom
	// validatorValues[name] = fv
	validatorMetas[name] = newFuncMeta(name, false, fv)
}

// Validators get all validator names
func Validators() map[string]int8 {
	return validators
}

/*************************************************************
 * region context: field value check compare
 *  - requiredXXX
 *************************************************************/

// Required field val check
func (v *Validation) Required(field string, val any) bool {
	if v.isInOptional(field) {
		return true
	}

	if v.data != nil && v.data.Type() == sourceForm {
		// check is upload file
		if v.data.(*FormData).HasFile(field) {
			return true
		}
	}

	if v.isIgnoreableZeroNumeric(field) {
		return true
	}

	// check value
	return !IsEmpty(val)
}

// RuleOneOf 规则级"逻辑或"(#292): val 满足列出的任一子校验器即通过, 全部不满足才失败。
//
//   - rules 为子校验器名列表 (来自 rule.go 的 parseArgString, 即 args[0] 的 []string)。
//   - 子项名支持别名 (ip->isIP, cidr->isCIDR), 经 ValidatorName 解析后取 funcMeta。
//   - phase1 仅支持无参子校验器, 故在原 val 上以 addNum=1 直接调用, 不传 field/args。
//   - 未知子校验器名 → 直接 panic, 与现有"未知 validator"行为一致, 提前暴露拼写错误。
func (v *Validation) RuleOneOf(val any, rules any) bool {
	names, ok := rules.([]string)
	if !ok || len(names) == 0 {
		panicf("the validator 'rule_one_of' requires a non-empty list of validator names")
	}

	for _, name := range names {
		realName := ValidatorName(strings.TrimSpace(name))
		fm := v.validatorMeta(realName)
		if fm == nil {
			// fail-fast: 子校验器不存在(含拼写错误), 立即 panic。
			panicf("the validator '%s' for 'rule_one_of' does not exist", name)
		}

		// 在同一个值上调用子校验器, 任一返回 true 即整体通过。
		if callValidator(v, fm, "", val, nil, 1, nil) {
			return true
		}
	}
	return false
}

// RequiredIf field under validation must be present and not empty,
// if the anotherField field is equal to any value.
//
// Usage:
//
//	v.AddRule("password", "requiredIf", "username", "tom")
func (v *Validation) RequiredIf(sourceField string, val any, kvs ...string) bool {
	if len(kvs) < 2 {
		return false
	}

	dstField, args := kvs[0], kvs[1:]
	if dstVal, has := v.Get(dstField); has {
		// Unwrap pointers in the dst-field value so a *T field is
		// compared against the literal rule argument by its underlying
		// kind, not by the reflect.Pointer kind (which has no
		// string-to-pointer conversion and would silently skip the
		// rule). A nil pointer is treated as "field absent" so the
		// optional value is not required.
		rftDv := reflect.ValueOf(dstVal)
		for rftDv.Kind() == reflect.Pointer {
			if rftDv.IsNil() {
				return true
			}
			rftDv = rftDv.Elem()
			dstVal = rftDv.Interface()
		}

		// up: only one check value, direct compare value
		if len(args) == 1 {
			wantVal, err := reflectx.ConvTypeByBaseKind(args[0], rftDv.Kind())
			if err == nil && dstVal == wantVal {
				return requiredIfValIsPresent(val) || v.isIgnoreableZeroNumeric(sourceField)
			}
		} else if Enum(dstVal, args) {
			return requiredIfValIsPresent(val) || v.isIgnoreableZeroNumeric(sourceField)
		}
	}

	// default as True, skip check
	return true
}

// requiredIfValIsPresent reports whether the source-field value
// satisfies a required-style check. A nil pointer counts as absent and
// a pointer to a zero value (e.g. *string("")) is treated the same as
// the zero value itself, so requiredIf does not silently pass for
// pointer-typed empty values.
func requiredIfValIsPresent(val any) bool {
	if val == nil {
		return false
	}
	rv := reflect.ValueOf(val)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	return !ValueIsEmpty(rv)
}

// RequiredUnless field under validation must be present and not empty
// unless the dstField field is equal to any value.
//
//   - kvs format: [dstField, dstVal1, dstVal2 ...]
func (v *Validation) RequiredUnless(sourceField string, val any, kvs ...string) bool {
	if len(kvs) < 2 {
		return false
	}

	dstField, values := kvs[0], kvs[1:]
	if dstVal, has, _ := v.tryGet(dstField); has {
		if !Enum(dstVal, values) {
			return !IsEmpty(val) || v.isIgnoreableZeroNumeric(sourceField)
		}
	}

	// fields in values
	return true
}

// RequiredWith field under validation must be present and not empty only
// if any of the other specified fields are present.
//
//   - fields format: [field1, field2 ...]
func (v *Validation) RequiredWith(sourceField string, val any, fields ...string) bool {
	if len(fields) == 0 {
		return false
	}

	for _, field := range fields {
		if _, has, zero := v.tryGet(field); has && !zero {
			return !IsEmpty(val) || v.isIgnoreableZeroNumeric(sourceField)
		}
	}

	// all fields not exist
	return true
}

// RequiredWithAll field under validation must be present and not empty only if all the other specified fields are present.
func (v *Validation) RequiredWithAll(sourceField string, val any, fields ...string) bool {
	if len(fields) == 0 {
		return false
	}

	for _, field := range fields {
		if _, has, zero := v.tryGet(field); !has || zero {
			// if any field does not exist, not continue.
			return true
		}
	}

	// all fields exist
	return !IsEmpty(val) || v.isIgnoreableZeroNumeric(sourceField)
}

// RequiredWithout field under validation must be present and not empty only when any of the other specified fields are not present.
func (v *Validation) RequiredWithout(sourceField string, val any, fields ...string) bool {
	if len(fields) == 0 {
		return false
	}

	for _, field := range fields {
		if _, has, zero := v.tryGet(field); !has || zero {
			return !IsEmpty(val) || v.isIgnoreableZeroNumeric(sourceField)
		}
	}

	// all fields exist
	return true
}

// RequiredWithoutAll field under validation must be present and not empty only when any of the other specified fields are not present.
func (v *Validation) RequiredWithoutAll(sourceField string, val any, fields ...string) bool {
	if len(fields) == 0 {
		return false
	}

	for _, name := range fields {
		// if any field exists, not continue.
		if _, has, zero := v.tryGet(name); has && !zero {
			return true
		}
	}

	// all fields not exists, required
	return !IsEmpty(val) || v.isIgnoreableZeroNumeric(sourceField)
}

// EqField value should EQ the dst field value
func (v *Validation) EqField(val any, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return IsEqual(val, dstVal)
}

// NeField value should not equal the dst field value
func (v *Validation) NeField(val any, dstField string) bool {
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return !IsEqual(val, dstVal)
}

// GtField value should GT the dst field value
func (v *Validation) GtField(val any, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return reflectx.ValueCompare(val, dstVal, ">")
}

// GteField value should GTE the dst field value
func (v *Validation) GteField(val any, dstField string) bool {
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return reflectx.ValueCompare(val, dstVal, ">=")
}

// LtField value should LT the dst field value
func (v *Validation) LtField(val any, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return reflectx.ValueCompare(val, dstVal, "<")
}

// LteField value should LTE the dst field value(for int, string)
func (v *Validation) LteField(val any, dstField string) bool {
	// get dst field value.
	dstVal, has := v.Get(dstField)
	if !has {
		return false
	}

	return reflectx.ValueCompare(val, dstVal, "<=")
}

/*
 ******************************************************************
 * region context: file validators
 ******************************************************************
 */

const fileValidators = "|isFile|isImage|inMimeTypes|"

var (
	imageMimeTypes = map[string]string{
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
)

func isFileValidator(name string) bool {
	return strings.Contains(fileValidators, "|"+name+"|")
}

// IsFormFile check field is uploaded file. validator: isFile
func (v *Validation) IsFormFile(fd *FormData, field string) (ok bool) {
	field, _, _ = strings.Cut(field, ".*")
	if files := fd.GetFiles(field); len(files) > 0 {
		for i := range files {
			_, err := files[i].Open()
			if err != nil {
				return false
			}
		}
		return true
	}
	return false
}

// IsFormImage check field is uploaded image file. validator: isImage
//
// Usage:
//
//	v.AddRule("avatar", "image")
//	v.AddRule("avatar", "image", "jpg", "png", "gif") // set ext limit
//	v.AddRule("images.*", "image")
//	v.AddRule("images.*", "image", "jpg", "png", "gif") // set ext limit
func (v *Validation) IsFormImage(fd *FormData, field string, exts ...string) (ok bool) {
	field, _, expectArray := strings.Cut(field, ".*")
	if expectArray {
		for _, mime := range fd.FilesMimeType(field) {
			if !v.isImageMimeTypes(mime, exts...) {
				return false
			}
		}
		return true
	}

	return v.isImageMimeTypes(fd.FileMimeType(field), exts...)
}

func (v *Validation) isImageMimeTypes(mime string, exts ...string) (ok bool) {
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

// InMimeTypes check field is uploaded file and mimetype is in the mimeTypes. validator: inMimeTypes
//
// Usage:
//
//	v.AddRule("video", "mimeTypes", "video/avi", "video/mpeg", "video/quicktime")
//	v.AddRule("videos.*", "mimeTypes", "video/avi", "video/mpeg", "video/quicktime")
func (v *Validation) InMimeTypes(fd *FormData, field, mimeType string, moreTypes ...string) bool {
	field, _, expectArray := strings.Cut(field, ".*")
	mimeTypes := append(moreTypes, mimeType) //nolint:gocritic
	if expectArray {
		for _, mime := range fd.FilesMimeType(field) {
			if !v.inMimeTypes(mime, mimeTypes) {
				return false
			}
		}
		return true
	}

	return v.inMimeTypes(fd.FileMimeType(field), mimeTypes)
}

func (v *Validation) inMimeTypes(mime string, mimeTypes []string) bool {
	if mime == "" {
		return false
	}
	return Enum(mime, mimeTypes)
}

func (v *Validation) isIgnoreableZeroNumeric(field string) bool {
	if v.data != nil && v.data.Type() == sourceMap {
		if val, ok := v.data.Get(field); ok {
			k := reflect.ValueOf(val).Kind()
			return k >= reflect.Int && k <= reflect.Float64
		}
	}
	return false
}

/*************************************************************
 * region global: basic validators
 *************************************************************/

// IsEmpty of the value
func IsEmpty(val any) bool {
	if val == nil {
		return true
	}
	if s, ok := val.(string); ok {
		return s == ""
	}

	var rv reflect.Value

	// type check val is reflect.Value
	if v2, ok := val.(reflect.Value); ok {
		rv = v2
	} else {
		rv = reflect.ValueOf(val)
	}
	return ValueIsEmpty(rv)
}

// Contains check that the specified string, list(array, slice) or map contains the
// specified substring or element.
//
// Notice: list check value exist. map check key exist.
func Contains(s, sub any) bool {
	ok, found := includeElement(s, sub)

	// ok == false: 's' could not be applied builtin len()
	// found == false: 's' does not contain 'sub'
	return ok && found
}

// NotContains check that the specified string, list(array, slice) or map does NOT contain the
// specified substring or element.
//
// Notice: list check value exist. map check key exist.
func NotContains(s, sub any) bool {
	ok, found := includeElement(s, sub)

	// ok == false: could not be applied builtin len()
	// found == true: 's' contain 'sub'
	return ok && !found
}
