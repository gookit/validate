package validate

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2/internal/fieldval"
)

func TestFieldCtx_methods(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		fv := fieldval.New("addr.city", "hello")
		fc := &fieldCtx{fv: fv, field: "addr.city", args: []any{"x", 2}}

		assert.Eq(t, "hello", fc.Value().String())
		assert.Eq(t, reflect.String, fc.Raw().Kind())
		assert.Eq(t, "addr.city", fc.FieldName())

		a0, ok0 := fc.Arg(0)
		assert.True(t, ok0)
		assert.Eq(t, "x", a0)

		a5, ok5 := fc.Arg(5)
		assert.False(t, ok5)
		assert.Nil(t, a5)

		assert.Eq(t, 2, len(fc.Args()))
	})

	t.Run("pointer value (de-pointered by Value)", func(t *testing.T) {
		p := new(int)
		*p = 7
		fv := fieldval.New("n", p)
		fc := &fieldCtx{fv: fv, field: "n"}

		assert.Eq(t, reflect.Int, fc.Value().Kind()) // RealV de-pointers
		assert.Eq(t, reflect.Ptr, fc.Raw().Kind())   // RV keeps the pointer
	})
}

func TestNewFuncMeta_style(t *testing.T) {
	t.Run("legacy func(val any) bool", func(t *testing.T) {
		fm := newFuncMeta("legacy", false, reflect.ValueOf(func(val any) bool { return true }))
		assert.Eq(t, styleLegacy, fm.style)
		assert.Nil(t, fm.fcFunc)
	})

	t.Run("fieldctx func(FieldCtx) bool", func(t *testing.T) {
		fm := newFuncMeta("fc", false, reflect.ValueOf(func(fc FieldCtx) bool { return true }))
		assert.Eq(t, styleFieldCtx, fm.style)
		assert.NotNil(t, fm.fcFunc)
	})
}
