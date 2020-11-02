package validate

/*
// TODO: idea for optimize ...
// - use pool for Validation
// - cache struct reflect value

// Usage:
// // use a single instance of Validate, it caches struct info
// vf := validate.NewFactory()
// v := vf.New(data)
// v.Validate()

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
*/
