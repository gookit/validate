package validate

import (
	"fmt"
	"github.com/gookit/filter"
	"io/ioutil"
	"mime/multipart"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

/*************************************************************
 * Map Data
 *************************************************************/

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

	return filter.GetByPath(field, d.Map)
}

// Create a Validation from data
func (d *MapData) Create(err ...error) *Validation {
	return d.Validation(err...)
}

// Validation create from data
func (d *MapData) Validation(err ...error) *Validation {
	if len(err) > 0 {
		return NewValidation(d).WithError(err[0])
	}

	return NewValidation(d)
}

/*************************************************************
 * Struct Data
 *************************************************************/

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

// Create a Validation from the StructData
func (d *StructData) Create(err ...error) *Validation {
	return d.Validation(err...)
}

// Validation create from the StructData
func (d *StructData) Validation(err ...error) *Validation {
	v := NewValidation(d)

	if len(err) > 0 && err[0] != nil {
		return v.WithError(err[0])
	}

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
	fnName := filter.UpperFirst(name)
	fv := d.value.MethodByName(fnName)

	return fv, fv.IsValid()
}

/*************************************************************
 * Form Data
 *************************************************************/

// FormData obtained from the request body or url query parameters or user custom setting.
type FormData struct {
	// Form holds any basic key-value string data
	// This includes all fields from urlencoded form,
	// and the form fields only (not files) from a multipart form
	Form url.Values
	// Files holds files from a multipart form only.
	// For any other type of request, it will always
	// be empty. Files only supports one file per key,
	// since this is by far the most common use. If you
	// need to have more than one file per key, parse the
	// files manually using r.MultipartForm.File.
	Files map[string]*multipart.FileHeader
	// jsonBodies holds the original body of the request.
	// Only available for json requests.
	jsonBodies []byte
}

func newFormData() *FormData {
	return &FormData{
		Form:  make(map[string][]string),
		Files: make(map[string]*multipart.FileHeader),
	}
}

/*************************************************************
 * Form data operate
 *************************************************************/

// Create a Validation from data
func (d *FormData) Create(err ...error) *Validation {
	return d.Validation(err...)
}

// Validation create from data
func (d *FormData) Validation(err ...error) *Validation {
	if len(err) > 0 && err[0] != nil {
		return NewValidation(d).WithError(err[0])
	}

	return NewValidation(d)
}

// Add adds the value to key. It appends to any existing values associated with key.
func (d *FormData) Add(key string, value string) {
	d.Form.Add(key, value)
}

// AddValues to Data.Form
func (d *FormData) AddValues(values url.Values) {
	for key, vals := range values {
		for _, val := range vals {
			d.Form.Add(key, val)
		}
	}
}

// AddFiles adds the multipart form files to data
func (d *FormData) AddFiles(files map[string][]*multipart.FileHeader) {
	for key, files := range files {
		if len(files) != 0 {
			d.AddFile(key, files[0])
		}
	}
}

// AddFile adds the multipart form file to data with the given key.
func (d *FormData) AddFile(key string, file *multipart.FileHeader) {
	d.Files[key] = file
}

// Del deletes the values associated with key.
func (d *FormData) Del(key string) {
	d.Form.Del(key)
}

// DelFile deletes the file associated with key (if any).
// If there is no file associated with key, it does nothing.
func (d *FormData) DelFile(key string) {
	delete(d.Files, key)
}

// Encode encodes the values into “URL encoded” form ("bar=baz&foo=quux") sorted by key.
// Any files in d will be ignored because there is no direct way to convert a file to a
// URL encoded value.
func (d *FormData) Encode() string {
	return d.Form.Encode()
}

// Set sets the key to value. It replaces any existing values.
func (d *FormData) Set(field string, val interface{}) error {
	switch val.(type) {
	case string:
		d.Form.Set(field, val.(string))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		d.Form.Set(field, fmt.Sprint(val))
	}

	return nil
}

// Get value by key
func (d FormData) Get(key string) (interface{}, bool) {
	return d.Form.Get(key), true
}

// Trimmed gets the first value by key, and apply strings.TrimSpace
func (d FormData) Trimmed(key string) string {
	return strings.TrimSpace(d.Form.Get(key))
}

// GetFile returns the multipart form file associated with key, if any, as a *multipart.FileHeader.
// If there is no file associated with key, it returns nil. If you just want the body of the
// file, use GetFileBytes.
func (d FormData) GetFile(key string) *multipart.FileHeader {
	return d.Files[key]
}

// Has key in the Data
func (d FormData) Has(key string) bool {
	if vs, ok := d.Form[key]; ok && len(vs) > 0 {
		return true
	}

	if _, ok := d.Files[key]; ok {
		return true
	}

	return false
}

// HasField returns true iff data.Form[key] exists. When parsing a request body, the key
// is considered to be in existence if it was provided in the request body, even if its value
// is empty.
func (d FormData) HasField(key string) bool {
	_, found := d.Form[key]
	return found
}

// HasFile returns true iff data.Files[key] exists. When parsing a request body, the key
// is considered to be in existence if it was provided in the request body, even if the file
// is empty.
func (d FormData) HasFile(key string) bool {
	_, found := d.Files[key]
	return found
}

// Int returns the first element in data[key] converted to an int.
func (d FormData) Int(key string) int {
	if !d.HasField(key) || len(d.Form[key]) == 0 {
		return 0
	}

	str, _ := d.Get(key)
	if result, err := strconv.Atoi(str.(string)); err != nil {
		panic(err)
	} else {
		return result
	}
}

// MustInt64 returns the first element in data[key] converted to an int64.
func (d FormData) MustInt64(key string) int64 {
	if !d.HasField(key) || len(d.Form[key]) == 0 {
		return 0
	}

	val, _ := strconv.ParseInt(d.Trimmed(key), 10, 0)
	return val
}

// Float returns the first element in data[key] converted to a float.
func (d FormData) Float(key string) float64 {
	if !d.HasField(key) || len(d.Form[key]) == 0 {
		return 0
	}

	result, err := strconv.ParseFloat(d.String(key), 64)
	if err != nil {
		panic(err)
	}

	return result
}

// Bool returns the first element in data[key] converted to a bool.
func (d FormData) Bool(key string) bool {
	if !d.HasField(key) || len(d.Form[key]) == 0 {
		return false
	}

	if result, err := strconv.ParseBool(d.String(key)); err != nil {
		panic(err)
	} else {
		return result
	}
}

// String value get by key
func (d FormData) String(key string) string {
	return d.Form.Get(key)
}

// FileBytes returns the body of the file associated with key. If there is no
// file associated with key, it returns nil (not an error). It may return an error if
// there was a problem reading the file. If you need to know whether or not the file
// exists (i.e. whether it was provided in the request), use the FileExists method.
func (d FormData) FileBytes(key string) ([]byte, error) {
	fileHeader, found := d.Files[key]
	if !found {
		return nil, nil
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(file)
}

// BindJSON binds v to the json data in the request body. It calls json.Unmarshal and
// sets the value of v.
func (d FormData) BindJSON(v interface{}) error {
	if len(d.jsonBodies) == 0 {
		return nil
	}

	return Unmarshal(d.jsonBodies, v)
}

// MapTo the v
func (d FormData) MapTo(v interface{}) error {
	return d.BindJSON(v)
}
