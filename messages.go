package validate

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/strutil"
)

const defaultErrMsg = " field did not pass validation"

// some internal error definition
var (
	ErrSetValue = errors.New("set value failure")
	ErrNoField  = errors.New("field not exist in the source data")

	ErrEmptyData   = errors.New("please input data use for validate")
	ErrInvalidData = errors.New("invalid input data")
)

/*************************************************************
 * Validate Errors
 *************************************************************/

// Errors validate errors definition
//
// Example:
// 	{
// 		"field": {validator: message, validator1: message1}
// 	}
type Errors map[string]MS

// Empty no error
func (es Errors) Empty() bool {
	return len(es) == 0
}

// Add a error for the field
func (es Errors) Add(field, validator, message string) {
	if _, ok := es[field]; ok {
		es[field][validator] = message
	} else {
		es[field] = MS{validator: message}
	}
}

// One returns an random error message text
func (es Errors) One() string {
	return es.Random()
}

// OneError returns an random error
func (es Errors) OneError() error {
	return errorx.Raw(es.Random())
}

// Random returns an random error message text
func (es Errors) Random() string {
	if len(es) > 0 {
		for _, fe := range es {
			return fe.One()
		}
	}
	return ""
}

// All get all errors data
func (es Errors) All() map[string]map[string]string {
	mm := make(map[string]map[string]string, len(es))

	for field, fe := range es {
		mm[field] = fe
	}
	return mm
}

// Error string get
func (es Errors) Error() string {
	return es.String()
}

// String errors to string
func (es Errors) String() string {
	buf := new(bytes.Buffer)
	for field, fe := range es {
		buf.WriteString(fmt.Sprintf("%s:\n%s\n", field, fe.String()))
	}

	return strings.TrimSpace(buf.String())
}

// HasField in the errors
func (es Errors) HasField(field string) bool {
	_, ok := es[field]
	return ok
}

// Field gets all errors for the field
func (es Errors) Field(field string) map[string]string {
	return es[field]
}

// FieldOne returns an error message for the field
func (es Errors) FieldOne(field string) string {
	if fe, ok := es[field]; ok {
		return fe.One()
	}
	return ""
}

/*************************************************************
 * Validator error messages
 *************************************************************/

