package validate

import (
	"net/url"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// TestCrossFieldRule_lowercaseFieldName: a rule may name an acronym-style field in lower
// case ("md5" for MD5, "url" for URL), the lookup matches the field name ignoring case.
// https://github.com/gookit/validate/v2/issues/347
func TestCrossFieldRule_lowercaseFieldName(t *testing.T) {
	type form struct {
		MD5  string `form:"md5"  validate:"required_without_all:url,file"`
		URL  string `form:"url"  validate:"required_without_all:md5,file"`
		File string `form:"file" validate:"required_without_all:md5,url"`
	}

	tests := []struct {
		name string
		form *form
		ok   bool
	}{
		{"only MD5", &form{MD5: "x"}, true},
		{"none", &form{}, false},
		{"URL+File", &form{URL: "u", File: "f"}, true},
		{"all three", &form{MD5: "m", URL: "u", File: "f"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, tt.ok, Struct(tt.form).Validate())
		})
	}

	// the same names work in the field getters and in the upper-case form
	f := &form{MD5: "m", URL: "u"}
	d, err := FromStruct(f)
	assert.NoErr(t, err)
	for _, path := range []string{"MD5", "md5", "Md5", "url", "URL"} {
		val, exist := d.Get(path)
		assert.True(t, exist, path)
		assert.NotEmpty(t, val, path)
	}

	// a really unknown field is still not found
	_, exist := d.Get("tpyo")
	assert.False(t, exist)
}

// TestEnumRule_mapValue: the enum rules report an unsupported map value instead of the
// "must be in the enum" message, which reads like the value is simply not in the list.
func TestEnumRule_mapValue(t *testing.T) {
	type form struct {
		Tags map[string]string `validate:"in:a,b"`
	}

	v := Struct(&form{Tags: map[string]string{"k": "a"}})
	assert.False(t, v.Validate())
	assert.StrContains(t, v.Errors.FieldOne("Tags"), "does not support a map value")

	// a slice of strings is supported as before (#266)
	type sliceForm struct {
		Tags []string `validate:"in:a,b"`
	}
	assert.True(t, Struct(&sliceForm{Tags: []string{"a", "b"}}).Validate())
	assert.False(t, Struct(&sliceForm{Tags: []string{"a", "x"}}).Validate())
}

// TestStructRule_promotedFieldVsUnexportedField: a rule on a promoted field must not
// resolve to an unexported field of the outer struct whose name differs only in case.
// https://github.com/gookit/validate/v2/issues/360
func TestStructRule_promotedFieldVsUnexportedField(t *testing.T) {
	type Settings struct {
		WebURL string
	}
	type Config struct {
		Settings
		webURL *url.URL // parsed copy of WebURL
	}

	c := &Config{Settings: Settings{WebURL: "https://example.com"}}
	c.webURL, _ = url.Parse(c.WebURL)

	v := Struct(c)
	v.AddRule("WebURL", "required")
	assert.True(t, v.Validate(), v.Errors.One())

	// the case-insensitive fallback still resolves the lower-case name to the exported
	// WebURL, it just no longer picks the unexported field
	val, exist := v.Get("webURL")
	assert.True(t, exist)
	assert.Eq(t, "https://example.com", val)
}
