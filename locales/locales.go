package locales

import "github.com/gookit/validate"

// Locales supported language data map
var Locales = map[string]validate.MS{
	"zh-CN": zhCN,
}

// Register language data to Validation
func Register(v *validate.Validation, name string) bool {
	if data, ok := Locales[name]; ok {
		v.AddMessages(data)
		return true
	}

	return false
}

// zh-CN language messages
var zhCN = map[string]string{
	"_": "{field} 没有通过验证",
	// int
	"min": "{field} 的最小值是 %d",
	"max": "{field} 的最大值是 %d",
	// Length
	"minLength": "{field} 的最小长度是 %d",
	"maxLength": "{field} 的最大长度是 %d",
	// range
	"enum":  "{field} 值必须在下列枚举中 %v",
	"range": "{field} 值必须在此范围内 %d - %d",
	// required
	"required":    "{field} 是必填项",
	"required_if": "当 %v 的值在下列枚举中时 {sArgs}, {field} 是必填项",
	// email
	"email": "{field}不是合法邮箱",
	// field compare
	"eqField":  "{field} 值必须等于该字段 %s",
	"neField":  "{field} 值不能等于该字段 %s",
	"ltField":  "{field} 值应小于该字段 %s",
	"lteField": "{field} 值应小于等于该字段 %s",
	"gtField":  "{field} 值应大于该字段 %s",
	"gteField": "{field} 值应大于等于该字段 %s",
}
