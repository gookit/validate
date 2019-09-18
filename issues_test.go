package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssues19(t *testing.T)  {
	is := assert.New(t)

	type smsReq struct {
		CountryCode string `json:"country_code" validate:"required" filter:"trim|lower"`
		Phone       string `json:"phone" validate:"required" filter:"trim"`
		Type        string `json:"type" validate:"required|in:register,forget_password,set_pay_password,reset_pay_password,reset_password" filter:"trim"`
	}

	req1 := &smsReq{
		" ABcd ", "13677778888", "register",
	}

	v := New(req1)
	is.True(v.Validate())
}

func TestIssues20(t *testing.T)  {
	// is := assert.New(t)

}
