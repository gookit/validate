package validate

import "reflect"

const defaultTag = "validate"

// StructData definition
type StructData struct {
	// source struct data, from user setting
	source interface{}
	// from reflect Map or Struct
	value reflect.Value
	// field values cache
	fieldValues map[string]reflect.Value
	// TagName in the struct tags.
	TagName string
}

// FieldTranslatorFace definition. you can custom field translates.
// Usage:
// 	type User struct {
// 		Name string `json:"name" validate:"required|minLen:5"`
// 	}
//
//	func (u *User) Translates() map[string]string {
//		return SMap{
//			"Name": "User name",
// 		}
// 	}
type FieldTranslatorFace interface {
	Translates() map[string]string
}

// CustomMessagesFace definition. you can custom validator error messages.
// Usage:
// 	type User struct {
// 		Name string `json:"name" validate:"required|minLen:5"`
// 	}
//
//	func (u *User) Messages() map[string]string {
//		return SMap{
//			"Name.required": "oh! User name is required",
// 		}
// 	}
type CustomMessagesFace interface {
	Messages() map[string]string
}

// StructOption definition
type StructOption struct {
	// TagName in the struct tags.
	TagName    string
	MethodName string
}

var (
	ftFaceType  = reflect.TypeOf(new(FieldTranslatorFace))
	msgFaceType = reflect.TypeOf(new(FieldTranslatorFace))
)

// Create a Validation from the StructData
func (d *StructData) Create(scene ...string) *Validation {
	return d.Validation(scene...)
}

// New a Validation from the StructData
func (d *StructData) New(scene ...string) *Validation {
	return d.Validation(scene...)
}

// Validation create from the StructData
func (d *StructData) Validation(scene ...string) *Validation {
	v := NewValidation(d, scene...)

	// collect field validate rules
	d.parseRulesFromTag(v)

	// collect custom field translates config
	if d.value.Type().Implements(ftFaceType) {
		fv := d.value.MethodByName("Translates")
		vs := fv.Call([]reflect.Value{})
		mp := vs[0].Interface().(map[string]string)
		v.WithTranslates(mp)
	}

	// collect custom error messages config
	if d.value.Type().Implements(msgFaceType) {
		fv := d.value.MethodByName("Messages")
		vs := fv.Call([]reflect.Value{})
		mp := vs[0].Interface().(map[string]string)
		v.WithMessages(mp)
	}

	return v
}

// parse and collect rules from struct tags.
func (d *StructData) parseRulesFromTag(v *Validation) {
	if d.TagName == "" {
		d.TagName = defaultTag
	}

	vt := d.value.Type()
	for i := 0; i < vt.NumField(); i++ {
		name := vt.Field(i).Name

		// don't exported field
		if name[0] >= 'a' && name[0] <= 'z' {
			continue
		}

		rule := vt.Field(i).Tag.Get(d.TagName)
		if rule != "" && rule != "-" {
			v.StringRule(name, rule)
		}
	}
}

/*************************************************************
 * Struct data operate
 *************************************************************/

// Get value by field name
func (d *StructData) Get(field string) (interface{}, bool) {
	field = upperFirst(field)
	fv := d.value.FieldByName(field)
	// fv.IsNil()
	if !fv.IsValid() { // not found
		return nil, false
	}

	return fv.Interface(), true
}

// Set
func (d *StructData) Set(field string, val interface{}) error {
	panic("implement me")
}

// FuncValue get
func (d *StructData) FuncValue(name string) (reflect.Value, bool) {
	fnName := upperFirst(name)
	fv := d.value.MethodByName(fnName)

	return fv, fv.IsValid()
}
