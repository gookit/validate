package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

func TestContainsNilContainer(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		container, element    any
		contains, notContains bool
	}{
		{"nil", nil, "x", false, false},
		{"unsupported scalar", 42, "x", false, false},
		{"nil slice", []string(nil), "x", false, true},
		{"nil map", map[string]int(nil), "x", false, true},
		{"slice match", []string{"x"}, "x", true, false},
		{"slice miss", []string{"x"}, "y", false, true},
		{"nil element", []any{nil}, nil, true, false},
		{"map key", map[string]int{"x": 1}, "x", true, false},
		{"substring", "example", "amp", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Eq(t, tc.contains, Contains(tc.container, tc.element))
			assert.Eq(t, tc.notContains, NotContains(tc.container, tc.element))
		})
	}
}
