package validate

import "reflect"

const defaultTag = "validate"

// StructData
type StructData struct {
	// Struct data from user setting
	Struct interface{}
	// from reflect Map or Struct
	value reflect.Value
	// TagName in the struct tags.
	TagName string
}

// Create a Validation from data
func (d *StructData) Create(scene ...string) *Validation {
	return d.Validation(scene...)
}

// New a Validation from data
func (d *StructData) New(scene ...string) *Validation {
	return d.Validation(scene...)
}

// Validation create from data
func (d *StructData) Validation(scene ...string) *Validation {
	v := NewValidation(d, scene...)

	d.addRulesFromTag(v)

	// custom field translates config
	mName := "Translates"
	if m := d.collectTranslates(mName); m != nil {
		v.WithTranslates(m)
	}

	// custom error message config
	mName = "Messages"
	if m := d.collectTranslates(mName); m != nil {
		v.WithMessages(m)
	}

	return v
}

// parse and collect rules from struct tags.
func (d *StructData) addRulesFromTag(v *Validation) {
	if d.TagName == "" {
		d.TagName = defaultTag
	}

	vt := d.value.Type()
	for i := 0; i < vt.NumField(); i++ {
		rule := vt.Field(i).Tag.Get(d.TagName)

		if rule != "" {
			v.StringRule(vt.Field(i).Name, rule)
		}
	}

	panic("implement me")
}

// collect custom field translates
func (d *StructData) collectTranslates(method string) map[string]string {
	if fv, has := d.FuncValue(method); has {
		ft := fv.Type()

		if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Map {
			return nil
		}

		vs := fv.Call([]reflect.Value{})
		ts, ok := vs[0].Interface().(map[string]string)
		if ok {
			return ts
		}
	}

	return nil
}

// collect custom error messages
func (d *StructData) collectMessages(method string) map[string]string {
	if fv, has := d.FuncValue(method); has {
		ft := fv.Type()

		if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Map {
			return nil
		}

		vs := fv.Call([]reflect.Value{})
		ts, ok := vs[0].Interface().(map[string]string)
		if ok {
			return ts
		}
	}

	return nil
}

/*************************************************************
 * Struct data operate
 *************************************************************/

// Get
func (d *StructData) Get(key string) (interface{}, bool) {
	panic("implement me")
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