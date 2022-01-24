// Package validate is a generic go data validate, filtering library.
//
// Source code and other details for the project are available at GitHub:
//
// 	https://github.com/gookit/validate
//
package validate

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

// M is short name for map[string]interface{}
type M map[string]interface{}

// MS is short name for map[string]string
type MS map[string]string

// SValues simple values
type SValues map[string][]string

// One get one item's value string
func (ms MS) One() string {
	for _, msg := range ms {
		return msg
	}
	return ""
}

// String convert map[string]string to string
func (ms MS) String() string {
	if len(ms) == 0 {
		return ""
	}

	var ss []string
	for name, msg := range ms {
		ss = append(ss, " "+name+": "+msg)
	}

	return strings.Join(ss, "\n")
}

// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag name in the struct tags.
	FilterTag string
	// LabelTag name in the struct tags.
	LabelTag string
	// ValidateTag in the struct tags.
	ValidateTag string
	// FieldTag name in the struct tags. for define field translate. default: json
	FieldTag string
	// FieldNameTag display name in the struct tags . for define field translate. default: label
	FieldNameTag string
	// MessageTag define error message for the field.
	MessageTag string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
	// UpdateSource Whether to update source field value, useful for struct validate
	UpdateSource bool
	// CheckDefault Whether to validate the default value set by the user
	CheckDefault bool
	// CheckZero Whether validate the default zero value. (intX,uintX: 0, string: "")
	CheckZero bool
	// OutGoFmt Whether to output the native fields of go.
	OutGoFmt string
}

// global options
var gOpt = newGlobalOption()

// Config global options
func Config(fn func(opt *GlobalOption)) {
	fn(gOpt)
}

// ResetOption reset global option
func ResetOption() {
	gOpt = newGlobalOption()
}

// Option get global options
func Option() GlobalOption {
	return *gOpt
}

func newGlobalOption() *GlobalOption {
	return &GlobalOption{
		StopOnError: true,
		SkipOnEmpty: true,
		// tag name in struct tags
		FieldTag: fieldTag,
		// label tag - display name in struct tags
		LabelTag: labelTag,
		// tag name in struct tags
		FilterTag:  filterTag,
		MessageTag: messageTag,
		// tag name in struct tags
		ValidateTag: validateTag,
		OutGoFmt:    outgofmt,
	}
}

func newValidation(data DataFace) *Validation {
	v := &Validation{
		Errors: make(Errors),
		// add data source on usage
		data: data,
		// create message translator
		trans: StdTranslator,
		// validated data
		safeData: make(map[string]interface{}),
		// validator names
		validators: make(map[string]int8),
		// filtered data
		filteredData: make(map[string]interface{}),
		// default config
		StopOnError: gOpt.StopOnError,
		SkipOnEmpty: gOpt.SkipOnEmpty,
	}

	// init build in context validator
	v.validatorValues = map[string]reflect.Value{
		"required":           reflect.ValueOf(v.Required),
		"requiredIf":         reflect.ValueOf(v.RequiredIf),
		"requiredUnless":     reflect.ValueOf(v.RequiredUnless),
		"requiredWith":       reflect.ValueOf(v.RequiredWith),
		"requiredWithAll":    reflect.ValueOf(v.RequiredWithAll),
		"requiredWithout":    reflect.ValueOf(v.RequiredWithout),
		"requiredWithoutAll": reflect.ValueOf(v.RequiredWithoutAll),
		// field compare
		"eqField":  reflect.ValueOf(v.EqField),
		"neField":  reflect.ValueOf(v.NeField),
		"gtField":  reflect.ValueOf(v.GtField),
		"gteField": reflect.ValueOf(v.GteField),
		"ltField":  reflect.ValueOf(v.LtField),
		"lteField": reflect.ValueOf(v.LteField),
		// file upload check
		"isFile":      reflect.ValueOf(v.IsFormFile),
		"isImage":     reflect.ValueOf(v.IsFormImage),
		"inMimeTypes": reflect.ValueOf(v.InMimeTypes),
	}

	v.validatorMetas = make(map[string]*funcMeta)

	// collect meta info
	for n, fv := range v.validatorValues {
		v.validators[n] = 1 // built in
		v.validatorMetas[n] = newFuncMeta(n, true, fv)
	}

	// v.pool = &sync.Pool{
	// 	New: func() interface{} {
	// 		return &Validation{
	// 			v: v,
	// 		}
	// 	},
	// }

	return v
}

/*************************************************************
 * quick create Validation
 *************************************************************/

