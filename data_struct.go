package validate

import "reflect"

// StructData
type StructData struct {
	// Struct data from user setting
	Struct interface{}
	// from reflect Map or Struct
	value reflect.Value
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
	return NewValidation(d, scene...)
}
