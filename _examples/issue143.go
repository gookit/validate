package main

import (
	"github.com/gookit/goutil/cliutil"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/validate"
)

type Data struct {
	Name *string `validate:"required_notNull"`
	Age  *int    `validate:"required"`
}

// go run ./_examples/issue143.go
func main() {
	validate.AddValidator("required_notNull", func(val any) bool {
		dump.Println(val, *val.(*string))
		return false
	})
	validate.AddGlobalMessages(map[string]string{
		"required_notNull": "{field} 不可为空",
	})

	age := 18
	name := ""
	v := validate.New(&Data{
		Name: &name,
		Age:  &age,
	})

	if !v.Validate() {
		dump.P(v.Errors)
	} else {
		cliutil.Infoln("验证成功")
	}
}
