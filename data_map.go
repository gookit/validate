package validate

import (
	"reflect"
	"strings"
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

// Set
func (d *MapData) Set(field string, val interface{}) error {
	d.Map[field] = val
	return nil
}

// Get field value
func (d *MapData) Get(field string) (interface{}, bool) {
	// if fv, ok := d.fields[field]; ok {
	// 	return fv, true
	// }

	if val, ok := d.Map[field]; ok {
		// d.fields[field] = reflect.ValueOf(val)
		return val, true
	}

	// has sub key? eg. "top.sub"
	if strings.ContainsRune(field, '.') {
		return d.GetByPath(field)
	}

	return nil, false
}

// GetByPath get value by key path. eg "top.sub"
func (d *MapData) GetByPath(key string) (val interface{}, ok bool) {
	// has sub key? eg. "top.sub"
	if !strings.ContainsRune(key, '.') {
		return d.Get(key)
	}

	keys := strings.Split(key, ".")
	topK := keys[0]

	// find top item data based on top key
	var item interface{}
	if item, ok = d.Map[topK]; !ok {
		return
	}

	for _, k := range keys[1:] {
		switch tData := item.(type) {
		case map[string]string: // is simple map
			item, ok = tData[k]
			if !ok {
				return
			}
		case map[string]interface{}: // is map(decode from toml/json)
			if item, ok = tData[k]; !ok {
				return
			}
		case map[interface{}]interface{}: // is map(decode from yaml)
			if item, ok = tData[k]; !ok {
				return
			}
		default: // error
			ok = false
			return
		}
	}

	return item, true
}

// Create a Validation from data
func (d *MapData) Create(scene ...string) *Validation {
	return d.Validation(scene...)
}

// Validation create from data
func (d *MapData) Validation(scene ...string) *Validation {
	return NewValidation(d, scene...)
}
