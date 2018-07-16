package validate

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	notInitializedYet = "Validation instance not initialized"
)

// Validate
type Validation struct {
	data       interface{}
	dataM      map[string]interface{}
	rules      map[string]string
	validators map[string]*Validator

	validated bool
}

// Option contains the options that a Validator instance will use.
// It is passed to the New() function
type Option struct {
	TagName      string // "validate" "v"
	FieldNameTag string // "json"
}

func (v *Validation) initCheck() {
	if v == nil {
		panic(notInitializedYet)
	}
}

func (v *Validation) AddValidator() {

}

func (v *Validation) ValidateMap(data map[string]interface{}, vFields ...string) {

}

func (v *Validation) ValidateStruct(data interface{}, vFields ...string) {

}

func (v *Validation) DataTo(s interface{}) {

}

func (v *Validation) MapTo(s interface{}) {

}

func (v *Validation) Safe(field string) {

}

func (v *Validation) SafeData() (data map[string]interface{}) {
	return
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
	Name    string
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

// Map validate the input map data
func Map(data map[string]interface{}) {

}

// Struct validate the input data
func Struct(data interface{}) {

}
