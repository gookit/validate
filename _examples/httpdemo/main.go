package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/validate"
)

// UserForm struct
type UserForm struct {
	Name     string
	Email    string
	Age      int
	CreateAt int
	Safe     int
	UpdateAt time.Time
	Code     string
}

func main() {
	mux := http.NewServeMux()

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := validate.FromRequest(r)
		if err != nil {
			panic(err)
		}

		v := data.Create()
		// setting rules
		v.FilterRule("age", "int") // convert value to int

		v.AddRule("name", "required")
		v.AddRule("name", "minLen", 7)
		v.AddRule("age", "max", 99)
		v.AddRule("code", "regex", `\d{4,6}`)

		if v.Validate() { // validate ok
			// safeData := v.SafeData()
			userForm := &UserForm{}
			v.BindSafeData(userForm)

			// do something ...
			fmt.Println(userForm.Name)
		} else {
			fmt.Println(v.Errors)       // all error messages
			fmt.Println(v.Errors.One()) // returns a random error message text
		}
	})

	mux.HandleFunc("post-json", handler1)

	goutil.PanicIfErr(http.ListenAndServe(":8090", mux))
}
