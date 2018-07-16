package validation

// IValidator
type IValidator interface {
	Name() string
	Validate() error
}

// Validator
type Validator struct {
	Name string
	Func func() error
}

func (v *Validator) Name() string {
	panic("implement me")
}

func (v *Validator) Validate() error {
	panic("implement me")
}

