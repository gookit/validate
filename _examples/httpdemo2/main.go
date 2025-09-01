package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gookit/goutil"
	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/gookit/validate"
)

func main() {
	mux := http.NewServeMux()
	// 字符串转整型
	validate.AddFilter("newIsInt", func(val any) any {
		switch val.(type) {
		case int:
			return val.(int)
		case string:
			num, parseErr := strconv.Atoi(val.(string))
			if parseErr != nil {
				return errors.New("参数类型错误！")
			}
			return num
		default:
			return 0
		}
	})

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump.P(r.Header.Get("Content-Type"))
		data, err := validate.FromRequest(r)
		if err != nil {
			panic(err)
		}
		dump.P(data)
		v := data.Create()
		// setting rules
		v.
			StringRule("categoryId", "required|int|min:1", "newIsInt").
			AddMessages(map[string]string{
				"required": "the {field} is required",
				"min":      "{field} min value is %d",
				"int":      "the {field} must be integer",
			})

		if v.Validate() { // validate ok
			dump.P(v.SafeData())
			_, _ = w.Write([]byte("hello"))
		} else {
			fmt.Println(v.Errors.One()) // returns a random error message text
			fmt.Println(v.Errors)       // all error messages
			_, _ = w.Write([]byte(v.Errors.String()))
		}
	})

	mux.HandleFunc("/post-json", handler1)

	ccolor.Cyanln("server start on localhost:8090")
	goutil.PanicIfErr(http.ListenAndServe(":8090", mux))
}
