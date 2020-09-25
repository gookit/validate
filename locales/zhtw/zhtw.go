package zhtw

import "github.com/gookit/validate"

// Name language name
const Name = "zh-TW"

// Register language data to validate.Validation
func Register(v *validate.Validation) {
	v.AddMessages(Data)
}

// RegisterGlobal register to the validate global messages
func RegisterGlobal() {
	validate.AddGlobalMessages(Data)
}

// Data zh-TW language messages
var Data = map[string]string{
	"_": "{field} 沒有通過驗證",
	// int
	"min": "{field} 的最小值是 %d",
	"max": "{field} 的最大值是 %d",
	// Length
	"minLength": "{field} 的最小長度是 %d",
	"maxLength": "{field} 的最大長度是 %d",
	// range
	"enum":  "{field} 值必須在下列枚舉中 %v",
	"range": "{field} 值必須在此範圍內 %d - %d",
	// required
	"required":             "{field} 是必填項",
	"required_if":          "當 %v 為 {args} 時 {field} 不能為空。",
	"required_unless":      "當 %v 不為 {args} 時 {field} 不能為空。",
	"required_with":        "當 {values} 存在時 {field} 不能為空。",
	"required_with_all":    "當 {values} 存在時 {field} 不能為空。",
	"required_without":     "當 {values} 不存在時 {field} 不能為空。",
	"required_without_all": "當 {values} 都不存在時 {field} 不能為空。",
	// email
	"email": "{field}不是合法郵箱",
	// field compare
	"eqField":  "{field} 值必須等於該字段 %s",
	"neField":  "{field} 值不能等於該字段 %s",
	"ltField":  "{field} 值應小於該字段 %s",
	"lteField": "{field} 值應小於等於該字段 %s",
	"gtField":  "{field} 值應大於該字段 %s",
	"gteField": "{field} 值應大於等於該字段 %s",
	// check string
	"isString":     "{field} 值必須是壹個字符串",
	"isString1":    "{field} 值必須是壹個字符串，最小長度為 %d",
	"stringLength": "{field} 值長度必須在 %d - %d 範圍內",
	// check resource
	"isURL":     "{field} 值必須是壹個有效的URL地址",
	"isFullURL": "{field} 值必須是壹個完整、有效的URL地址",
	"isFile":    "{field} 值必須是壹個可上傳的文件",
	"isImage":   "{field} 值必須是壹個可上傳的圖像文件",
	// data type
	"bool":    "{field} 值必須是壹個bool類型",
	"float":   "{field} 值必須是壹個float類型",
	"slice":   "{field} 值必須是壹個slice類型",
	"map":     "{field} 值必須是壹個map類型",
	"array":   "{field} 值必須是壹個array類型",
	"strings": "{field} 值必須是壹個[]string類型",
	//
	"notIn":       "{field} 值不能出現在給定枚舉列表中 %d",
	"contains":    "{field} 值不能出現在枚舉列表中 %s",
	"notContains": "{field} 值包含輸入指定值 %s",
	"startsWith":  "{field} 值的前綴必須是： %s ",
	"endsWith":    "{field} 值的後綴必須是：%s ",
	"regex":       "{field} 值沒有通過正則匹配",
	"file":        "{field} 值必須是壹個文件",
	"image":       "{field} 值必須是壹圖像",
	// date
	"date":    "{field} 值應該是壹個日期字符串",
	"gtDate":  "{field} 日期應該在 %s 之後",
	"ltDate":  "{field} 日期應該在 %s 之前",
	"gteDate": "{field} 日期應該等於 %s 或者在其之後",
	"lteDate": "{field} 日期應該等於 %s 或者在其之前",
	// check char
	"hasWhitespace":  "{field} 值應該包含空格",
	"ascii":          "{field} 值應該是壹個 ASCII 字符串",
	"alpha":          "{field} 值僅包含字母字符",
	"alphaNum":       "{field} 值僅包含字母字符和數字",
	"alphaDash":      "{field} 值僅包含字母字符、數字、破折號（-）、下劃線（_）",
	"multiByte":      "{field} 值應該是壹個多字節字符串",
	"base64":         "{field} 值應該是壹個Base64字符串",
	"dnsName":        "{field} 值應該是壹個DNS名稱字符串",
	"dataURI":        "{field} 值應該是壹個DataURI字符串",
	"empty":          "{field} 值應該為空",
	"hexColor":       "{field} 值應該是十六進制的顏色字符串",
	"hexadecimal":    "{field} 值應該是十六進制字符串",
	"json":           "{field} 值應該是壹個json字符串",
	"lat":            "{field} 值應該是壹個緯度坐標",
	"lon":            "{field} 值應該是壹個經度坐標",
	"mac":            "{field} 值應該是壹個MAC字符串",
	"num":            "{field} 值應該是壹個數字字符串(>=0)",
	"cnMobile":       "{field} 值應該是中國11位手機號碼字符串",
	"printableASCII": "{field} 值應該是可打印ASCII字符串",
	"rgbColor":       "{field} 值應該是RGP顏色字符串",
	"fullUrl":        "{field} 值應該是壹個完整的URL字符串",
	"url":            "{field} 值應該是壹個URL字符串",
	"ip":             "{field} 值應該是壹個IP（v4或v6）字符串",
	"ipv4":           "{field} 值應該是壹個IPv4字符串",
	"ipv6":           "{field} 值應該是壹個IPv6字符串",
	"CIDR":           "{field} 值應該是壹個CIDR字符串",
	"CIDRv4":         "{field} 值應該是壹個CIDRv4字符串",
	"CIDRv6":         "{field} 值應該是壹個CIDRv6字符串",
	"uuid":           "{field} 值應該是壹個UUID字符串",
	"uuid3":          "{field} 值應該是壹個UUID3字符串",
	"uuid4":          "{field} 值應該是壹個UUID4字符串",
	"uuid5":          "{field} 值應該是壹個UUID5字符串",
	"filePath":       "{field} 值應該是壹個存在的文件路徑",
	"unixPath":       "{field} 值應該是壹個Unix路徑字符串",
	"winPath":        "{field} 值應該是壹個Windows路徑字符串",
	"isbn10":         "{field} 值應該是壹個ISBN10字符串",
	"isbn13":         "{field} 值應該是壹個ISBN13字符串",
}

