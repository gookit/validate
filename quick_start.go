package validate

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

// New a Validation
func New(data interface{}, scene ...string) *Validation {
	switch td := data.(type) {
	case DataFace:
		return NewValidation(td, scene...)
	case GMap:
		return FromMap(td).Create(scene...)
	case map[string]interface{}:
		return FromMap(td).Create(scene...)
	case SValues:
		return FromURLValues(url.Values(td)).Create(scene...)
	case map[string][]string:
		return FromURLValues(data.(map[string][]string)).Create(scene...)
	}

	if reflect.TypeOf(data).Kind() == reflect.Struct {
		d, err := FromStruct(data)
		if err != nil {
			panic(err)
		}

		return d.Create(scene...)
	}

	panic("invalid input data")
}

// Map validation create
func Map(m map[string]interface{}, scene ...string) *Validation {
	return FromMap(m).New(scene...)
}

// Struct validation create
func Struct(s interface{}, scene ...string) *Validation {
	d, err := FromStruct(s)
	if err != nil {
		panicf(err.Error())
	}

	return d.New(scene...)
}

/*************************************************************
 * create data instance
 *************************************************************/

// ErrNoData define
var (
	// ErrNoData = errors.New("validate: no any data can be collected")
	ErrEmptyData = errors.New("validate: please input data use for validate")
)

// FromMap build data instance.
func FromMap(m map[string]interface{}) *MapData {
	data := &MapData{}
	if m != nil {
		val := reflect.ValueOf(m)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		data.Map = m
		data.value = val
	}

	return data
}

// FromStruct build data instance.
func FromStruct(s interface{}) (*StructData, error) {
	data := &StructData{TagName: defaultTag}
	val := reflect.ValueOf(s)

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type() == timeType {
		return nil, &ErrInvalidType{Type: reflect.TypeOf(s)}
	}

	data.source = s
	data.value = val

	return data, nil
}

// FromRequest collect data from request instance
func FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error) {
	// no body. like GET DELETE ....
	if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
		return FromURLValues(r.URL.Query()), nil
	}

	cType := r.Header.Get("Content-Type")

	// contains file uploaded form
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

		// collection uploaded files
		for key, files := range r.MultipartForm.File {
			if len(files) != 0 {
				data.AddFile(key, files[0])
			}
		}

		// add queries data
		data.AddValues(r.URL.Query())
		return data, nil
	}

	// basic POST form
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
	if strings.Contains(cType, "application/json") {
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
	data := &FormData{}

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

// FromJSON string build data instance.
func FromJSON(s string) (*MapData, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}

	return FromMap(m), nil
}

// FromJSONBytes string build data instance.
func FromJSONBytes(bs []byte) (*MapData, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal(bs, &m); err != nil {
		return nil, err
	}

	return FromMap(m), nil
}
