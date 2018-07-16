package validation

// Validation
type Validation struct {
	data interface{}
	rules map[string]string
	validators map[string]Validator
}

// IValidator
type IValidator interface {
	Name() string
	Validate() error
}

// Validator
type Validator struct {
	Name string
	Validate func() error
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
