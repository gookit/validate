package validate_test

import (
	"encoding/json"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

func TestRequiredUnlessMissingField(t *testing.T) {
	for _, tt := range []struct {
		name string
		data map[string]any
		want bool
	}{
		{"missing dependency and value", map[string]any{}, false},
		{"missing dependency with empty value", map[string]any{"name": ""}, false},
		{"missing dependency with value", map[string]any{"name": "Ada"}, true},
		{"matching dependency without value", map[string]any{"mode": "guest"}, true},
		{"matching dependency with empty value", map[string]any{"mode": "guest", "name": ""}, true},
		{"other dependency without value", map[string]any{"mode": "member"}, false},
		{"other dependency with value", map[string]any{"mode": "member", "name": "Ada"}, true},
		{"missing dependency with zero number", map[string]any{"name": 0}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := validate.Map(tt.data)
			v.StringRule("name", "required_unless:mode,guest")
			assert.Eq(t, tt.want, v.Validate())
		})
	}
}

func TestRequiredUnlessJSONMissingField(t *testing.T) {
	for _, source := range []string{`{}`, `{"name":""}`} {
		var data map[string]any
		assert.NoErr(t, json.Unmarshal([]byte(source), &data))
		v := validate.Map(data)
		v.StringRule("name", "required_unless:mode,guest")
		assert.False(t, v.Validate())
		assert.NotEmpty(t, v.Errors.Field("name"))
	}
}
