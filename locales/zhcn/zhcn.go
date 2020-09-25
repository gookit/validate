package zhcn

import "github.com/gookit/validate"

// Name language name
const Name = "zh-CN"

// Register language data to validate.Validation
func Register(v *validate.Validation) {
	v.AddMessages(Data)
}

// RegisterGlobal register to the validate global messages
func RegisterGlobal() {
	validate.AddGlobalMessages(Data)
}

// Data zh-CN language messages
var Data = map[string]string{
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
	"required":           "{field} 是必填项",
	"requiredIf":         "当 %v 为 {args} 时 {field} 不能为空。",
	"requiredUnless":     "当 %v 不为 {args} 时 {field} 不能为空。",
	"requiredWith":       "当 {values} 存在时 {field} 不能为空。",
	"requiredWithAll":    "当 {values} 存在时 {field} 不能为空。",
	"requiredWithout":    "当 {values} 不存在时 {field} 不能为空。",
	"requiredWithoutAll": "当 {values} 都不存在时 {field} 不能为空。",
	// email
	"email": "{field}不是合法邮箱",
	// field compare
	"eqField":  "{field} 值必须等于该字段 %s",
	"neField":  "{field} 值不能等于该字段 %s",
	"ltField":  "{field} 值应小于该字段 %s",
	"lteField": "{field} 值应小于等于该字段 %s",
	"gtField":  "{field} 值应大于该字段 %s",
	"gteField": "{field} 值应大于等于该字段 %s",
	// check string
	"isString":     "{field} 值必须是一个字符串",
	"isString1":    "{field} 值必须是一个字符串，最小长度为 %d",
	"stringLength": "{field} 值长度必须在 %d - %d 范围内",
	// check resource
	"isURL":     "{field} 值必须是一个有效的URL地址",
	"isFullURL": "{field} 值必须是一个完整、有效的URL地址",
	"isFile":    "{field} 值必须是一个可上传的文件",
	"isImage":   "{field} 值必须是一个可上传的图像文件",
	// data type
	"bool":    "{field} 值必须是一个bool类型",
	"float":   "{field} 值必须是一个float类型",
	"slice":   "{field} 值必须是一个slice类型",
	"map":     "{field} 值必须是一个map类型",
	"array":   "{field} 值必须是一个array类型",
	"strings": "{field} 值必须是一个[]string类型",
	//
	"notIn":       "{field} 值不能出现在给定枚举列表中 %d",
	"contains":    "{field} 值不能出现在枚举列表中 %s",
	"notContains": "{field} 值包含输入指定值 %s",
	"startsWith":  "{field} 值的前缀必须是：%s ",
	"endsWith":    "{field} 值的后缀必须是：%s ",
	"regex":       "{field} 值没有通过正则匹配",
	"file":        "{field} 值必须是一个文件",
	"image":       "{field} 值必须是一图像",
	// date
	"date":    "{field} 值应该是一个日期字符串",
	"gtDate":  "{field} 日期应该在 %s 之后",
	"ltDate":  "{field} 日期应该在 %s 之前",
	"gteDate": "{field} 日期应该等于 %s 或者在其之后",
	"lteDate": "{field} 日期应该等于 %s 或者在其之前",
	// check char
	"hasWhitespace":  "{field} 值应该包含空格",
	"ascii":          "{field} 值应该是一个 ASCII 字符串",
	"alpha":          "{field} 值仅包含字母字符",
	"alphaNum":       "{field} 值仅包含字母字符和数字",
	"alphaDash":      "{field} 值仅包含字母字符、数字、破折号（-）、下划线（_）",
	"multiByte":      "{field} 值应该是一个多字节字符串",
	"base64":         "{field} 值应该是一个Base64字符串",
	"dnsName":        "{field} 值应该是一个DNS名称字符串",
	"dataURI":        "{field} 值应该是一个DataURI字符串",
	"empty":          "{field} 值应该为空",
	"hexColor":       "{field} 值应该是十六进制的颜色字符串",
	"hexadecimal":    "{field} 值应该是十六进制字符串",
	"json":           "{field} 值应该是一个json字符串",
	"lat":            "{field} 值应该是一个纬度坐标",
	"lon":            "{field} 值应该是一个经度坐标",
	"mac":            "{field} 值应该是一个 MAC 字符串",
	"num":            "{field} 值应该是一个数字字符串(>=0)",
	"cnMobile":       "{field} 值应该是中国11位手机号码字符串",
	"printableASCII": "{field} 值应该是可打印ASCII字符串",
	"rgbColor":       "{field} 值应该是RGP颜色字符串",
	"fullUrl":        "{field} 值应该是一个完整的URL字符串",
	"url":            "{field} 值应该是一个URL字符串",
	"ip":             "{field} 值应该是一个IP（v4或v6）字符串",
	"ipv4":           "{field} 值应该是一个IPv4字符串",
	"ipv6":           "{field} 值应该是一个IPv6字符串",
	"CIDR":           "{field} 值应该是一个CIDR字符串",
	"CIDRv4":         "{field} 值应该是一个CIDRv4字符串",
	"CIDRv6":         "{field} 值应该是一个CIDRv6字符串",
	"uuid":           "{field} 值应该是一个UUID字符串",
	"uuid3":          "{field} 值应该是一个UUID3字符串",
	"uuid4":          "{field} 值应该是一个UUID4字符串",
	"uuid5":          "{field} 值应该是一个UUID5字符串",
	"filePath":       "{field} 值应该是一个存在的文件路径",
	"unixPath":       "{field} 值应该是一个Unix路径字符串",
	"winPath":        "{field} 值应该是一个Windows路径字符串",
	"isbn10":         "{field} 值应该是一个ISBN10字符串",
	"isbn13":         "{field} 值应该是一个ISBN13字符串",
}
