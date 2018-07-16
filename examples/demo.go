package main

import (
	"github.com/gookit/validate"
	"fmt"
)

func main() {
	myV := &validate.Validator{
		Name: "test",
		Func: func() error {
			return nil
		},
	}

	fmt.Printf("%v\n", myV)
}
