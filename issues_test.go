package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssues19(t *testing.T) {
	is := assert.New(t)

	// use tag name: country_code
	type smsReq struct {
		CountryCode string `json:"country_code" validate:"required" filter:"trim|lower"`
		Phone       string `json:"phone" validate:"required" filter:"trim"`
		Type        string `json:"type" validate:"required|in:register,forget_password,set_pay_password,reset_pay_password,reset_password" filter:"trim"`
	}

	req := &smsReq{
		" ABcd   ", "13677778888  ", "register",
	}

	v := New(req)
	is.True(v.Validate())
	sd := v.SafeData()
	is.Equal("abcd", sd["CountryCode"])
	is.Equal("13677778888", sd["Phone"])

	// Notice: since 1.2, filtered value will update to struct
	// err := v.BindSafeData(req)
	// is.NoError(err)
	is.Equal("abcd", req.CountryCode)
	is.Equal("13677778888", req.Phone)

	// use tag name: countrycode
	type smsReq1 struct {
		// CountryCode string `json:"countryCode" validate:"required" filter:"trim|lower"`
		CountryCode string `json:"countrycode" validate:"required" filter:"trim|lower"`
		Phone       string `json:"phone" validate:"required" filter:"trim"`
		Type        string `json:"type" validate:"required|in:register,forget_password,set_pay_password,reset_pay_password,reset_password" filter:"trim"`
	}

	req1 := &smsReq1{
		" ABcd   ", "13677778888  ", "register",
	}

	v = New(req1)
	is.True(v.Validate())
	sd = v.SafeData()
	is.Equal("abcd", sd["CountryCode"])

	is.Equal("abcd", req1.CountryCode)
	is.Equal("13677778888", req1.Phone)
}

func TestIssues20(t *testing.T) {
	is := assert.New(t)
	type setProfileReq struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|url" filter:"trim"`
	}

	req := &setProfileReq{"123nickname111", "123"}
	v := New(req)
	is.True(v.Validate())

	type setProfileReq1 struct {
		Nickname string `json:"nickname" validate:"string" filter:"trim"`
		Avatar   string `json:"avatar" validate:"required|fullUrl" filter:"trim"`
	}
	req1 := &setProfileReq1{"123nickname111", "123"}

	v = New(req1)
	is.False(v.Validate())
	is.Len(v.Errors, 1)
	is.Equal("Avatar must be an valid full URL address", v.Errors.One())
}
