package validate

import (
	"reflect"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// FieldCtx 是传给(自定义)校验器的字段级上下文句柄:不装箱地拿到字段值(reflect.Value)、
// 字段名/路径与规则参数。命名见 RFC §12 D6(不跟随 go-playground 的 FieldLevel)。
type FieldCtx interface {
	Value() reflect.Value  // 去指针后的字段值(= 内部 RealV)
	Raw() reflect.Value    // 原始 reflect.Value(= 内部 RV)
	FieldName() string     // 字段路径,如 "addr.city"
	Arg(i int) (any, bool) // 规则第 i 个参数
	Args() []any           // 全部规则参数
}

// fieldCtx 是 FieldCtx 的内部实现,由值载体 *fieldval.FieldValue + 字段名 + 规则参数支撑。
type fieldCtx struct {
	fv    *fieldval.FieldValue
	field string
	args  []any
}

var _ FieldCtx = (*fieldCtx)(nil)

func (c *fieldCtx) Value() reflect.Value { return c.fv.RealV() }
func (c *fieldCtx) Raw() reflect.Value   { return c.fv.RV() }
func (c *fieldCtx) FieldName() string    { return c.field }
func (c *fieldCtx) Arg(i int) (any, bool) {
	if i >= 0 && i < len(c.args) {
		return c.args[i], true
	}
	return nil, false
}
func (c *fieldCtx) Args() []any { return c.args }
