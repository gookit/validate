package validate

import (
	"reflect"
	"strings"
)

var (
	// global validators. contains built-in and user custom
	validators map[string]int
	// all validators func meta information
	validatorMetas map[string]*funcMeta
)

// init: register all built-in validators
func init() {
	validators = make(map[string]int)
	validatorMetas = make(map[string]*funcMeta)

	for n, fv := range validatorValues {
		validators[n] = 1 // built in
		validatorMetas[n] = newFuncMeta(n, true, fv)
	}
}

// validator func reflect.Value
var validatorValues = map[string]reflect.Value{
	// int value
	"lt":  reflect.ValueOf(Lt),
	"gt":  reflect.ValueOf(Gt),
	"min": reflect.ValueOf(Min),
	"max": reflect.ValueOf(Max),
	// value check
	"enum":       reflect.ValueOf(Enum),
	"notIn":      reflect.ValueOf(NotIn),
	"inIntegers": reflect.ValueOf(InIntegers),
	"inStrings":  reflect.ValueOf(InStrings),
	"between":    reflect.ValueOf(Between),
	"regexp":     reflect.ValueOf(Regexp),
	"isEqual":    reflect.ValueOf(IsEqual),
	"intEqual":   reflect.ValueOf(IntEqual),
	"notEqual":   reflect.ValueOf(NotEqual),
	// contains
	"contains":    reflect.ValueOf(Contains),
	"notContains": reflect.ValueOf(NotContains),
	// string contains
	"stringContains": reflect.ValueOf(StringContains),
	"startsWith":     reflect.ValueOf(StartsWith),
	"endsWith":       reflect.ValueOf(EndsWith),
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
	"length":       reflect.ValueOf(Length),
	"minLength":    reflect.ValueOf(MinLength),
	"maxLength":    reflect.ValueOf(MaxLength),
	"stringLength": reflect.ValueOf(StringLength),
	// string
	"isIntString": reflect.ValueOf(IsIntString),
	// ip
	"isIP":        reflect.ValueOf(IsIP),
	"isIPv4":      reflect.ValueOf(IsIPv4),
	"isIPv6":      reflect.ValueOf(IsIPv6),
	"isEmail":     reflect.ValueOf(IsEmail),
	"isASCII":     reflect.ValueOf(IsASCII),
	"isAlpha":     reflect.ValueOf(IsAlpha),
	"isAlphaNum":  reflect.ValueOf(IsAlphaNum),
	"isAlphaDash": reflect.ValueOf(IsAlphaDash),
	"isBase64":    reflect.ValueOf(IsBase64),
	"isCIDR":      reflect.ValueOf(IsCIDR),
	"isCIDRv4":    reflect.ValueOf(IsCIDRv4),
	"isCIDRv6":    reflect.ValueOf(IsCIDRv6),
	"isDNSName":   reflect.ValueOf(IsDNSName),
	"isDataURI":   reflect.ValueOf(IsDataURI),
	"isEmpty":     reflect.ValueOf(IsEmpty),
	"isHexColor":  reflect.ValueOf(IsHexColor),
	"isISBN10":    reflect.ValueOf(IsISBN10),
	"isISBN13":    reflect.ValueOf(IsISBN13),
	"isJSON":      reflect.ValueOf(IsJSON),
	"isLatitude":  reflect.ValueOf(IsLatitude),
	"isLongitude": reflect.ValueOf(IsLongitude),
	"isMAC":       reflect.ValueOf(IsMAC),
	"isMultiByte": reflect.ValueOf(IsMultiByte),
	"isNumber":    reflect.ValueOf(IsNumber),
	"isNumeric":   reflect.ValueOf(IsNumeric),
	"isCnMobile":  reflect.ValueOf(IsCnMobile),
	//
	"isStringNumber":   reflect.ValueOf(IsStringNumber),
	"hasWhitespace":    reflect.ValueOf(HasWhitespace),
	"isHexadecimal":    reflect.ValueOf(IsHexadecimal),
	"isPrintableASCII": reflect.ValueOf(IsPrintableASCII),
	//
	"isRGBColor": reflect.ValueOf(IsRGBColor),
	"isURL":      reflect.ValueOf(IsURL),
	"isFullURL":  reflect.ValueOf(IsFullURL),
	"isUUID":     reflect.ValueOf(IsUUID),
	"isUUID3":    reflect.ValueOf(IsUUID3),
	"isUUID4":    reflect.ValueOf(IsUUID4),
	"isUUID5":    reflect.ValueOf(IsUUID5),
	// file system
	"pathExists": reflect.ValueOf(PathExists),
	"isDirPath":  reflect.ValueOf(IsDirPath),
	"isFilePath": reflect.ValueOf(IsFilePath),
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

// define validator alias name mapping
var validatorAliases = map[string]string{
	// alias -> real name
	"in":     "enum",
	"not_in": "notIn",
	"size":   "between",
	"range":  "between",

	"in_integers": "inIntegers",
	"in_ints":     "inIntegers",
	"enum_int":    "inIntegers",
	"enum_ints":   "inIntegers",
	"in_strings":  "inStrings",
	"enum_str":    "inStrings",
	"enum_string": "inStrings",
	// type
	"int":       "isInt",
	"integer":   "isInt",
	"uint":      "isUint",
	"bool":      "isBool",
	"float":     "isFloat",
	"map":       "isMap",
	"ints":      "isInts", // []int
	"int_slice": "isInts",
	"str":       "isString",
	"string":    "isString",
	"strings":   "isStrings", // []string
	"str_slice": "isStrings",
	"arr":       "isArray",
	"array":     "isArray",
	"slice":     "isSlice",
	// val
	"regex":  "regexp",
	"eq":     "isEqual",
	"equal":  "isEqual",
	"intEq":  "intEqual",
	"int_eq": "intEqual",
	"ne":     "notEqual",
	"notEq":  "notEqual",
	"not_eq": "notEqual",
	// int compare
	"lte":          "max",
	"gte":          "min",
	"lessThan":     "lt",
	"less_than":    "lt",
	"greaterThan":  "gt",
	"greater_than": "gt",
	// len
	"len":       "length",
	"lenEq":     "length",
	"len_eq":    "length",
	"lengthEq":  "length",
	"length_eq": "length",
	"minLen":    "minLength",
	"min_len":   "minLength",
	"maxLen":    "maxLength",
	"max_len":   "maxLength",
	"minSize":   "minLength",
	"min_size":  "minLength",
	"maxSize":   "maxLength",
	"max_size":  "maxLength",
	// string rune length
	"strlen":      "stringLength",
	"strLen":      "stringLength",
	"str_len":     "stringLength",
	"strLength":   "stringLength",
	"str_length":  "stringLength",
	"runeLen":     "stringLength",
	"rune_len":    "stringLength",
	"runeLength":  "stringLength",
	"rune_length": "stringLength",
	// string contains
	"string_contains": "stringContains",
	"str_contains":    "stringContains",
	"startWith":       "startsWith",
	"start_with":      "startsWith",
	"starts_with":     "startsWith",
	"endWith":         "endsWith",
	"end_with":        "endsWith",
	"ends_with":       "endsWith",
	// string
	"ip":         "isIP",
	"IP":         "isIP",
	"ipv4":       "isIPv4",
	"IPv4":       "isIPv4",
	"ipv6":       "isIPv6",
	"IPv6":       "isIPv6",
	"email":      "isEmail",
	"intStr":     "isIntString",
	"int_str":    "isIntString",
	"strInt":     "isIntString",
	"str_int":    "isIntString",
	"intString":  "isIntString",
	"int_string": "isIntString",
	//
	"stringNum":       "isStringNumber",
	"string_num":      "isStringNumber",
	"strNumber":       "isStringNumber",
	"str_number":      "isStringNumber",
	"strNum":          "isStringNumber",
	"str_num":         "isStringNumber",
	"stringNumber":    "isStringNumber",
	"string_number":   "isStringNumber",
	"hexadecimal":     "isHexadecimal",
	"hasWhitespace":   "hasWhitespace",
	"has_whitespace":  "hasWhitespace",
	"has_wp":          "hasWhitespace",
	"printableASCII":  "isPrintableASCII",
	"printable_ascii": "isPrintableASCII",
	"printable_ASCII": "isPrintableASCII",
	//
	"ascii":      "isASCII",
	"ASCII":      "isASCII",
	"alpha":      "isAlpha",
	"alphaNum":   "isAlphaNum",
	"alpha_num":  "isAlphaNum",
	"alphaDash":  "isAlphaDash",
	"alpha_dash": "isAlphaDash",
	"base64":     "isBase64",
	"cidr":       "isCIDR",
	"CIDR":       "isCIDR",
	"cidr_v4":    "isCIDRv4",
	"CIDRv4":     "isCIDRv4",
	"CIDRv6":     "isCIDRv6",
	"cidr_v6":    "isCIDRv6",
	"dnsName":    "isDNSName",
	"dns_name":   "isDNSName",
	"DNSName":    "isDNSName",
	"dataURI":    "isDataURI",
	"data_URI":   "isDataURI",
	"data_uri":   "isDataURI",
	"empty":      "isEmpty",
	"hexColor":   "isHexColor",
	"hex_color":  "isHexColor",
	"isbn10":     "isISBN10",
	"ISBN10":     "isISBN10",
	"isbn13":     "isISBN13",
	"ISBN13":     "isISBN13",
	"json":       "isJSON",
	"JSON":       "isJSON",
	"lat":        "isLatitude",
	"latitude":   "isLatitude",
	"lon":        "isLongitude",
	"longitude":  "isLongitude",
	"mac":        "isMAC",
	"multiByte":  "isMultiByte",
	"num":        "isNumber",
	"number":     "isNumber",
	"numeric":    "isNumeric",
	"rgbColor":   "isRGBColor",
	"rgb_color":  "isRGBColor",
	"RGBColor":   "isRGBColor",
	"RGB_color":  "isRGBColor",
	"url":        "isURL",
	"URL":        "isURL",
	"fullURL":    "isFullURL",
	"fullUrl":    "isFullURL",
	"uuid":       "isUUID",
	"UUID":       "isUUID",
	"uuid3":      "isUUID3",
	"UUID3":      "isUUID3",
	"uuid4":      "isUUID4",
	"UUID4":      "isUUID4",
	"uuid5":      "isUUID5",
	"UUID5":      "isUUID5",
	"cnMobile":   "isCnMobile",
	"cn_mobile":  "isCnMobile",
	// file system
	"path_exists": "pathExists",
	"pathExist":   "pathExists",
	"path_exist":  "pathExists",
	"filePath":    "isFilePath",
	"filepath":    "isFilePath",
	"unixPath":    "isUnixPath",
	"unix_path":   "isUnixPath",
	"winPath":     "isWinPath",
	"win_path":    "isWinPath",
	// date
	"date":     "isDate",
	"gtDate":   "afterDate",
	"gt_date":  "afterDate",
	"ltDate":   "beforeDate",
	"lt_date":  "beforeDate",
	"gteDate":  "afterOrEqualDate",
	"gte_date": "afterOrEqualDate",
	"lteDate":  "beforeOrEqualDate",
	"lte_date": "beforeOrEqualDate",
	// uploaded file
	"img":        "isImage",
	"image":      "isImage",
	"file":       "isFile",
	"mime":       "inMimeTypes",
	"mimes":      "inMimeTypes",
	"mimeType":   "inMimeTypes",
	"mime_type":  "inMimeTypes",
	"mimeTypes":  "inMimeTypes",
	"mime_types": "inMimeTypes",
	// filed compare
	"eq_field":  "eqField",
	"ne_field":  "neField",
	"gt_field":  "gtField",
	"gte_field": "gteField",
	"lt_field":  "ltField",
	"lte_field": "lteField",
	// requiredXXX
	"required_if":          "requiredIf",
	"required_unless":      "requiredUnless",
	"required_with":        "requiredWith",
	"required_with_all":    "requiredWithAll",
	"required_without":     "requiredWithout",
	"required_without_all": "requiredWithoutAll",
	// other
	"not_contains": "notContains",
}

/*************************************************************
 * register validate rules
 *************************************************************/

// StringRule add field rules by string
// Usage:
// 	v.StringRule("name", "required|string|minLen:6")
// 	// will try convert to int before apply validate.
// 	v.StringRule("age", "required|int|min:12", "toInt")
func (v *Validation) StringRule(field, rule string, filterRule ...string) *Validation {
	rule = strings.TrimSpace(rule)
	rules := stringSplit(strings.Trim(rule, "|:"), "|")
	for _, validator := range rules {
		validator = strings.Trim(validator, ":")
		if validator == "" { // empty
			continue
		}

		// has args "min:12"
		if strings.ContainsRune(validator, ':') {
			list := stringSplit(validator, ":")
			// reassign value
			validator := list[0]
			realName := ValidatorName(validator)
			switch realName {
			// set error message for the field
			case "message":
				// message key like "age.required"
				v.trans.AddMessage(field+"."+validator, list[1])
			// add default value for the field
			case "default":
				v.SetDefValue(field, list[1])
			// eg 'regex:\d{4,6}' dont need split args. args is "\d{4,6}"
			case "regexp":
				v.AddRule(field, validator, list[1])
			// some special validator. need merge args to one.
			case "enum", "notIn", "inIntegers", "inStrings":
				v.AddRule(field, validator, parseArgString(list[1]))
			default:
				args := parseArgString(list[1])
				v.AddRule(field, validator, strings2Args(args)...)
			}
		} else {
			v.AddRule(field, validator)
		}
	}

	if len(filterRule) > 0 {
		v.FilterRule(field, filterRule[0])
	}
	return v
}

// StringRules add multi rules by string map.
// Usage:
// 	v.StringRules(map[string]string{
// 		"name": "required|string|min:12",
// 		"age": "required|int|min:12",
// 	})
func (v *Validation) StringRules(mp MS) *Validation {
	for name, rule := range mp {
		v.StringRule(name, rule)
	}
	return v
}

// ConfigRules add multi rules by string map. alias of StringRules()
// Usage:
// 	v.ConfigRules(map[string]string{
// 		"name": "required|string|min:12",
// 		"age": "required|int|min:12",
// 	})
func (v *Validation) ConfigRules(mp MS) *Validation {
	for name, rule := range mp {
		v.StringRule(name, rule)
	}
	return v
}

// AddRule for current validation
func (v *Validation) AddRule(fields, validator string, args ...interface{}) *Rule {
	return v.addOneRule(fields, validator, ValidatorName(validator), args)
}

// add one Rule for current validation
func (v *Validation) addOneRule(fields, validator, realName string, args []interface{}) *Rule {
	rule := NewRule(fields, validator, args...)

	// init some settings
	rule.realName = realName
	rule.skipEmpty = v.SkipOnEmpty
	// validator name is not "required"
	rule.nameNotRequired = !strings.HasPrefix(realName, "required")

	// append
	v.rules = append(v.rules, rule)
	return rule
}

// AppendRule instance
func (v *Validation) AppendRule(rule *Rule) *Rule {
	rule.realName = ValidatorName(rule.validator)
	rule.skipEmpty = v.SkipOnEmpty
	// validator name is not "required"
	rule.nameNotRequired = !strings.HasPrefix(rule.realName, "required")

	// append
	v.rules = append(v.rules, rule)
	return rule
}
