package validate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinMessages(t *testing.T) {
	bm := BuiltinMessages()
	assert.NotContains(t, bm, "not-exists")

	AddBuiltinMessages(map[string]string{
		"not-exists": "message value",
	})

	bm = BuiltinMessages()

	assert.Contains(t, bm, "not-exists")
}

func TestErrorsBasic(t *testing.T) {
	es := Errors{}

	assert.True(t, es.Empty())
	assert.Equal(t, "", es.One())

	es.Add("test", "v0", "err msg0")
	assert.Len(t, es, 1)
	assert.Equal(t, "err msg0", es.One())
	assert.Equal(t, "err msg0", es.FieldOne("test"))
	assert.Equal(t, "test:\n v0: err msg0", es.String())

	es.Add("test2", "v1", "err msg2")
	assert.Contains(t, fmt.Sprintf("%v", es.All()), "test:map[v0:err msg0]")
	assert.Contains(t, fmt.Sprintf("%v", es.All()), "test2:map[v1:err msg2]")

	es.Add("test", "v1", "err msg1")
	assert.Len(t, es.Field("test"), 2)
}

func TestTranslatorBasic(t *testing.T) {
	tr := NewTranslator()

	assert.True(t, tr.HasMessage("min"))
	assert.False(t, tr.HasMessage("not-exists"))
	assert.False(t, tr.HasField("FIELD1"))

	tr.AddMessage("FIELD1.min", "{field} message1")
	assert.True(t, tr.HasMessage("FIELD1.min"))
	assert.Equal(t, "FIELD1 message1", tr.Message("min", "FIELD1"))

	tr.AddFieldMap(map[string]string{"FIELD1": "Show Name"})
	assert.Equal(t, "Show Name message1", tr.Message("min", "FIELD1"))

	tr.Reset()
}
