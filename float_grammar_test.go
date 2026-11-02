package validate

import (
	"regexp"
	"testing"
)

func TestFloatStringsRequireMantissaDigits(t *testing.T) {
	for _, input := range []string{".", "e3", "E-3", ".e3", ".E+3", "e0"} {
		t.Run("invalid_"+input, func(t *testing.T) {
			if IsFloat(input) {
				t.Errorf("IsFloat accepted digitless mantissa %q", input)
			}
			if regexp.MustCompile(Float).MatchString(input) {
				t.Errorf("public Float pattern accepted %q", input)
			}
			v := Map(map[string]any{"value": input})
			v.StringRule("value", "required|float")
			if v.Validate() {
				t.Errorf("float rule accepted %q", input)
			}
		})
	}
	for _, input := range []string{"0", "3.4", "2", "1.", ".5", "-0.4", "+3.4", "1e3", "1.e-3", "0.5E+3"} {
		t.Run("valid_"+input, func(t *testing.T) {
			if !IsFloat(input) {
				t.Errorf("rejected valid float %q", input)
			}
		})
	}
	for _, input := range []any{nil, "", 2, "abc", float32(1.5), float64(2.5)} {
		want := false
		switch input.(type) {
		case float32, float64:
			want = true
		}
		if IsFloat(input) != want {
			t.Errorf("typed control %T %v", input, input)
		}
	}
}