// default internal error messages for all validators.
var builtinMessages = map[string]string{
	"_": "{field}" + defaultErrMsg, // default message
	// builtin
	"_validate": "{field} did not pass validate", // default validate message
	"_filter":   "{field} data is invalid",       // data filter error
	// int value
	"min": "{field} min value is %v",
	"max": "{field} max value is %v",
	// type check: int
	"isInt":  "{field} value must be an integer",
	"isInt1": "{field} value must be an integer and mix value is %d",      // has min check
	"isInt2": "{field} value must be an integer and in the range %d - %d", // has min, max check
	"isInts": "{field} value must be an int slice",
	"isUint": "{field} value must be an unsigned integer(>= 0)",
	// type check: string
	"isString":  "{field} value must be a string",
	"isString1": "{field} value must be a string and min length is %d", // has min len check
	// length
	"minLength": "{field} min length is %d",
	"maxLength": "{field} max length is %d",
	// string length. calc rune
	"stringLength":  "{field} length must be in the range %d - %d",
	"stringLength1": "{field} min length is %d",
	"stringLength2": "{field} length must be in the range %d - %d",

	"isURL":     "{field} must be an valid URL address",
	"isFullURL": "{field} must be an valid full URL address",

	"isFile":  "{field} must be an uploaded file",
	"isImage": "{field} must be an uploaded image file",

	"enum":  "{field} value must be in the enum %v",
	"range": "{field} value must be in the range %d - %d",
	// int compare
	"lt": "{field} value should less than %v",
	"gt": "{field} value should greater the %v",
	// required
	"required":           "{field} is required and not empty",
	"requiredIf":         "{field} is required when {args0} is {args1end}",
	"requiredUnless":     "{field} field is required unless {args0} is in {args1end}",
	"requiredWith":       "{field} field is required when {values} is present",
	"requiredWithAll":    "{field} field is required when {values} is present",
	"requiredWithout":    "{field} field is required when {values} is not present",
	"requiredWithoutAll": "{field} field is required when none of {values} are present",
	// field compare
	"eqField":  "{field} value must be equal the field %s",
	"neField":  "{field} value cannot be equal the field %s",
	"ltField":  "{field} value should be less than the field %s",
	"lteField": "{field} value should be less than or equal to field %s",
	"gtField":  "{field} value must be greater the field %s",
	"gteField": "{field} value should be greater or equal to field %s",
	// data type
	"bool":    "{field} value must be a bool",
	"float":   "{field} value must be a float",
	"slice":   "{field} value must be a slice",
	"map":     "{field} value must be a map",
	"array":   "{field} value  must be an array",
	"strings": "{field} value must be a []string",
	"notIn":   "{field} value must not in the given enum list %d",
	//
	"contains":    "{field} value does not contain this %s",
	"notContains": "{field} value contains the given %s",
	"startsWith":  "{field} value does not start with the given %s",
	"endsWith":    "{field} value does not end with the given %s",
	"email":       "{field} value is invalid mail",
	"regex":       "{field} value does not pass regex check",
	"file":        "{field} value must be a file",
	"image":       "{field} value must be an image",
	// date
	"date":    "{field} value should be an date string",
	"gtDate":  "{field} value should be after %s",
	"ltDate":  "{field} value should be before %s",
	"gteDate": "{field} value should be after or equal to %s",
	"lteDate": "{field} value should be before or equal to %s",
	// check char
	"hasWhitespace":  "{field} value should contains spaces",
	"ascii":          "{field} value should be an ASCII string",
	"alpha":          "{field} value contains only alpha char",
	"alphaNum":       "{field} value contains only alpha char and num",
	"alphaDash":      "{field} value contains only letters,num,dashes (-) and underscores (_)",
	"multiByte":      "{field} value should be a multiByte string",
	"base64":         "{field} value should be a base64 string",
	"dnsName":        "{field} value should be a DNS string",
	"dataURI":        "{field} value should be a DataURL string",
	"empty":          "{field} value should be empty",
	"hexColor":       "{field} value should be a color string in hexadecimal",
	"hexadecimal":    "{field} value should be a hexadecimal string",
	"json":           "{field} value should be a json string",
	"lat":            "{field} value should be latitude coordinates",
	"lon":            "{field} value should be longitude coordinates",
	"num":            "{field} value should be a num (>=0) string.",
	"mac":            "{field} value should be mac string",
	"cnMobile":       "{field} value should be string of Chinese 11-digit mobile phone numbers",
	"printableASCII": "{field} value should be a printable ASCII string",
	"rgbColor":       "{field} value should be a RGB color string",
	"fullURL":        "{field} value should be a complete URL string",
	"full":           "{field} value should be a URL string",
	"ip":             "{field} value should be an ip (v4 or v6) string",
	"ipv4":           "{field} value should be an ipv4 string",
	"ipv6":           "{field} value should be an ipv6 string",
	"CIDR":           "{field} value should be a CIDR string",
	"CIDRv4":         "{field} value should be a CIDRv4 string",
	"CIDRv6":         "{field} value should be a CIDRv6 string",
	"uuid":           "{field} value should be a UUID string",
	"uuid3":          "{field} value should be a UUID3 string",
	"uuid4":          "{field} value should be a UUID4 string",
	"uuid5":          "{field} value should be a UUID5 string",
	"filePath":       "{field} value should be an existing file path",
	"unixPath":       "{field} value should be a unix path string",
	"winPath":        "{field} value should be a windows path string",
	"isbn10":         "{field} value should be a isbn10 string",
	"isbn13":         "{field} value should be a isbn13 string",
}

// AddGlobalMessages add global builtin messages
func AddGlobalMessages(mp map[string]string) {
	for name, msg := range mp {
		builtinMessages[name] = msg
	}
}

// AddBuiltinMessages alias of the AddGlobalMessages()
func AddBuiltinMessages(mp map[string]string) {
	for name, msg := range mp {
		builtinMessages[name] = msg
	}
}

// BuiltinMessages get builtin messages
func BuiltinMessages() map[string]string {
	return builtinMessages
}

/*************************************************************
 * Error messages translator
 *************************************************************/

// StdTranslator for default. TODO
// var StdTranslator = NewTranslator()

// Translator definition
type Translator struct {
	// the field output name, use for Errors key.
	// format: {"field": "output name"}
	fieldMap map[string]string
	// the field translate name in message.
	// format: {"field": "translate name"}
	labelMap map[string]string
	// the error message data map.
	// key allow:
	// TODO
	messages map[string]string
}

// NewTranslator instance
func NewTranslator() *Translator {
	tr := &Translator{}
	tr.Reset()
	return tr
}

// Reset translator to default
func (t *Translator) Reset() {
	newMessages := make(map[string]string)
	for k, v := range builtinMessages {
		newMessages[k] = v
	}

	t.messages = newMessages
	t.labelMap = make(map[string]string)
	t.fieldMap = make(map[string]string)
}

// FieldMap data get
func (t *Translator) FieldMap() map[string]string {
	return t.fieldMap
}

// AddFieldMap config field output name data.
// If you want to display in the field with the original field is not the same
func (t *Translator) AddFieldMap(fieldMap map[string]string) {
	for name, outName := range fieldMap {
		t.fieldMap[name] = outName
	}
}

