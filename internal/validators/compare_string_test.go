package validators

import (
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// TestCompareStringValue pins the string semantics of the numeric comparators: a
// string field is compared with the rule argument lexicographically, NOT by length.
// Use MinLength/MaxLength (min_len/max_len) for a length check.
func TestCompareStringValue(t *testing.T) {
	// "ab" >= "3" lexicographically, so min:3 passes even though len("ab") == 2
	assert.True(t, Min(fieldval.New("name", "ab"), "3"))
	assert.False(t, Min(fieldval.New("name", "1"), "3"))
	assert.True(t, Min(fieldval.New("name", "3"), "3"))

	assert.True(t, Max(fieldval.New("name", "1"), "3"))
	assert.False(t, Max(fieldval.New("name", "9"), "3"))

	// the length rules are the ones that look at the string size
	assert.False(t, MinLength(fieldval.New("name", "ab"), 3))
	assert.True(t, MinLength(fieldval.New("name", "abc"), 3))
	assert.False(t, MaxLength(fieldval.New("name", "abcd"), 3))

	// numbers keep the numeric comparison
	assert.True(t, Min(fieldval.New("age", 18), "18"))
	assert.False(t, Min(fieldval.New("age", 17), "18"))
	assert.True(t, Max(fieldval.New("age", 17), "18"))
}
