package validate

// some validator alias name
var validatorAliases = map[string]string{
	"int": "integer",
	"num": "number",
	"str": "string",
	"map": "mapping",
	"arr": "array",

	"regex": "regexp",
}

// get real validator name.
func validatorName(name string) string {
	if realName, ok := validatorAliases[name]; ok {
		return realName
	}

	return name
}
