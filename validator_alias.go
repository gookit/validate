package validate

// define validator alias name mapping
var validatorAliases = map[string]string{
	// alias -> real name
	"in":    "enum",
	"num":   "number",
	"range": "between",
	// type
	"int":     "isInt",
	"integer": "isInt",
	"uint":    "isUint",
	"bool":    "isBool",
	"float":   "isFloat",
	"map":     "isMap",
	"ints":    "isInts", // []int
	"str":     "isString",
	"string":  "isString",
	"strings": "isStrings", // []string
	"arr":     "isArray",
	"array":   "isArray",
	"slice":   "isSlice",
	// val
	"regex": "regexp",
	"eq":    "isEqual",
	"equal": "isEqual",
	"intEq": "intEqual",
	"ne":    "notEqual",
	"notEq": "notEqual",
	// int
	"lte":         "max",
	"gte":         "min",
	"lessThan":    "lt",
	"greaterThan": "gt",
	// len
	"len":      "length",
	"lenEq":    "length",
	"lengthEq": "length",
	"minLen":   "minLength",
	"maxLen":   "maxLength",
	"minSize":  "minLength",
	"maxSize":  "maxLength",
	// string rune length
	"strlen":     "stringLength",
	"strLen":     "stringLength",
	"strLength":  "stringLength",
	"runeLength": "stringLength",
	// string
	"ip":        "isIP",
	"ipv4":      "isIPv4",
	"ipv6":      "isIPv6",
	"email":     "isEmail",
	"intStr":    "isIntString",
	"strInt":    "isIntString",
	"intString": "isIntString",
	//
	"hexadecimal":    "isHexadecimal",
	"hasWhitespace":  "hasWhitespace",
	"printableASCII": "isPrintableASCII",
	//
	"ascii":     "isASCII",
	"ASCII":     "isASCII",
	"alpha":     "isAlpha",
	"alphaNum":  "isAlphaNum",
	"alphaDash": "isAlphaDash",
	"base64":    "isBase64",
	"CIDR":      "isCIDR",
	"CIDRv4":    "isCIDRv4",
	"CIDRv6":    "isCIDRv6",
	"dnsName":   "isDNSName",
	"DNSName":   "isDNSName",
	"dataURI":   "isDataURI",
	"empty":     "isEmpty",
	"filePath":  "isFilePath",
	"hexColor":  "isHexColor",
	"isbn10":    "isISBN10",
	"ISBN10":    "isISBN10",
	"isbn13":    "isISBN13",
	"ISBN13":    "isISBN13",
	"json":      "isJSON",
	"JSON":      "isJSON",
	"lat":       "isLatitude",
	"latitude":  "isLatitude",
	"lon":       "isLongitude",
	"longitude": "isLongitude",
	"mac":       "isMAC",
	"multiByte": "isMultiByte",
	"number":    "isNumber",
	"rgbColor":  "isRGBColor",
	"RGBColor":  "isRGBColor",
	"url":       "isURL",
	"URL":       "isURL",
	"uuid":      "isUUID",
	"uuid3":     "isUUID3",
	"uuid4":     "isUUID4",
	"uuid5":     "isUUID5",
	"UUID":      "isUUID",
	"UUID3":     "isUUID3",
	"UUID4":     "isUUID4",
	"UUID5":     "isUUID5",
	"unixPath":  "isUnixPath",
	"winPath":   "isWinPath",
	// date
	"date":    "isDate",
	"gtDate":  "afterDate",
	"ltDate":  "beforeDate",
	"gteDate": "afterOrEqualDate",
	"lteDate": "beforeOrEqualDate",
	// uploaded file
	"img":       "isImage",
	"file":      "isFile",
	"image":     "isImage",
	"mimes":     "inMimeTypes",
	"mimeType":  "inMimeTypes",
	"mimeTypes": "inMimeTypes",
}

// ValidatorName get real validator name.
func ValidatorName(name string) string {
	if rName, ok := validatorAliases[name]; ok {
		return rName
	}

	return name
}
