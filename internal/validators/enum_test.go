package validators

import (
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/fieldval"
)

func TestEnumSliceValue(t *testing.T) {
	enum := []string{"go", "php"}

	tests := []struct {
		name  string
		val   any
		in    bool
		notIn bool
	}{
		// scalar values keep the old semantics: not_in is the inverse of in
		{name: "scalar hit", val: "go", in: true, notIn: false},
		{name: "scalar miss", val: "java", in: false, notIn: true},
		// a slice is "in" when every element is in the enum
		{name: "slice all in", val: []string{"go", "php"}, in: true, notIn: false},
		{name: "slice one in", val: []string{"go"}, in: true, notIn: false},
		{name: "slice one out", val: []string{"go", "java"}, in: false, notIn: false},
		{name: "slice all out", val: []string{"java", "rust"}, in: false, notIn: true},
		{name: "empty slice", val: []string{}, in: true, notIn: true},
		{name: "array all in", val: [2]string{"go", "php"}, in: true, notIn: false},
		{name: "array one out", val: [2]string{"go", "java"}, in: false, notIn: false},
		{name: "slice with nil pointer", val: []*string{nil, ptrOf("go")}, in: true, notIn: false},
		{name: "slice with pointer hit", val: []*string{ptrOf("php")}, in: true, notIn: false},
		{name: "slice with pointer miss", val: []*string{ptrOf("java")}, in: false, notIn: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl := fieldval.New("tags", tt.val)
			assert.Eq(t, tt.in, Enum(fl, enum))
			assert.Eq(t, tt.notIn, NotIn(fl, enum))
		})
	}

	// numeric slices go through the int64 path
	intEnum := []string{"1", "2"}
	assert.True(t, Enum(fieldval.New("ids", []int{1, 2}), intEnum))
	assert.False(t, Enum(fieldval.New("ids", []int{1, 3}), intEnum))
	assert.True(t, Enum(fieldval.New("ids", []uint8{2}), intEnum))
	assert.False(t, NotIn(fieldval.New("ids", []int{2, 3}), intEnum))
	assert.True(t, NotIn(fieldval.New("ids", []int{3, 4}), intEnum))
}

func ptrOf(s string) *string { return &s }
