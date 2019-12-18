package validate

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const defaultErrMsg = " field did not pass validation"

// some internal error definition
var (
	ErrSetValue = errors.New("set value failure")
	// ErrNoData = errors.New("validate: no any data can be collected")
	ErrNoField     = errors.New("field not exist in the source data")
	ErrEmptyData   = errors.New("please input data use for validate")
	ErrInvalidData = errors.New("invalid input data")
)

/*************************************************************
 * Validate Errors
 *************************************************************/

// example {validator0: message0, validator1: message1}
type fieldErrors map[string]string

func (fe fieldErrors) one() string {
	for _, msg := range fe {
		return msg
	}
	return "" // should never exec.
}

func (fe fieldErrors) string() string {
	var ss []string
	for name, msg := range fe {
		ss = append(ss, " "+name+": "+msg)
	}

	return strings.Join(ss, "\n")
}

// Errors validate errors definition
// Example:
// 	{
// 		"field": {validator: message, validator1: message1}
// 	}
type Errors map[string]fieldErrors

// Empty no error
func (es Errors) Empty() bool {
	return len(es) == 0
}

// Add a error for the field
func (es Errors) Add(field, validator, message string) {
	if _, ok := es[field]; ok {
		es[field][validator] = message
	} else {
		es[field] = fieldErrors{validator: message}
	}
}

// One returns an random error message text
func (es Errors) One() string {
	if len(es) > 0 {
		for _, fe := range es {
			return fe.one()
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
		buf.WriteString(fmt.Sprintf("%s:\n%s\n", field, fe.string()))
	}

	return strings.TrimSpace(buf.String())
}

// Field get all errors for the field
func (es Errors) Field(field string) map[string]string {
	return es[field]
}

// FieldOne returns an error message for the field
func (es Errors) FieldOne(field string) string {
	if fe, ok := es[field]; ok {
		return fe.one()
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
	"min": "{field} min value is %d",
	"max": "{field} max value is %d",
	// type check: int
	"isInt":  "{field} value must be an integer",
	"isInt1": "{field} value must be an integer and mix value is %d",      // has min check
	"isInt2": "{field} value must be an integer and in the range %d - %d", // has min, max check
	"isInts": "{field} value must be an int slice",
	"isUint": "{field} value must be an unsigned integer(>= 0)",
	// type check: string
	"isString":  "{field} value must be an string",
	"isString1": "{field} value must be an string and min length is %d", // has min len check
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
	// required
	"required":             "{field} is required and not empty",
	"required_if":          "{field} is required when {args0} is {args1end}",
	"required_unless":      "{field} field is required unless {args0} is in {args1end}",
	"required_with":        "{field} field is required when {values} is present",
	"required_with_all":    "{field} field is required when {values} is present",
	"required_without":     "{field} field is required when {values} is not present",
	"required_without_all": "{field} field is required when none of {values} are present",
	// field compare
	"eqField":  "{field} value must be equal the field %s",
	"neField":  "{field} value cannot be equal the field %s",
	"ltField":  "{field} value should be less than the field %s",
	"lteField": "{field} value should be less than or equal to field %s",
	"gtField":  "{field} value must be greater the field %s",
	"gteField": "{field} value should be greater or equal to field %s",
}

/*************************************************************
 * Error messages translator
 *************************************************************/

// Translator definition
type Translator struct {
	// field map {"field name": "display name"}
	fieldMap map[string]string
	// message data map
	messages map[string]string
}

// NewTranslator instance
func NewTranslator() *Translator {
	newMessages := make(map[string]string)
	for k, v := range builtinMessages {
		newMessages[k] = v
	}

	return &Translator{
		fieldMap: make(map[string]string),
		messages: newMessages,
	}
}

// Reset translator to default
func (t *Translator) Reset() {
	newMessages := make(map[string]string)
	for k, v := range builtinMessages {
		newMessages[k] = v
	}

	t.messages = newMessages
	t.fieldMap = make(map[string]string)
}

// FieldMap data get
func (t *Translator) FieldMap() map[string]string {
	return t.fieldMap
}

// AddMessages data to translator
func (t *Translator) AddMessages(data map[string]string) {
	for n, m := range data {
		t.messages[n] = m
	}
}

// AddFieldMap config field data.
// If you want to display in the field with the original field is not the same
func (t *Translator) AddFieldMap(fieldMap map[string]string) {
	for name, showName := range fieldMap {
		t.fieldMap[name] = showName
	}
}

// AddMessage to translator
func (t *Translator) AddMessage(key, msg string) {
	t.messages[key] = msg
}

// HasField name in the t.fieldMap
func (t *Translator) HasField(field string) bool {
	_, ok := t.fieldMap[field]
	return ok
}

// HasMessage key in the t.messages
func (t *Translator) HasMessage(key string) bool {
	_, ok := t.messages[key]
	return ok
}

// Message get by validator name and field name.
func (t *Translator) Message(validator, field string, args ...interface{}) (msg string) {
	var ok bool
	if rName, has := validatorAliases[validator]; has {
		msg, ok = t.format(rName, field, args...)
		if ok {
			return
		}
	}

	msg, ok = t.format(validator, field, args...)

	// not found, fallback - use default error message
	if !ok {
		// get field display name.
		if trName, ok := t.fieldMap[field]; ok {
			field = trName
		}
		return field + defaultErrMsg
	}
	return
}

// format message for the validator
func (t *Translator) format(validator, field string, args ...interface{}) (string, bool) {
	argLen := len(args)
	errMsg := t.findMessage(validator, field, argLen)
	if errMsg == "" {
		return "", false
	}

	// not contains vars
	if !strings.ContainsRune(errMsg, '{') {
		return fmt.Sprintf(errMsg, args...), true
	}

	// get field display name.
	if trName, ok := t.fieldMap[field]; ok {
		field = trName
	}

	if argLen > 0 {
		// if need call fmt.Sprintf
		if strings.ContainsRune(errMsg, '%') {
			errMsg = fmt.Sprintf(errMsg, args...)
		}

		msgArgs := []string{
			"{field}", field,
			"{values}", fmt.Sprintf("%v", args),
			"{args0}", fmt.Sprintf("%v", args[0]),
		}

		// {args1end} -> args[1:]
		if argLen > 1 {
			msgArgs = append(msgArgs, "{args1end}", fmt.Sprintf("%v", args[1:]))
		}

		// replace message vars
		errMsg = strings.NewReplacer(msgArgs...).Replace(errMsg)
	} else {
		errMsg = strings.Replace(errMsg, "{field}", field, 1)
	}

	return errMsg, true
}

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
