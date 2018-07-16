package validation

// Messages internal error message for some rules.
var Messages = map[string]string{
	"_":   "data did not pass verification", // default message
	"min": "%s value min is %d",
	"max": "%s value max is %d",

	"range": "%s value must be in the range %d - %d",
}

// some validator alias name
var aliases = map[string]string{
	"int": "integer",
	"num": "number",
	"str": "string",
	"map": "mapping",
	"arr": "array",

	"regex": "regexp",
}
