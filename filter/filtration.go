// Package filter provide filter, sanitize, convert data
package filter

import "github.com/gookit/validate/helper"

// Filtration definition. Sanitization Sanitizing Sanitize
type Filtration struct {
	// filtered data
	filtered map[string]interface{}
}

// Get value by key
func (f *Filtration) Get(key string) (interface{}, bool) {
	return helper.GetByPath(key, f.filtered)
}

// Set value by key
func (f *Filtration) Set(field string, val interface{}) error {
	panic("implement me")
}
