package validate

const (
	notInitializedYet = "Validation instance not initialized"
)

// Validate
type Validation struct {
	// structure data
	dataS interface{}
	// map data
	dataM map[string]interface{}

	rules map[string]string

	errors Errors

	aliases    map[string]string
	validators map[string]*Validator

	dataIsMap bool
	dataIsStruct bool

	validated bool
	translator *Translator

	stopOnError bool
}

// Option contains the options that a Validator instance will use.
// It is passed to the New() function
type Option struct {
	TagName      string // "validate" "v"
	FieldNameTag string // "json"

	StopOnError bool
}

// Rules
type Rules map[string]interface{}

func (v *Validation) SetRules(rules Rules) {

}


func (v *Validation) initCheck() {
	if v == nil {
		panic(notInitializedYet)
	}
}

func (v *Validation) StopOnError(stopOnError bool) {
	v.stopOnError = stopOnError
}

// AddValidator
func (v *Validation) AddValidator() {

}

func (v *Validation) Validate(vFields ...string) {
	v.initCheck()
}

// ValidateMap
func (v *Validation) ValidateMap(data map[string]interface{}, vFields ...string) bool {
	v.initCheck()
	v.dataIsMap = true

	return v.IsOk()
}

// ValidateStruct
func (v *Validation) ValidateStruct(data interface{}, vFields ...string) bool {
	v.initCheck()
	v.dataIsStruct = true

	return v.IsOk()
}

// IsOk
func (v *Validation) IsOk() bool {
	return v.errors.Empty()
}

// IsFail
func (v *Validation) IsFail() bool {
	return !v.errors.Empty()
}

// DataTo
func (v *Validation) DataTo(s interface{}) {

}

// MapTo
func (v *Validation) MapTo(s interface{}) {

}

// Safe
func (v *Validation) Safe(field string) {

}

// Destroy
func (v *Validation) Destroy() {
	v.translator = NewTranslator()
	v.rules = nil
	v.errors = Errors{}
}

// AddTranslates
func (v *Validation) AddTranslates(translates map[string]string) {
	if len(translates) > 0 {
		v.translator.Add(translates)
	}

	v.translator.Add(translates)
}

func (v *Validation) SafeData() (data map[string]interface{}) {
	return
}

// create new Validation
func New() *Validation {
	return &Validation{
		translator: NewTranslator(),
	}
}

// Map validate the input map data
func Map(data map[string]interface{}, rules Rules, translates map[string]string) Errors {
	v := New()

	v.AddTranslates(translates)
	v.ValidateMap(data)

	return v.errors
}

// Struct validate the input data
func Struct(data interface{}, rules Rules, translates map[string]string) Errors {
	v := New()

	v.AddTranslates(translates)
	v.ValidateStruct(data)

	return v.errors
}
