package validate

import (
	"bytes"
	"fmt"
	"strings"
)

// defTranslates internal error message for some rules.
var defTranslates = map[string]string{
	"_": "data did not pass validate", // default message

	"min": "%s value min is %d",
	"max": "%s value max is %d",

	"minSize": "%s value min size is %d",
	"maxSize": "%s value max size is %d",

	"range": "%s value must be in the range %d - %d",
}

// some validator alias name
var validatorAliases = map[string]string{
	"int": "integer",
	"num": "number",
	"str": "string",
	"map": "mapping",
	"arr": "array",

	"regex": "regexp",
}

// Translator
type Translator struct {
	data map[string]string
}

// NewTranslator
func NewTranslator() *Translator {
	return &Translator{defTranslates}
}

// Add new data to translator
func (t *Translator) Add(data map[string]string) {
	for n, m := range data {
		t.data[n] = m
	}
}

func (t *Translator) Tr(key string, args ...interface{}) string {
	if msg, ok := t.data[key]; ok {
		return fmt.Sprintf(msg, args...)
	}

	return t.data["_"]
}

// Reset translator to default
func (t *Translator) Reset() {
	t.data = defTranslates
}

// Errors [map["filed": "name", "msg": "err msg"], ...]
type Errors []map[string]string
type ErrorFields map[string]int

// FieldError
type FieldError struct {
	Name    string
	Message string
}

// Empty no error
func (es Errors) Empty() bool {
	return len(es) == 0
}

// First
func (es Errors) First() map[string]string {
	if len(es) > 0 {
		return es[0]
	}

	return nil
}

// FirstMsg
func (es Errors) FirstMsg() string {
	if len(es) > 0 {
		return es[0]["msg"]
	}

	return ""
}

// All
func (es Errors) All() []map[string]string {
	return es
}

// Field get errors for the field
func (es Errors) Field(field string) (fieldErs []map[string]string) {
	if len(es) == 0 {
		return
	}

	for _, item := range es {
		if item["field"] == field {
			fieldErs = append(fieldErs, item)
		}
	}

	return
}

// String
func (es Errors) String() string {
	buff := bytes.NewBufferString(blank)

	for _, err := range es {
		buff.WriteString(fmt.Sprintf("%s: %s\n", err["field"], err["msg"]))
	}

	return strings.TrimSpace(buff.String())
}
