package validate

import (
	"fmt"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/testutil/assert"
)

func TestBuiltinMessages(t *testing.T) {
	bm := BuiltinMessages()
	assert.NotContains(t, bm, "testMsg0")

	AddBuiltinMessages(map[string]string{
		"testMsg0": "message value",
	})

	bm = BuiltinMessages()

	assert.Contains(t, bm, "testMsg0")
	AddGlobalMessages(map[string]string{
		"testMsg1": "message value",
	})

	bm = BuiltinMessages()

	assert.Contains(t, bm, "testMsg1")
}

func TestErrorsBasic(t *testing.T) {
	es := Errors{}

	assert.True(t, es.Empty())
	assert.Equal(t, "", es.One())
	assert.Nil(t, es.ErrOrNil())

	es.Add("field", "required", "error msg0")
	assert.Len(t, es, 1)
	assert.Equal(t, "error msg0", es.One())
	assert.Equal(t, "error msg0", es.FieldOne("field"))
	assert.Equal(t, "field:\n required: error msg0", es.String())

	es.Add("field2", "min", "error msg2")
	assert.Contains(t, fmt.Sprintf("%v", es.All()), "field:map[required:error msg0]")
	assert.Contains(t, fmt.Sprintf("%v", es.All()), "field2:map[min:error msg2]")

	es.Add("field", "minLen", "error msg1")
	assert.Len(t, es.Field("field"), 2)

	jsonsStr, err := jsonutil.Pretty(es)
	assert.NoError(t, err)
	fmt.Println(jsonsStr)
	dump.V(es)
}

func TestTranslatorBasic(t *testing.T) {
	tr := NewTranslator()

	assert.True(t, tr.HasMessage("min"))
	assert.False(t, tr.HasMessage("not-exists"))
	assert.False(t, tr.HasLabel("FIELD1"))
	assert.False(t, tr.HasField("FIELD1"))

	tr.AddMessage("FIELD1.min", "{field} message1")
	assert.True(t, tr.HasMessage("FIELD1.min"))
	assert.Equal(t, "FIELD1 message1", tr.Message("min", "FIELD1"))

	tr.AddFieldMap(map[string]string{"FIELD1": "output_name"})
	assert.Equal(t, "output_name message1", tr.Message("min", "FIELD1"))

	tr.AddLabelMap(map[string]string{"FIELD1": "Show Name"})
	assert.Equal(t, "Show Name message1", tr.Message("min", "FIELD1"))

	tr.Reset()
}

func TestUseAliasMessageKey(t *testing.T) {
	is := assert.New(t)
	v := New(M{
		"name": "inhere",
	})
	v.StringRule("name", "required|string|minLen:7|maxLen:15")
	v.WithMessages(map[string]string{
		"name.minLen": "USERNAME min length is 7",
		// "minLen": "USERNAME min length is 7",
		// "name.minLength": "USERNAME min length is 7",
	})

	is.False(v.Validate())
	is.Equal("USERNAME min length is 7", v.Errors.One())
}

func TestMessageOnStruct(t *testing.T) {
	is := assert.New(t)

	s := &struct {
		Name     string `validate:"string"`
		BirthDay string `validate:"date" message:"出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v := Struct(s)

	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())

	s1 := &struct {
		Name     string `validate:"string"`
		BirthDay string `validate:"date" message:"date: 出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v = Struct(s1)
	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())

	s2 := &struct {
		Name     string `validate:"string"`
		BirthDay string `validate:"required|date" message:"date: 出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v = Struct(s2)
	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())

	s3 := &struct {
		Name     string `validate:"string"`
		BirthDay string `validate:"date|maxlen:20" message:"出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v = Struct(s3)
	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())

	s4 := &struct {
		Name     string `validate:"string" json:"name"`
		BirthDay string `validate:"date|maxlen:20" json:"birth_day" message:"出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v = Struct(s4)
	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())

	// Ensure message override with no specified field applies to all validation errors if SkipOnError=false is set in
	// global options
	Config(func(opt *GlobalOption) {
		opt.StopOnError = false
	})

	s5 := &struct {
		Name     string `validate:"string"`
		BirthDay string `validate:"max_len:1|date" message:"出生日期有误"`
	}{
		"tom",
		"ff",
	}

	v = Struct(s5)

	is.False(v.Validate())
	is.Contains(v.Errors.String(), "BirthDay")
	is.Contains(v.Errors.String(), "max_len: 出生日期有误")
	is.Contains(v.Errors.String(), "date: 出生日期有误")

	// Restore original global options
	Config(func(opt *GlobalOption) {
		opt.StopOnError = true
	})
}

// with field tag: json
func TestMessageOnStruct_withFieldTag(t *testing.T) {
	is := assert.New(t)
	s1 := &struct {
		Name     string `validate:"string" json:"name"`
		BirthDay string `validate:"date|maxlen:20" json:"birth_day" message:"出生日期有误"`
	}{
		"tom",
		"invalid",
	}

	v := Struct(s1)
	is.False(v.Validate())
	is.Equal("出生日期有误", v.Errors.One())
}

func TestMessageOnStruct_withNested(t *testing.T) {
	is := assert.New(t)
	type subSt struct {
		Tags []string `json:"tags"`
		Key1 string
	}

	s1 := &struct {
		Name     string `validate:"string" json:"name"`
		BirthDay string `validate:"date|maxlen:20" json:"birth_day" label:"birth day" message:"{field} 出生日期有误"`
		SubSt    subSt
	}{
		"tom",
		"invalid",
		subSt{
			Key1: "abc",
		},
	}

	v := Struct(s1)
	tr := v.Trans()
	dump.V(tr.FieldMap(), tr.LabelMap())
	is.Contains(tr.FieldMap(), "BirthDay")
	is.Contains(tr.FieldMap(), "SubSt.Tags")
	is.Equal("birth_day", tr.FieldName("BirthDay"))
	is.Equal("tags", tr.FieldName("SubSt.Tags"))

	is.Contains(tr.LabelMap(), "BirthDay")
	is.Equal("birth day", tr.LabelName("BirthDay"))
	is.Equal("tags", tr.LabelName("SubSt.Tags"))

	is.False(v.Validate())
	dump.V(v.Errors)
	is.Equal("birth day 出生日期有误", v.Errors.One())
}
