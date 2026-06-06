// Package validate is a generic go data validate, filtering library.
//
// Source code and other details for the project are available at GitHub:
//
//	https://github.com/gookit/validate/v2
package validate

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/gookit/goutil/reflects"
)

// M is a short name for map[string]any
type M map[string]any

// MS is a short name for map[string]string
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

	ss := make([]string, 0, len(ms))
	for name, msg := range ms {
		ss = append(ss, " "+name+": "+msg)
	}

	return strings.Join(ss, "\n")
}

func (ms MS) OrderedRange(fn func(key, value string)) {
	if len(ms) == 0 || fn == nil {
		return
	}

	keys := make([]string, 0, len(ms))
	for k := range ms {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fn(k, ms[k])
	}
}

// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag name in the struct tags.
	//
	// default: filter
	FilterTag string
	// ValidateTag in the struct tags.
	//
	// default: validate
	ValidateTag string
	// FieldTag the output field name in the struct tags.
	// it as placeholder on error message.
	//
	// default: json
	FieldTag string
	// LabelTag the display name in the struct tags.
	// use for define field translate name on error.
	//
	// default: label
	LabelTag string
	// MessageTag define error message for the field.
	//
	// default: message
	MessageTag string
	// DefaultTag define default value for the field.
	//
	// tag: default TODO
	DefaultTag string
	// StopOnError If true: An error occurs, it will cease to continue to verify. default is True.
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty. default is True.
	SkipOnEmpty bool
	// UpdateSource Whether to update source field value, useful for struct validate
	UpdateSource bool
	// CheckDefault Whether to validate the default value set by the user
	CheckDefault bool
	// ErrShowValue Whether to append the original value that triggered the error
	// to the error message. opt-in, default is false (keeps error messages
	// byte-for-byte unchanged). When true, the failing value is appended in the
	// form " (value: <val>)". see GitHub issue #184.
	ErrShowValue bool
	// CheckZero whether to validate the zero value. (intX,uintX: 0, string: "")
	//
	// Deprecated: this flag is a no-op — it was declared but never wired into the
	// validation logic. To validate empty/zero values instead of skipping them,
	// use per-rule Rule.SetSkipEmpty(false), set SkipOnEmpty=false on the
	// Validation/global option, or mark the field as required. It will be removed
	// in a future release.
	CheckZero bool
	// ErrKeyFmt config. TODO
	//
	// allow:
	// - 0 use struct field name as key. (for compatible)
	// - 1 use FieldTag defined name as key.
	ErrKeyFmt int8
	// CheckSubOnParentMarked controls sub-struct (struct / *struct / slice-of-struct /
	// map-of-struct) cascade validation.
	//
	// 默认 true: 仅当父字段带有 `validate` tag(值可为空, 如 `validate:""`)时才级联验证
	// 其子结构体; 完全没有 `validate` tag 的字段不会下探。设为 false 则无条件级联(v1 行为)。
	CheckSubOnParentMarked bool
	// ValidatePrivateFields Whether to validate private fields or not, especially when inheriting other other structs.
	//
	//  type foo struct {
	//	  Field int `json:"field" validate:"required"`
	//  }
	//  type bar struct {
	//    foo // <-- validate this field
	//    Field2 int `json:"field2" validate:"required"`
	//  }
	//
	// default: false
	ValidatePrivateFields bool

	// RestoreRequestBody Whether to restore the request body after reading it.
	// default: false
	RestoreRequestBody bool
}

// global options
var gOpt = newGlobalOption()

// Config global options
func Config(fn func(opt *GlobalOption)) {
	fn(gOpt)
	// bump tag-config version so any type meta cached under the previous tag
	// names is invalidated (see cache.go typeKey).
	atomic.AddUint32(&tagVer, 1)
}

// ResetOption reset global option
func ResetOption() {
	*gOpt = *newGlobalOption()
	// invalidate type meta cache built under the previous tag config.
	atomic.AddUint32(&tagVer, 1)
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
		// 默认仅在父字段带有 validate tag 时才级联验证子结构体 (Java @Valid 风格的简化版)
		CheckSubOnParentMarked: true,
	}
}

func newValidation(data DataFace) *Validation {
	v := newEmpty()
	v.data = data
	return v
}

// ctxValidatorBuilders is the package-level static binder table for the 16
// build-in context validators. Each entry returns the bound-method's
// reflect.Value for a specific Validation instance.
//
// perf (P5b): previously newEmpty() eagerly built all 16 funcMeta (16 bound
// methods + 16 newFuncMeta + 2 map fills) for EVERY instance, but the common
// struct/map validation never needs any per-instance context meta:
//   - "required" (no ".*") takes the fast path in valueValidate and returns
//     before looking up a funcMeta;
//   - minLen/email/int/min/max ... are global shared free-function validators.
//
// So they are now built lazily on first lookup (see validatorMeta). The table
// itself is built once; each value is a tiny closure with no per-instance cost.
var ctxValidatorBuilders = map[string]func(v *Validation) reflect.Value{
	"required":           func(v *Validation) reflect.Value { return reflect.ValueOf(v.Required) },
	"requiredIf":         func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredIf) },
	"requiredUnless":     func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredUnless) },
	"requiredWith":       func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredWith) },
	"requiredWithAll":    func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredWithAll) },
	"requiredWithout":    func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredWithout) },
	"requiredWithoutAll": func(v *Validation) reflect.Value { return reflect.ValueOf(v.RequiredWithoutAll) },
	// field compare
	"eqField":  func(v *Validation) reflect.Value { return reflect.ValueOf(v.EqField) },
	"neField":  func(v *Validation) reflect.Value { return reflect.ValueOf(v.NeField) },
	"gtField":  func(v *Validation) reflect.Value { return reflect.ValueOf(v.GtField) },
	"gteField": func(v *Validation) reflect.Value { return reflect.ValueOf(v.GteField) },
	"ltField":  func(v *Validation) reflect.Value { return reflect.ValueOf(v.LtField) },
	"lteField": func(v *Validation) reflect.Value { return reflect.ValueOf(v.LteField) },
	// file upload check. NOTE: method names differ from the validator names.
	"isFile":      func(v *Validation) reflect.Value { return reflect.ValueOf(v.IsFormFile) },
	"isImage":     func(v *Validation) reflect.Value { return reflect.ValueOf(v.IsFormImage) },
	"inMimeTypes": func(v *Validation) reflect.Value { return reflect.ValueOf(v.InMimeTypes) },
}

