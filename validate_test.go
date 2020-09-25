package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMS_String(t *testing.T) {
	ms := MS{}

	assert.Equal(t, "", ms.One())
	assert.Equal(t, "", ms.String())

	ms["key"] = "val"
	assert.Equal(t, "val", ms.One())
	assert.Equal(t, " key: val", ms.String())
}

func TestOption(t *testing.T) {
	opt := Option()

	assert.Equal(t, "json", opt.FieldTag)
	assert.Equal(t, "validate", opt.ValidateTag)

	Config(func(opt *GlobalOption) {
		opt.ValidateTag = "valid"
	})

	opt = Option()
	assert.Equal(t, "valid", opt.ValidateTag)

	ResetOption()
}
