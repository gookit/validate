package main

import "github.com/gookit/validation"

func main() {
	myV := &validation.Validator{
		Name: "test",
		Func: func() error {
			return nil
		},
	}
}
