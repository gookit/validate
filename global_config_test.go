package validate

import (
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
