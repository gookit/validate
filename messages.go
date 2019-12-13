package validate

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// some internal error definition
var (
	ErrSetValue = errors.New("set value failure")
	// ErrNoData = errors.New("validate: no any data can be collected")
	ErrNoField     = errors.New("field not exist in the source data")
	ErrEmptyData   = errors.New("please input data use for validate")
	ErrInvalidData = errors.New("invalid input data")
)

/*************************************************************
 * errors messages
 *************************************************************/

// Errors definition.
// Example:
// 	{
// 		"field": ["error msg 0", "error msg 1"]
// 	}
// TODO change error format [field: {validator: message}]
type Errors map[string][]string

// Empty no error
func (es Errors) Empty() bool {
	return len(es) == 0
}

// Add a error for the field
func (es Errors) Add(field, message string) {
	if ss, ok := es[field]; ok {
		es[field] = append(ss, message)
	} else {
		es[field] = []string{message}
	}
}

// One returns a random error message text
func (es Errors) One() string {
	if len(es) > 0 {
		for _, ss := range es {
			return ss[0]
		}
	}
	return ""
}

// Get returns a error message for the field
func (es Errors) Get(field string) string {
	if ms, ok := es[field]; ok {
		return ms[0]
	}

	return ""
}

// Field get all errors for the field
func (es Errors) Field(field string) (fieldErs []string) {
	return es[field]
}

// All all error get
func (es Errors) All() map[string][]string {
	return es
}

// Error string get
func (es Errors) Error() string {
	return es.String()
}

// String errors to string
func (es Errors) String() string {
	buf := new(bytes.Buffer)
	for field, ms := range es {
		buf.WriteString(fmt.Sprintf("%s:\n %s\n", field, strings.Join(ms, "\n ")))
	}

	return strings.TrimSpace(buf.String())
}

/*************************************************************
 * validators messages
 *************************************************************/

// default internal error message for some rules.
var defMessages = map[string]string{
	"_":       "{field} did not pass validate", // default message
	"_filter": "{field} data is invalid",       // data filter error
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
	"required":    "{field} is required",
	"required_if": "{field} is required when %v is {sArgs}",
	// field compare
	"eqField":  "{field} value must be equal the field %s",
	"neField":  "{field} value cannot be equal the field %s",
	"ltField":  "{field} value should be less than the field %s",
	"lteField": "{field} value should be less than or equal to field %s",
	"gtField":  "{field} value must be greater the field %s",
	"gteField": "{field} value should be greater or equal to field %s",
}

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
	for k, v := range defMessages {
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
	for k, v := range defMessages {
		newMessages[k] = v
	}

	t.messages = newMessages
	t.fieldMap = make(map[string]string)
}

// FieldMap data get
func (t *Translator) FieldMap() map[string]string {
	return t.fieldMap
}

// LoadMessages data to translator
func (t *Translator) LoadMessages(data map[string]string) {
	for n, m := range data {
		t.messages[n] = m
	}
}

// AddFieldMap config data
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
	}

	if !ok {
		msg, ok = t.format(validator, field, args...)
		// fallback, use default message
		if !ok {
			msg = t.messages["_"]
		}
	}

	if !strings.Contains(msg, "{") {
		return
	}

	// get field translate.
	if trName, ok := t.fieldMap[field]; ok {
		field = trName
	}

	strArgs := fmt.Sprintf("%v", args)
	subArgs := fmt.Sprintf("%v", args[1:])

	msg = strings.Replace(msg, "{field}", field, 1)
	msg = strings.Replace(msg, "{args}", strArgs, 1)
	msg = strings.Replace(msg, "{sArgs}", subArgs, 1)

	return
}

// format message for the validator
func (t *Translator) format(validator, field string, args ...interface{}) (msg string, ok bool) {
	// validator support variadic params. eg: isInt1 isInt2
	if ln := len(args); ln > 0 {
		newKey := fmt.Sprint(validator, ln)

		if msg, ok = t.messages[newKey]; ok {
			msg = fmt.Sprintf(msg, args...)
			return
		}
	}

	key := field + "." + validator

	// "field.required"
	if msg, ok = t.messages[key]; ok {
		msg = fmt.Sprintf(msg, args...)
		// only validator name. "required"
	} else if msg, ok = t.messages[validator]; ok {
		msg = fmt.Sprintf(msg, args...)
	}
	return
}
