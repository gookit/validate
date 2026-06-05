package validate

import (
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/gookit/goutil/jsonutil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/validate/v2/internal/reflectx"
)

/*************************************************************
 * region global: string validators
 *************************************************************/

// HasWhitespace check. eg "10"
func HasWhitespace(s string) bool {
	return s != "" && strings.ContainsRune(s, ' ')
}

// IsIntString check. eg "10"
func IsIntString(s string) bool { return s != "" && rxInt.MatchString(s) }

// IsASCII string.
func IsASCII(s string) bool { return s != "" && rxASCII.MatchString(s) }

// IsPrintableASCII string.
func IsPrintableASCII(s string) bool {
	return s != "" && rxPrintableASCII.MatchString(s)
}

// IsBase64 string.
func IsBase64(s string) bool { return s != "" && rxBase64.MatchString(s) }

// IsLatitude string.
func IsLatitude(s string) bool { return s != "" && rxLatitude.MatchString(s) }

// IsLongitude string.
func IsLongitude(s string) bool { return s != "" && rxLongitude.MatchString(s) }

// IsDNSName string.
func IsDNSName(s string) bool { return s != "" && rxDNSName.MatchString(s) }

// HasURLSchema string.
func HasURLSchema(s string) bool { return s != "" && rxURLSchema.MatchString(s) }

// IsFullURL string.
func IsFullURL(s string) bool { return s != "" && rxFullURL.MatchString(s) }

// IsURL string. This is a loose URI-reference check (relative refs, paths and
// bare hosts are accepted); for a strict absolute URL use IsFullURL.
func IsURL(s string) bool {
	if s == "" {
		return false
	}

	// a URL/URI reference cannot contain raw whitespace; url.Parse is otherwise
	// lenient enough to accept things like "not a url" (#138).
	if strings.ContainsAny(s, " \t\r\n\f\v") {
		return false
	}

	_, err := url.Parse(s)
	return err == nil
}

// IsDataURI string.
//
// data:[<mime type>] ( [;charset=<charset>] ) [;base64],码内容
// eg. "data:image/gif;base64,R0lGODlhA..."
func IsDataURI(s string) bool { return s != "" && rxDataURI.MatchString(s) }

// IsMultiByte string.
func IsMultiByte(s string) bool { return s != "" && rxMultiByte.MatchString(s) }

// IsISBN10 string.
func IsISBN10(s string) bool { return s != "" && rxISBN10.MatchString(s) }

// IsISBN13 string.
func IsISBN13(s string) bool { return s != "" && rxISBN13.MatchString(s) }

// IsHexadecimal string.
func IsHexadecimal(s string) bool { return s != "" && rxHexadecimal.MatchString(s) }

// IsCnMobile string.
func IsCnMobile(s string) bool { return s != "" && rxCnMobile.MatchString(s) }

// IsHexColor string.
func IsHexColor(s string) bool { return s != "" && rxHexColor.MatchString(s) }

// IsRGBColor string.
func IsRGBColor(s string) bool { return s != "" && rxRGBColor.MatchString(s) }

// IsAlpha string.
func IsAlpha(s string) bool { return s != "" && rxAlpha.MatchString(s) }

// IsAlphaNum string.
func IsAlphaNum(s string) bool { return s != "" && rxAlphaNum.MatchString(s) }

// IsAlphaDash string.
func IsAlphaDash(s string) bool { return s != "" && rxAlphaDash.MatchString(s) }

// IsNumber string. should >= 0
func IsNumber(v any) bool {
	v = reflectx.IndirectValue(v)

	if v == nil {
		return false
	}

	if s, err := strutil.ToString(v); err == nil {
		return s != "" && rxNumber.MatchString(s)
	}
	return false
}

// IsNumeric is string/int number. should >= 0
func IsNumeric(v any) bool {
	v = reflectx.IndirectValue(v)

	if v == nil {
		return false
	}

	if s, err := strutil.ToString(v); err == nil {
		return s != "" && rxNumber.MatchString(s)
	}
	return false
}

// IsStringNumber is string number. should >= 0
func IsStringNumber(s string) bool { return s != "" && rxNumber.MatchString(s) }

// IsEmail check
func IsEmail(s string) bool { return s != "" && rxEmail.MatchString(s) }

// IsUUID string
func IsUUID(s string) bool { return s != "" && rxUUID.MatchString(s) }

// IsUUID3 string
func IsUUID3(s string) bool { return s != "" && rxUUID3.MatchString(s) }

// IsUUID4 string
func IsUUID4(s string) bool { return s != "" && rxUUID4.MatchString(s) }

// IsUUID5 string
func IsUUID5(s string) bool { return s != "" && rxUUID5.MatchString(s) }

// IsIP is the validation function for validating if the field's value is a valid v4 or v6 IP address.
func IsIP(s string) bool { return s != "" && net.ParseIP(s) != nil }

// IsIPv4 is the validation function for validating if a value is a valid v4 IP address.
func IsIPv4(s string) bool {
	if s == "" {
		return false
	}

	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

// IsIPv6 is the validation function for validating if the field's value is a valid v6 IP address.
func IsIPv6(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() == nil
}

// IsMAC is the validation function for validating if the field's value is a valid MAC address.
func IsMAC(s string) bool {
	if s == "" {
		return false
	}
	_, err := net.ParseMAC(s)
	return err == nil
}

// IsCIDRv4 is the validation function for validating if the field's value is a valid v4 CIDR address.
func IsCIDRv4(s string) bool {
	if s == "" {
		return false
	}
	ip, _, err := net.ParseCIDR(s)
	return err == nil && ip.To4() != nil
}

// IsCIDRv6 is the validation function for validating if the field's value is a valid v6 CIDR address.
func IsCIDRv6(s string) bool {
	if s == "" {
		return false
	}

	ip, _, err := net.ParseCIDR(s)
	return err == nil && ip.To4() == nil
}

// IsCIDR is the validation function for validating if the field's value is a valid v4 or v6 CIDR address.
func IsCIDR(s string) bool {
	if s == "" {
		return false
	}

	_, _, err := net.ParseCIDR(s)
	return err == nil
}

// IsJSON check if the string is valid JSON (note: uses json.Unmarshal).
func IsJSON(s string) bool {
	if s == "" {
		return false
	}
	if !jsonutil.IsJSONFast(s) {
		return false
	}

	var js json.RawMessage
	return Unmarshal([]byte(s), &js) == nil
}

// HasLowerCase check string has lower case
func HasLowerCase(s string) bool {
	if s == "" {
		return false
	}
	return rxHasLowerCase.MatchString(s)
}

// HasUpperCase check string has upper case
func HasUpperCase(s string) bool {
	return s != "" && rxHasUpperCase.MatchString(s)
}

// StartsWith check string is starts with sub-string
func StartsWith(s, sub string) bool { return s != "" && strings.HasPrefix(s, sub) }

// EndsWith check string is ends with sub-string
func EndsWith(s, sub string) bool { return s != "" && strings.HasSuffix(s, sub) }

// StringContains check string is containing sub-string
func StringContains(s, sub string) bool { return s != "" && strings.Contains(s, sub) }

// Regexp match value string
func Regexp(str string, pattern string) bool {
	ok, _ := regexp.MatchString(pattern, str)
	return ok
}
