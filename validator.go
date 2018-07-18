package validate

// IValidator
type IValidator interface {
	GetName() string
	Run() error
}

// Validator
type Validator struct {
	Name string
	Func func() error
}

func (v *Validator) GetName() string {
	panic("implement me")
}

func (v *Validator) Run() error {
	panic("implement me")
}

// global user validators
var gUserValidators = map[string]interface{}{}

func AddValidator() {

}
