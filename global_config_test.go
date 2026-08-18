package validate

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

const issue344Validator = "issue344NonBlank"

type issue344Payload struct {
	Name string `validate:"required|issue344NonBlank"`
}

func (issue344Payload) ConfigValidation(*Validation) {
	AddValidator(issue344Validator, func(val any) bool {
		s, ok := val.(string)
		return ok && s != ""
	})
}

func TestIssue344GlobalValidatorConcurrent(t *testing.T) {
	var failed atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !Struct(&issue344Payload{Name: "ok"}).Validate() {
					failed.Store(true)
				}
			}
		}()
	}
	wg.Wait()
	assert.False(t, failed.Load())
}

func TestGlobalMessageMapIsolation(t *testing.T) {
	original := CopyGlobalMessages()
	t.Cleanup(func() { SetBuiltinMessages(original) })

	messages := MS{"isolated": "original"}
	SetBuiltinMessages(messages)
	messages["isolated"] = "changed input"
	assert.Eq(t, "original", BuiltinMessages()["isolated"])

	snapshot := BuiltinMessages()
	snapshot["isolated"] = "changed output"
	assert.Eq(t, "original", BuiltinMessages()["isolated"])
}

func TestGlobalMessagesConcurrent(t *testing.T) {
	original := BuiltinMessages()["required"]
	defer AddGlobalMessages(MS{"required": original})

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 500; i++ {
			AddGlobalMessages(MS{"required": fmt.Sprintf("required-%d", i)})
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 500; i++ {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Name",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(fmt.Sprintf(`validate:"required" message:"required:custom-%d"`, i)),
			}})
			Struct(reflect.New(typ).Interface())
		}
	}()

	close(start)
	wg.Wait()
}

func TestFreezeGlobal(t *testing.T) {
	if os.Getenv("GO_WANT_FREEZE_GLOBAL_HELPER") == "1" {
		writerStarted := make(chan struct{})
		writerPanic := make(chan any, 1)
		go func() {
			close(writerStarted)
			defer func() { writerPanic <- recover() }()
			for {
				AddGlobalMessages(MS{"freezeRace": "writing"})
			}
		}()
		<-writerStarted
		FreezeGlobal()
		assert.Eq(t, "validate: global configuration is frozen", <-writerPanic)
		FreezeGlobal()

		assertFrozen := func(fn func()) {
			assert.PanicsMsg(t, fn, "validate: global configuration is frozen")
		}
		assertFrozen(func() { AddValidator("frozenValidator", func(any) bool { return true }) })
		assertFrozen(func() { AddValidators(M{"frozenValidators": func(any) bool { return true }}) })
		assertFrozen(func() { AddFilter("frozenFilter", func(val any) any { return val }) })
		assertFrozen(func() { AddFilters(M{"frozenFilters": func(val any) any { return val }}) })
		assertFrozen(func() { AddGlobalMessages(MS{"frozen": "frozen"}) })
		assertFrozen(func() { AddBuiltinMessages(MS{"frozen": "frozen"}) })
		assertFrozen(func() { SetBuiltinMessages(MS{"frozen": "frozen"}) })
		assertFrozen(func() { AddCustomType(func(reflect.Value) any { return nil }, struct{}{}) })
		assertFrozen(ResetCustomTypes)

		v := Map(M{"name": "ok"})
		assert.NotPanics(t, func() { v.AddValidator("localValidator", func(any) bool { return true }) })
		assert.NotPanics(t, func() { v.AddFilter("localFilter", func(val any) any { return val }) })
		assert.NotPanics(t, func() { v.Trans().AddMessage("localMessage", "local") })
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestFreezeGlobal$")
	cmd.Env = append(os.Environ(), "GO_WANT_FREEZE_GLOBAL_HELPER=1")
	out, err := cmd.CombinedOutput()
	assert.NoErr(t, err, string(out))
}