func init() {
	// rule-level logical OR (#292). registered in init() rather than the map
	// literal to avoid a static initialization cycle: RuleOneOf -> validatorMeta
	// -> ctxValidatorBuilders. the bound method shape mirrors Enum(val, list):
	// numIn=2 so checkArgNum(1 list arg + addNum=1 = 2) passes, same as enum.
	ctxValidatorBuilders["rule_one_of"] = func(v *Validation) reflect.Value {
		return reflect.ValueOf(v.RuleOneOf)
	}
}

func newEmpty() *Validation {
	// perf (Step 2): all per-instance maps below are now LAZILY allocated on
	// first write (see the ensure*() guards) instead of eagerly here. Most common
	// validations leave several of them empty (no error, no optional field, no
	// custom validator, no filtered data), so eager make() wasted ~4-6 allocs per
	// instance. Reads of a nil map are safe (return zero value); only writes need
	// the guard. trans's labelMap/fieldMap are likewise lazy (see messages.go).
	v := &Validation{
		// create message translator
		// trans: StdTranslator,
		trans: NewTranslator(),
		// default config
		StopOnError:  gOpt.StopOnError,
		SkipOnEmpty:  gOpt.SkipOnEmpty,
		ErrShowValue: gOpt.ErrShowValue,
	}

	return v
}

/*************************************************************
 * quick create Validation
 *************************************************************/

// New create a Validation instance
//
// data type support:
//   - DataFace
//   - M/map[string]any
//   - SValues/url.Values/map[string][]string
//   - struct ptr
func New(data any, scene ...string) *Validation {
	switch td := data.(type) {
	case DataFace:
		return NewValidation(td, scene...)
	case M:
		return FromMap(td).Create().SetScene(scene...)
	case map[string]any:
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

// NewWithOptions new Validation with options TODO
// func NewWithOptions(data any, fn func(opt *GlobalOption)) *Validation {
// 	fn(gOpt)
// 	return New(data)
// }

// Map validation create
func Map(m map[string]any, scene ...string) *Validation {
	return FromMap(m).Create().SetScene(scene...)
}

// MapWithRules validation create and with rules
// func MapWithRules(m map[string]any, rules MS) *Validation {
// 	return FromMap(m).Create().StringRules(rules)
// }

// JSON create validation from JSON string.
func JSON(s string, scene ...string) *Validation {
	return mustNewValidation(FromJSON(s)).SetScene(scene...)
}

// Struct validation create
func Struct(s any, scene ...string) *Validation {
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
func FromMap(m map[string]any) *MapData {
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
	mp := map[string]any{}
	if err := Unmarshal(bs, &mp); err != nil {
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
func FromStruct(s any) (*StructData, error) {
	data := &StructData{
		ValidateTag: gOpt.ValidateTag,
		// init map
		fieldNames: make(map[string]int8),
	}

	if s == nil {
		return data, ErrInvalidData
	}

	val := reflects.Elem(reflect.ValueOf(s))
	typ := val.Type()

	if val.Kind() != reflect.Struct || typ == timeType {
		return data, ErrInvalidData
	}

	data.src = s
	data.value = val
	data.valueTyp = typ
	// build/fetch cached type-level metadata (field index, tags, Implements...).
	data.meta = getTypeMeta(typ)

	return data, nil
}

var jsonContent = regexp.MustCompile(`(?i)application/((\w|\.|-)+\+)?json(-seq)?`)

// FromRequest collect data from request instance
func FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error) {
	// nobody. like GET DELETE ....
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
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
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		// restore request body
		if gOpt.RestoreRequestBody {
			r.Body = io.NopCloser(bytes.NewBuffer(bs))
		}
		return FromJSONBytes(bs)
	}

	return nil, ErrEmptyData
}

// FromURLValues build data instance.
//
// Bracket-style nested keys are normalized to dot paths so nested form fields
// can be validated and bound, eg "address[street]" -> "address.street" (#324).
func FromURLValues(values url.Values) *FormData {
	data := newFormData()
	for key, vals := range values {
		key = normalizeFormKey(key)
		for _, val := range vals {
			data.Add(key, val)
		}
	}

	return data
}

// FromQuery build data instance.
//
// Usage:
//
//	validate.FromQuery(r.URL.Query()).Create()
func FromQuery(values url.Values) *FormData {
	return FromURLValues(values)
}
