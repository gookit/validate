package validate

import (
	"reflect"
)

// MapData definition
type MapData struct {
	// Map form unmarshal JSON string, or user setting
	Map map[string]interface{}
	// from reflect Map or Struct
	src reflect.Value
	// bodyJSON from the original body of the request. only available for json http request.
	bodyJSON []byte
	// map field reflect.Value caches
	fields map[string]reflect.Value
}

/*************************************************************
 * Map data operate
 *************************************************************/

// Set value by key
func (d *MapData) Set(field string, val interface{}) error {
	d.Map[field] = val
	return nil
}

// Get value by key
func (d *MapData) Get(field string) (interface{}, bool) {
	// if fv, ok := d.fields[field]; ok {
	// 	return fv, true
	// }

	return GetByPath(field, d.Map)
}

// Create a Validation from data
func (d *MapData) Create(scene ...string) *Validation {
	return d.Validation(scene...)
}

// Validation create from data
func (d *MapData) Validation(scene ...string) *Validation {
	return NewValidation(d, scene...)
}
