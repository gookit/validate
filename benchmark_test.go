package validate

import "testing"

// NOTE: The two benchmarks below (BenchmarkFieldSuccess / BenchmarkFieldSuccessParallel)
// are "reuse-instance" form: they call v.Validate() repeatedly on the SAME Validation
// instance. Because Validate() short-circuits when v.hasValidated == true (see
// validating.go:49-52), only the FIRST iteration does real work; the rest just return.
// Their numbers therefore do NOT reflect the construct+parse+validate cost.
//
// The benchmarks added in P0 (BenchmarkStructFlat, BenchmarkStructNested, ...) instead
// build a BRAND-NEW Validation instance every iteration so they measure the full
// "construct (FromStruct/FromMap) + Create + parseRulesFromTag + Validate" pipeline,
// which is exactly what the v1.6.0 TypeMeta/RuleTpl work aims to optimize.

func BenchmarkFieldSuccess(b *testing.B) {
	v := New(M{
		"name": "inhere",
	})
	v.StringRule("name", "len:6")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = v.Validate()
	}
}

func BenchmarkFieldSuccessParallel(b *testing.B) {
	v := New(M{
		"name": "inhere",
	})
	v.StringRule("name", "len:6")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = v.Validate()
		}
	})
}

/*************************************************************
 * P0 baseline benchmarks (full construct + parse + validate)
 *************************************************************/

// flatUser is a 3-field flat struct used by BenchmarkStructFlat / BenchmarkStructFirstParse.
type flatUser struct {
	Name  string `validate:"required|minLen:3"`
	Email string `validate:"required|email"`
	Age   int    `validate:"required|int|min:1|max:120"`
}

// BenchmarkStructFlat measures the full pipeline for a flat 3-field struct:
// every iteration builds a brand-new Validation via Struct(&u) then validates it.
func BenchmarkStructFlat(b *testing.B) {
	u := flatUser{Name: "inhere", Email: "john@example.com", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := Struct(&u)
		v.Validate()
	}
}

// nestedProfile is the inner struct for BenchmarkStructNested (2-level nesting).
type nestedProfile struct {
	City string `validate:"required|minLen:2"`
	Zip  string `validate:"required|minLen:3"`
}

// nestedUser wraps a sub-struct field carrying its own rules.
type nestedUser struct {
	Name    string        `validate:"required|minLen:3"`
	Profile nestedProfile `validate:"required"`
}

// BenchmarkStructNested measures struct-in-struct (2 levels) validation, rebuilt per iteration.
func BenchmarkStructNested(b *testing.B) {
	u := nestedUser{
		Name:    "inhere",
		Profile: nestedProfile{City: "Beijing", Zip: "100000"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := Struct(&u)
		v.Validate()
	}
}

// sliceItem is the element type for the []Sub field, carrying its own rules.
type sliceItem struct {
	Title string `validate:"required|minLen:2"`
	Score int    `validate:"required|int|min:0|max:100"`
}

// sliceUser holds a slice-of-struct field whose elements are validated per index.
type sliceUser struct {
	Name  string      `validate:"required|minLen:3"`
	Items []sliceItem `validate:"required"`
}

// BenchmarkStructSliceOfStruct measures validation of a struct with a []Sub field
// (Sub itself has rules); the slice holds 3 elements. Rebuilt per iteration.
func BenchmarkStructSliceOfStruct(b *testing.B) {
	u := sliceUser{
		Name: "inhere",
		Items: []sliceItem{
			{Title: "go", Score: 90},
			{Title: "rust", Score: 80},
			{Title: "java", Score: 70},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := Struct(&u)
		v.Validate()
	}
}

// BenchmarkMapValidate is the control group: Map data source + StringRules, rebuilt
// and validated each iteration. Expected to stay flat across the v1.6.0 work.
func BenchmarkMapValidate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := New(M{
			"name":  "inhere",
			"email": "john@example.com",
			"age":   30,
		})
		v.StringRules(MS{
			"name":  "required|minLen:3",
			"email": "required|email",
			"age":   "required|int|min:1|max:120",
		})
		v.Validate()
	}
}

// BenchmarkValValue measures the quick single-value validate path Val(val, rule).
func BenchmarkValValue(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Val("john@example.com", "required|email")
	}
}

// BenchmarkStructFirstParse measures the pure construct+parse cost (no Validate):
// every iteration runs Struct(&u) which performs FromStruct + Create + parseRulesFromTag.
func BenchmarkStructFirstParse(b *testing.B) {
	u := flatUser{Name: "inhere", Email: "john@example.com", Age: 30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Struct(&u)
	}
}
