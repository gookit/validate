package validators

import "github.com/gookit/validate/v2/internal/fieldval"

// Gt check value greater dst value.
// only check for: int(X), uint(X), float(X), string(lexicographic).
func Gt(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">") }

// Gte check value greater or equal dst value.
// only check for: int(X), uint(X), float(X), string(lexicographic).
func Gte(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">=") }

// Min check value greater or equal dst value, alias Gte().
// only check for: int(X), uint(X), float(X), string(lexicographic).
//
// NOTICE: a string field is compared with the rule argument as a string, not by its
// length - `min:3` on "ab" passes and on "1" fails. Use minLength/min_len for a
// length check.
func Min(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, ">=") }

// Lt less than dst value.
// only check for: int(X), uint(X), float(X), string(lexicographic).
func Lt(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<") }

// Lte less than or equal dst value.
// only check for: int(X), uint(X), float(X), string(lexicographic).
func Lte(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<=") }

// Max less than or equal dst value, alias Lte().
// only check for: int(X), uint(X), float(X), string(lexicographic).
// NOTICE: use maxLength/max_len for the length of a string.
func Max(fl *fieldval.FieldValue, dstVal any) bool { return valueCompare(fl, dstVal, "<=") }

// Between value in the given range (inclusive).
// only check for: int(X), uint(X), float(X), string(lexicographic).
func Between(fl *fieldval.FieldValue, minVal, maxVal any) bool {
	return valueCompare(fl, minVal, ">=") && valueCompare(fl, maxVal, "<=")
}
