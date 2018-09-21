package validate

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToArray(t *testing.T) {
	is := assert.New(t)
	ss := ToArray("a,b,c", ",")
	is.Len(ss, 3)
	is.Equal(`[]string{"a", "b", "c"}`, fmt.Sprintf("%#v", ss))

	tests := map[string]string{
		// sample => want
		"a,b,c":    `[]string{"a", "b", "c"}`,
		"a,b,c,":   `[]string{"a", "b", "c"}`,
		",a,b,c":   `[]string{"a", "b", "c"}`,
		"a, b,c":   `[]string{"a", "b", "c"}`,
		"a,,b,c":   `[]string{"a", "b", "c"}`,
		"a, , b,c": `[]string{"a", "b", "c"}`,
	}

	for sample, want := range tests {
		ss = ToArray(sample)
		is.Equal(want, fmt.Sprintf("%#v", ss))
	}
}
