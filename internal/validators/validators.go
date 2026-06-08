// Package validators holds the internal, RV-native implementations of the
// stateless built-in validators. They consume the cached value carrier
// (*fieldval.FieldValue) so the same value is not repeatedly re-reflected
// (reflect.ValueOf) inside each validator.
//
// Dependency direction is one-way: root -> internal/validators ->
// internal/fieldval, internal/reflectx (and goutil). This package never imports
// the validate root package.
package validators

import (
	"reflect"

	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// valueCompare reuses fl's cached RV() to avoid the second reflect.ValueOf done
// inside reflectx.ValueCompare's IndirectValue. The pointer-deref / string
// branch / numeric branch semantics are kept byte-for-byte identical to
// reflectx.ValueCompare.
func valueCompare(fl *fieldval.FieldValue, dstVal any, op string) bool {
	rv := fl.RV()

	var srcVal any
	if rv.Kind() == reflect.Ptr {
		ev := reflect.Indirect(rv)
		if ev.IsValid() {
			srcVal = ev.Interface()
		} else {
			srcVal = nil
		}
	} else {
		// non-pointer: use the original boxed value, equivalent to
		// IndirectValue's non-pointer branch (and without a new boxing alloc).
		srcVal = fl.Src
	}

	// string compare
	if str1, ok := srcVal.(string); ok {
		str2, err := strutil.ToString(dstVal)
		if err != nil {
			return false
		}
		return strutil.Compare(str1, str2, op)
	}

	// as int or float to compare
	return mathutil.Compare(srcVal, dstVal, op)
}
