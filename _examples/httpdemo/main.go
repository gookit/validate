package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/validate/v2"
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

		vr := v.ValidateR()
		if vr.IsOK() { // validate ok
			// safeData := vr.SafeData()
			userForm := &UserForm{}
			vr.BindSafeData(userForm)

			// do something ...
			fmt.Println(userForm.Name)
		} else {
			fmt.Println(vr.Errors)       // all error messages
			fmt.Println(vr.Errors.One()) // returns a random error message text
		}
	})

	mux.HandleFunc("post-json", handler1)

	goutil.PanicIfErr(http.ListenAndServe(":8090", mux))
}
