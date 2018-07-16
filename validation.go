package validation

import (
	"bytes"
	"fmt"
	"strings"
)

// Validate
// Validation
type Validation struct {
	data interface{}
	dataM map[string]interface{}
	rules map[string]string
	validators map[string]*Validator
}

// Option contains the options that a Validator instance will use.
// It is passed to the New() function
type Option struct {
	TagName      string // "validate" "v"
	FieldNameTag string // "json"
}

func (v *Validation) AddValidator()  {

}

func (v *Validation) Map()  {

}

func (v *Validation) Struct()  {

}

// Errors
type Errors map[string]*FieldError

// String
func (es Errors) String() string {
	buff := bytes.NewBufferString(blank)

	for key, err := range es {
		buff.WriteString(fmt.Sprintf(fieldErrMsg, key, err.Field, err.Tag))
		buff.WriteString("\n")
	}

	return strings.TrimSpace(buff.String())
}

// FieldError
type FieldError struct {
	Name string
	Message string
}

// create new Validation
func New() *Validation {
	return &Validation{}
}

// default validation
var d = &Validation{}

// V validate the input data
func V(data interface{}) {

}