// New create a Validation instance
// data support:
// - DataFace
// - M/map[string]interface{}
// - SValues/url.Values/map[string][]string
// - struct ptr
func New(data interface{}, scene ...string) *Validation {
	switch td := data.(type) {
	case DataFace:
		return NewValidation(td, scene...)
	case M:
		return FromMap(td).Create().SetScene(scene...)
	case map[string]interface{}:
		return FromMap(td).Create().SetScene(scene...)
	case SValues:
		return FromURLValues(url.Values(td)).Create().SetScene(scene...)
	case url.Values:
		return FromURLValues(td).Create().SetScene(scene...)
	case map[string][]string:
		return FromURLValues(td).Create().SetScene(scene...)
	}

	return Struct(data, scene...)
}

// NewWithOptions new Validation with options
// func NewWithOptions(data interface{}, fn func(opt *GlobalOption)) *Validation {
// 	fn(gOpt)
// 	return New(data)
// }

// Map validation create
func Map(m map[string]interface{}, scene ...string) *Validation {
	return FromMap(m).Create().SetScene(scene...)
}

// JSON create validation from JSON string.
func JSON(s string, scene ...string) *Validation {
	return mustNewValidation(FromJSON(s)).SetScene(scene...)
}

// Struct validation create
func Struct(s interface{}, scene ...string) *Validation {
	return mustNewValidation(FromStruct(s)).SetScene(scene...)
}

// Request validation create
func Request(r *http.Request) *Validation {
	return mustNewValidation(FromRequest(r))
}

func mustNewValidation(d DataFace, err error) *Validation {
	if d == nil {
		if err != nil {
			return NewValidation(d).WithError(err)
		}
		return NewValidation(d)
	}

	return d.Create(err)
}

/*************************************************************
 * create data-source instance
 *************************************************************/

// FromMap build data instance.
func FromMap(m map[string]interface{}) *MapData {
	data := &MapData{}
	if m != nil {
		data.Map = m
		data.value = reflect.ValueOf(m)
	}
	return data
}

// FromJSON string build data instance.
func FromJSON(s string) (*MapData, error) {
	return FromJSONBytes([]byte(s))
}

// FromJSONBytes string build data instance.
func FromJSONBytes(bs []byte) (*MapData, error) {
	mp := map[string]interface{}{}
	if err := json.Unmarshal(bs, &mp); err != nil {
		return nil, err
	}

	data := &MapData{
		Map:   mp,
		value: reflect.ValueOf(mp),
		// save JSON bytes
		bodyJSON: bs,
	}

	return data, nil
}

// FromStruct create a Data from struct
func FromStruct(s interface{}) (*StructData, error) {
	data := &StructData{
		ValidateTag: gOpt.ValidateTag,
		// init map
		fieldNames:  make(map[string]int8),
		fieldValues: make(map[string]reflect.Value),
	}

	if s == nil {
		return data, ErrInvalidData
	}

	val := reflect.ValueOf(s)
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

var jsonContent = regexp.MustCompile(`(?i)application/((\w|\.|-)+\+)?json(-seq)?`)

// FromRequest collect data from request instance
func FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error) {
	// no body. like GET DELETE ....
	if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
		return FromURLValues(r.URL.Query()), nil
	}

	cType := r.Header.Get("Content-Type")

	// contains file uploaded form
	// strings.HasPrefix(mediaType, "multipart/")
	if strings.Contains(cType, "multipart/form-data") {
		maxMemory := defaultMaxMemory
		if len(maxMemoryLimit) > 0 {
			maxMemory = maxMemoryLimit[0]
		}

		if err := r.ParseMultipartForm(maxMemory); err != nil {
			return nil, err
		}

		// collect from values
		data := FromURLValues(r.MultipartForm.Value)
		// collect uploaded files
		data.AddFiles(r.MultipartForm.File)
		// add queries data
		data.AddValues(r.URL.Query())
		return data, nil
	}

	// basic POST form. content type: application/x-www-form-urlencoded
	if strings.Contains(cType, "form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return nil, err
		}

		data := FromURLValues(r.PostForm)
		// add queries data
		data.AddValues(r.URL.Query())
		return data, nil
	}

	// JSON body request
	if jsonContent.MatchString(cType) {
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}

		return FromJSONBytes(bs)
	}

	return nil, ErrEmptyData
}

// FromURLValues build data instance.
func FromURLValues(values url.Values) *FormData {
	data := newFormData()
	for key, vals := range values {
		for _, val := range vals {
			data.Add(key, val)
		}
	}

	return data
}

// FromQuery build data instance.
// Usage:
// 	validate.FromQuery(r.URL.Query()).Create()
func FromQuery(values url.Values) *FormData {
	return FromURLValues(values)
}
