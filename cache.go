package validate

import "sync"

// global factory for create Validation
var gf = newFactory()

// factory for create Validation instances
type factory struct {
	// pool for Validation instances
	pool sync.Pool
	// m atomic.Value // map[reflect.Type]*cStruct
}

func newFactory() *factory {
	f := &factory{}
	f.pool.New = func() interface{} {
		return newValidation(nil)
	}

	return f
}

func (f *factory) get() *Validation {
	v := f.pool.Get().(*Validation)
	// TODO clear something
	return v
}

func (f *factory) put(v *Validation) {
	f.pool.Put(v)
}
