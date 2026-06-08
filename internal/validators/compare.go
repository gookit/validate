package validators

import "github.com/gookit/validate/v2/internal/fieldval"

// Gt check value greater dst value.
// only check for: int(X), uint(X), float(X), string.
func Gt(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">") }

// Gte check value greater or equal dst value.
// only check for: int(X), uint(X), float(X), string.
func Gte(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">=") }

// Min check value greater or equal dst value, alias Gte().
// only check for: int(X), uint(X), float(X), string.
func Min(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">=") }

// Lt less than dst value.
// only check for: int(X), uint(X), float(X).
func Lt(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<") }

// Lte less than or equal dst value.
// only check for: int(X), uint(X), float(X).
func Lte(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<=") }

// Max less than or equal dst value, alias Lte().
// only check for: int(X), uint(X), float(X).
func Max(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<=") }

// Between value in the given range (inclusive).
// only check for: int(X), uint(X), float(X), string.
func Between(fl *fieldval.FieldValue, minVal, maxVal any) bool {
	return valueCompare(fl, minVal, ">=") && valueCompare(fl, maxVal, "<=")
}