// HasField name in the t.fieldMap.
func (t *Translator) HasField(field string) bool {
	_, ok := t.fieldMap[field]
	return ok
}

// FieldName get in the t.fieldMap
func (t *Translator) FieldName(field string) string {
	if trName, ok := t.fieldMap[field]; ok {
		field = trName
	}
	return field
}

// LabelMap data get
func (t *Translator) LabelMap() map[string]string {
	return t.labelMap
}

func (t *Translator) addLabelName(field, labelName string) {
	if labelName != "" {
		t.labelMap[field] = labelName
	}
}

// AddLabelMap config field translate data map.
// If you want to display in the field with the original field is not the same
func (t *Translator) AddLabelMap(fieldMap map[string]string) {
	for name, labelName := range fieldMap {
		t.addLabelName(name, labelName)
	}
}

// HasLabel name in the t.labelMap
func (t *Translator) HasLabel(field string) bool {
	_, ok := t.labelMap[field]
	return ok
}

// LabelName get label name from the t.labelMap, fallback get output name from t.fieldMap
func (t *Translator) LabelName(field string) string {
	if label, ok := t.labelMap[field]; ok {
		return label
	}
	return t.FieldName(field)
}

// LookupLabel get label name from the t.labelMap,
// fallback get output name from t.fieldMap. if not
// found, return "", false
func (t *Translator) LookupLabel(field string) (string, bool) {
	if label, ok := t.labelMap[field]; ok {
		return label, true
	}

	fName, ok := t.fieldMap[field]
	return fName, ok
}

// AddMessages data to translator
func (t *Translator) AddMessages(data map[string]string) {
	for n, m := range data {
		t.messages[n] = m
	}
}

// AddMessage to translator
func (t *Translator) AddMessage(key, msg string) {
	t.messages[key] = msg
}

// HasMessage key in the t.messages
func (t *Translator) HasMessage(key string) bool {
	_, ok := t.messages[key]
	return ok
}

// Message get by validator name and field name.
func (t *Translator) Message(validator, field string, args ...interface{}) (msg string) {
	argLen := len(args)
	errMsg := t.findMessage(validator, field, argLen)
	if errMsg == "" {
		// try check "validator" is an alias name
		if rName, has := validatorAliases[validator]; has {
			errMsg = t.findMessage(rName, field, argLen)
		}

		// not found, fallback - use default error message
		if errMsg == "" {
			return t.LabelName(field) + defaultErrMsg
		}
	}

	return t.format(errMsg, field, args)
}

// format message for the validator
func (t *Translator) format(errMsg, field string, args []interface{}) string {
	argLen := len(args)

	// fix: #111 argN maybe is a field name
	for i, arg := range args {
		if name, ok := arg.(string); ok {
			if lName, ok := t.LookupLabel(name); ok {
				args[i] = lName
			}
		}
	}

	// not contains vars. eg: {field}
	if !strings.ContainsRune(errMsg, '{') {
		// whether you need call fmt.Sprintf
		if argLen > 0 && strings.ContainsRune(errMsg, '%') {
			errMsg = fmt.Sprintf(errMsg, args...)
		}
		return errMsg
	}

	// get field display label name.
	field = t.LabelName(field)
	if argLen > 0 {
		// whether you need call fmt.Sprintf
		if strings.ContainsRune(errMsg, '%') {
			errMsg = fmt.Sprintf(errMsg, args...)
		}

		msgArgs := []string{
			"{field}", field,
			"{values}", sliceToString(args),
			"{args0}", strutil.MustString(args[0]),
		}

		// {args1end} -> args[1:]
		if argLen > 1 {
			msgArgs = append(msgArgs, "{args1end}", sliceToString(args[1:]))
		}

		// replace message vars
		errMsg = strings.NewReplacer(msgArgs...).Replace(errMsg)
	} else {
		errMsg = strings.Replace(errMsg, "{field}", field, 1)
	}

	return errMsg
}

// find message template.
func (t *Translator) findMessage(validator, field string, argLen int) string {
	// - format1: "field name" + "." + "validator name".
	// eg: "age.isInt" "name.required"
	fullKey := field + "." + validator

	// validator support variadic params. eg: isInt1 isInt2
	if argLen > 0 {
		lenStr := strconv.Itoa(argLen)

		// eg: "age.isInt1" "age.isInt2"
		newFullKey := fullKey + lenStr
		if msg, ok := t.messages[newFullKey]; ok {
			return msg
		}

		// eg: "isInt1" "isInt2"
		newNameKey := validator + lenStr
		if msg, ok := t.messages[newNameKey]; ok {
			return msg
		}
	}

	// use fullKey find
	if msg, ok := t.messages[fullKey]; ok {
		return msg
	}

	// only validator name. "required"
	if msg, ok := t.messages[validator]; ok {
		return msg
	}
	return ""
}
