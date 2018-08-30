package validate

type FieldCtx struct {
	v     *Validation
	Field string
	Value interface{}
}
