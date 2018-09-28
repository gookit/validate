package validate

import (
	"reflect"
	"strings"
)

// ConfigValidationFace definition. you can do something on create Validation.
type ConfigValidationFace interface {
	ConfigValidation(v *Validation)
}

// FieldTranslatorFace definition. you can custom field translates.
// Usage:
// 	type User struct {
// 		Name string `json:"name" validate:"required|minLen:5"`
// 	}
//
// 	func (u *User) Translates() map[string]string {
// 		return MS{
// 			"Name": "User name",
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
// 	func (u *User) Messages() map[string]string {
// 		return MS{
// 			"Name.required": "oh! User name is required",
// 		}
// 	}
type CustomMessagesFace interface {
	Messages() map[string]string
}

// StructData definition
type StructData struct {
	// source struct data, from user setting
	src interface{}
	// from reflect source Struct
	value reflect.Value
	// source struct reflect.Type
	valueTpy reflect.Type
	// field values cache
	fieldValues map[string]reflect.Value
	// FilterTag name in the struct tags.
	FilterTag string
	// ValidateTag name in the struct tags.
	ValidateTag string
}

// StructOption definition
type StructOption struct {
	// ValidateTag in the struct tags.
	ValidateTag string
	MethodName  string
}

var (
	cmFaceType = reflect.TypeOf(new(CustomMessagesFace)).Elem()
	ftFaceType = reflect.TypeOf(new(FieldTranslatorFace)).Elem()
	cvFaceType = reflect.TypeOf(new(ConfigValidationFace)).Elem()
)

func newStructData(s interface{}) (*StructData, error) {
	data := &StructData{
		ValidateTag: globalOpt.ValidateTag,
		// init map
		fieldValues: make(map[string]reflect.Value),
	}

	val := reflect.ValueOf(s)

	// fmt.Println(val.NumMethod())
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	typ := val.Type()
	if val.Kind() != reflect.Struct || typ == timeType {
		return data, ErrInvalidData
	}

	data.src = s
	data.value = val
	data.valueTpy = typ

	return data, nil
}

// Create a Validation from the StructData
func (d *StructData) Create(scene ...string) *Validation {
	return d.Validation(scene...)
}

// Validation create from the StructData
func (d *StructData) Validation(scene ...string) *Validation {
	v := NewValidation(d, scene...)

	// collect field validate rules
	d.parseRulesFromTag(v)

	// if has custom config func
	if d.valueTpy.Implements(cvFaceType) {
		fv := d.value.MethodByName("ConfigValidation")
		fv.Call([]reflect.Value{reflect.ValueOf(v)})
	}

	// collect custom field translates config
	if d.valueTpy.Implements(ftFaceType) {
		fv := d.value.MethodByName("Translates")
		vs := fv.Call([]reflect.Value{})
		v.WithTranslates(vs[0].Interface().(map[string]string))
	}

	// collect custom error messages config
	// if reflect.PtrTo(d.valueTpy).Implements(cmFaceType) {
	if d.valueTpy.Implements(cmFaceType) {
		fv := d.value.MethodByName("Messages")
		vs := fv.Call([]reflect.Value{})
		v.WithMessages(vs[0].Interface().(map[string]string))
	}

	return v
}

// parse and collect rules from struct tags.
func (d *StructData) parseRulesFromTag(v *Validation) {
	if d.ValidateTag == "" {
		d.ValidateTag = globalOpt.ValidateTag
	}

	vt := d.valueTpy
	for i := 0; i < vt.NumField(); i++ {
		name := vt.Field(i).Name

		// don't exported field
		if name[0] >= 'a' && name[0] <= 'z' {
			continue
		}

		rule := vt.Field(i).Tag.Get(d.ValidateTag)
		if rule != "" {
			v.StringRule(name, rule)
		}
	}
}

/*************************************************************
 * Struct data operate
 *************************************************************/

// Get value by field name
func (d *StructData) Get(field string) (interface{}, bool) {
	var fv reflect.Value

	// not found field
	if _, ok := d.valueTpy.FieldByName(field); !ok {
		// want get sub struct filed
		if !strings.ContainsRune(field, '.') {
			return nil, false
		}

		ss := strings.SplitN(field, ".", 2)
		field, subField := ss[0], ss[1]

		// check top field is an struct
		tft, ok := d.valueTpy.FieldByName(field)
		if !ok || tft.Type.Kind() != reflect.Struct { // not found OR not a struct
			return nil, false
		}

		fv = d.value.FieldByName(field).FieldByName(subField)
		if !fv.IsValid() { // not found
			return nil, false
		}
	} else {
		fv = d.value.FieldByName(field)
	}

	// check can get value
	if !fv.CanInterface() {
		return nil, false
	}

	return fv.Interface(), true
}

// Set value by field name
func (d *StructData) Set(field string, val interface{}) error {
	fv := d.value.FieldByName(field)

	// field not found
	if !fv.IsValid() {
		return ErrNoField
	}

	// check whether the value of v can be changed.
	if fv.CanSet() {
		fv.Set(reflect.ValueOf(val))
		return nil
	}

	return ErrSetValue
}

// FuncValue get
func (d *StructData) FuncValue(name string) (reflect.Value, bool) {
	fnName := upperFirst(name)
	fv := d.value.MethodByName(fnName)

	return fv, fv.IsValid()
}
