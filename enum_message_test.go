package validate_test

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

// TestNotInRule_message: the enum list is a []string argument, so it must not be formatted
// with %d (it used to render as "[%!d(string=go) %!d(string=php)]").
func TestNotInRule_message(t *testing.T) {
	v := validate.Map(map[string]any{"role": "user"})
	v.StringRule("role", "not_in:user,guest")
	assert.False(t, v.Validate())
	assert.Eq(t, "role value must not be in the given enum list [user guest]", v.Errors.One())
}
