package filter

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTrim(t *testing.T) {
	is := assert.New(t)

	is.Equal("abc", Trim("abc "))
	is.Equal("abc", Trim(" abc"))
	is.Equal("abc", Trim(" abc "))
	is.Equal("abc", Trim("abc,,", ","))
	is.Equal("abc", Trim("abc,.", ",."))
	is.Equal("abc", Trim("abc,.", ".,"))
}

func TestInt(t *testing.T) {
	is := assert.New(t)

	intVal, err := Int("2")
	is.Nil(err)
	is.Equal(2, intVal)

	intVal, err = ToInt("-2")
	is.Nil(err)
	is.Equal(-2, intVal)

	is.Equal(2, MustInt("2"))
	is.Equal(-2, MustInt("-2"))
	is.Equal(0, MustInt("2a"))

	uintVal, err := Uint("2")
	is.Nil(err)
	is.Equal(uint64(2), uintVal)
	_, err = ToUint("-2")
	is.Error(err)

	is.Equal(uint64(0), MustUint("-2"))
	is.Equal(uint64(0), MustUint("2a"))
}

func TestStr2Array(t *testing.T) {
	is := assert.New(t)

	ss := Str2Array("a,b,c", ",")
	is.Len(ss, 3)
	is.Equal(`[]string{"a", "b", "c"}`, fmt.Sprintf("%#v", ss))

	tests := []string{
		// sample
		"a,b,c",
		"a,b,c,",
		",a,b,c",
		"a, b,c",
		"a,,b,c",
		"a, , b,c",
	}

	for _, sample := range tests {
		ss = Str2Array(sample)
		is.Equal(`[]string{"a", "b", "c"}`, fmt.Sprintf("%#v", ss))
	}
}
