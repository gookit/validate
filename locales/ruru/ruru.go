package ruru

import "github.com/gookit/validate"

// Name language name
const Name = "ru-RU"

// Register language data to validate.Validation
func Register(v *validate.Validation) {
	v.AddMessages(Data)
}

// RegisterGlobal register to the validate global messages
func RegisterGlobal() {
	validate.AddGlobalMessages(Data)
}

// Data ru-RU language messages
var Data = map[string]string{
	"_":         "Поле {field} не прошло проверку",
	"_validate": "Поле {field} не прошло проверку",
	"_filter":   "Значение {field} некорректно",
	// int
	"min": "Минимальное значение {field} равно %v",
	"max": "Максимальное значение {field} равно %v",
	// type check: int
	"isInt":  "{field} должно быть числом",
	"isInt1": "{field} должно быть числом и не менее %d",         // has min check
	"isInt2": "{field} должно быть числом и в диапазоне %d - %d", // has min, max check
	"isInts": "{field} должно быть массивом чисел",
	"isUint": "{field} должно быть положительным числом",
	// type check: string
	"isString":  "{field} должно быть строкой",
	"isString1": "{field} должно быть строкой с минимальной длиной %d", // has min len check
	// length
	"minLength": "Длина {field} должна быть не меньше %d",
	"maxLength": "Длина {field} должна быть не более %d",
	// string length. calc rune
	"stringLength":  "Длина {field} должна быть в диапазоне %d - %d",
	"stringLength1": "Минимальная длина {field} равна %d",
	"stringLength2": "Длина {field} должна быть в диапазоне %d - %d",

	"isURL":     "{field} должно быть корректным URL адресом",
	"isFullURL": "{field} должно быть корректным полным URL адресом",

	"isFile":  "{field} должно быть загруженным файлом",
	"isImage": "{field} должно быть изображением",

	"enum":  "{field} должно иметь одно из указанных значений: %v",
	"range": "{field} должно быть в диапазоне %d - %d",
	// int compare
	"lt": "Значение {field} должно быть меньше %d",
	"gt": "Значение {field} должно быть больше %d",
	// required
	"required":           "{field} не может быть пустым",
	"requiredIf":         "{field} не может быть пустым, когда {args0} равно {args1end}",
	"requiredUnless":     "{field} не может быть пустым, если {args0} не равно {args1end}",
	"requiredWith":       "{field} не может быть пустым при наличии {values}",
	"requiredWithAll":    "{field} не может быть пустым при наличии {values}",
	"requiredWithout":    "{field} не может быть пустым, если поле {values} пустое",
	"requiredWithoutAll": "{field} не может быть пустым, если ни одной из {values} не присутствует",
	// field compare
	"eqField":  "{field} должно быть равно полю %s",
	"neField":  "{field} не может быть равно полю %s",
	"ltField":  "{field} должно быть меньше значения поля %s",
	"lteField": "{field} должно быть меньше или равно значению поля %s",
	"gtField":  "{field} должно быть больше значения поля %s",
	"gteField": "{field} должно быть больше или равно значению поля %s",
	// data type
	"bool":    "{field} должно быть логическим",
	"float":   "{field} должно быть плавающим числом",
	"slice":   "{field} должно быть слайсом",
	"map":     "{field} должно быть картой",
	"array":   "{field} должно быть массивом",
	"strings": "{field} должно быть массивом строк",
	"notIn":   "{field} не должно быть в данном списке %d",
	//
	"contains":    "{field} должно содержать %s",
	"notContains": "{field} не должно содержать %s",
	"startsWith":  "{field} должно начинаться с %s",
	"endsWith":    "{field} должно заканчиваться на %s",
	"email":       "{field} должно быть электронной почтой",
	"regex":       "{field} не прошло проверку регулярным выражением",
	"file":        "{field} должно быть файлом",
	"image":       "{field} должно быть изображением",
	// date
	"date":    "{field} должно быть строкой даты",
	"gtDate":  "{field} должно быть датой после %s",
	"ltDate":  "{field} должно быть датой до %s",
	"gteDate": "{field} должно быть датой после %s включительно",
	"lteDate": "{field} должно быть датой до %s включительно",
	// check char
	"hasWhitespace":  "{field} должно содержать пробелы",
	"ascii":          "{field} должно быть ASCII строкой",
	"alpha":          "{field} содержит только буквы",
	"alphaNum":       "{field} содержит только буквы и числа",
	"alphaDash":      "{field} содержит только буквы, цифры, тире (-) и подчеркивания (_)",
	"multiByte":      "{field} должно быть многобайтовой строкой",
	"base64":         "{field} должно быть base64 строкой",
	"dnsName":        "{field} должно быть DNS строкой",
	"dataURI":        "{field} должно быть DataURL строкой",
	"empty":          "{field} должно быть пустым",
	"hexColor":       "{field} должно быть цветовой шестнадцатеричной (HEX) строкой",
	"hexadecimal":    "{field} должно быть шестнадцатеричной (HEX) строкой",
	"json":           "{field} должно быть json строкой",
	"lat":            "{field} должно быть координатами широты",
	"lon":            "{field} должно быть координатами долготы",
	"num":            "{field} должно быть цифровой строкой (>=0)",
	"mac":            "{field} должно быть MAC адресом",
	"printableASCII": "{field} должно быть печатаемой ASCII строкой",
	"rgbColor":       "{field} должно быть строкой RGB цвета",
	"fullURL":        "{field} должно быть полной строкой URL-адреса",
	"full":           "{field} должно быть строкой URL-адреса",
	"ip":             "{field} должно быть строкой ip адреса (v4 или v6)",
	"ipv4":           "{field} должно быть ipv4 строкой",
	"ipv6":           "{field} должно быть ipv6 строкой",
	"CIDR":           "{field} должно быть CIDR строкой",
	"CIDRv4":         "{field} должно быть CIDRv4 строкой",
	"CIDRv6":         "{field} должно быть CIDRv6 строкой",
	"uuid":           "{field} должно быть UUID строкой",
	"uuid3":          "{field} должно быть UUID3 строкой",
	"uuid4":          "{field} должно быть UUID4 строкой",
	"uuid5":          "{field} должно быть UUID5 строкой",
	"filePath":       "{field} должно быть существующим путем к файлу",
	"unixPath":       "{field} должно быть строкой пути unix",
	"winPath":        "{field} должно быть строкой пути Windows",
	"isbn10":         "{field} должно быть isbn10 строкой",
	"isbn13":         "{field} должно быть isbn13 строкой",
}
