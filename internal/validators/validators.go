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

// Predeclared (concrete) reflect.Types used to gate compareRV's box-free fast
// paths. Only fields whose dynamic type is *exactly* one of these may skip the
// rv.Interface() boxing, because that is precisely the set of types for which
// mathutil.Compare (goutil) and the box-free path are provably equivalent —
// named types (type MyInt int, etc.) fall to compareAny, see compareRV.
var (
	stringType  = reflect.TypeOf("")
	float64Type = reflect.TypeOf(float64(0))
	float32Type = reflect.TypeOf(float32(0))

	intType   = reflect.TypeOf(int(0))
	int8Type  = reflect.TypeOf(int8(0))
	int16Type = reflect.TypeOf(int16(0))
	int32Type = reflect.TypeOf(int32(0))
	int64Type = reflect.TypeOf(int64(0))

	uintType   = reflect.TypeOf(uint(0))
	uint8Type  = reflect.TypeOf(uint8(0))
	uint16Type = reflect.TypeOf(uint16(0))
	uint32Type = reflect.TypeOf(uint32(0))
	uint64Type = reflect.TypeOf(uint64(0))
)

// compareAny is the exact pre-refactor behavior of valueCompare's non-pointer
// branch: a concrete-type string goes through strutil.Compare, everything else
// is handed to mathutil.Compare. Both the pointer branch of valueCompare and
// compareRV's fallback reuse it so results stay byte-for-byte identical.
func compareAny(srcVal, dstVal any, op string) bool {
	if str1, ok := srcVal.(string); ok {
		str2, err := strutil.ToString(dstVal)
		if err != nil {
			return false
		}
		return strutil.Compare(str1, str2, op)
	}
	return mathutil.Compare(srcVal, dstVal, op)
}

// valueCompare reuses fl's cached RV() to avoid the second reflect.ValueOf done
// inside reflectx.ValueCompare's IndirectValue. The pointer-deref / string
// branch / numeric branch semantics are kept byte-for-byte identical to
// reflectx.ValueCompare.
func valueCompare(fl *fieldval.FieldValue, dstVal any, op string) bool {
	rv := fl.RV()

	if rv.Kind() == reflect.Ptr {
		// pointer field (not on the bench / 0-alloc hot path): keep the exact
		// pre-refactor boxing behavior via compareAny.
		ev := reflect.Indirect(rv)
		var sv any
		if ev.IsValid() {
			sv = ev.Interface()
		}
		return compareAny(sv, dstVal, op)
	}

	// non-pointer: stay box-free for the provably-equivalent concrete forms,
	// otherwise fall back to compareAny (boxing, exact behavior).
	return compareRV(rv, dstVal, op)
}

// compareRV compares a non-pointer field value held as reflect.Value against
// dstVal without boxing the field value for the common concrete numeric/string
// types. Correctness contract: for any rv it must return exactly the same
// result as compareAny(rv.Interface(), dstVal, op).
//
// The fast paths are gated on the *exact* concrete type (rv.Type() == predeclared
// type), NOT on rv.Kind(): goutil's mathutil.ToInt64 dispatches on the concrete
// type, so a named numeric type (e.g. type MyInt int) does NOT convert and
// mathutil.Compare returns false for it. Gating on Kind would wrongly succeed
// for named types, so named float / named string / named int|uint all fall to
// the compareAny fallback (boxing) — matching mathutil's default→ToInt64 result.
func compareRV(rv reflect.Value, dstVal any, op string) bool {
	rt := rv.Type()
	// concrete string -> strutil path (compareAny's string branch); independent
	// of dstVal==nil since strutil.ToString(nil) is handled there too.
	if rt == stringType {
		str2, err := strutil.ToString(dstVal)
		if err != nil {
			return false
		}
		return strutil.Compare(rv.String(), str2, op)
	}
	// numeric fast paths feed mathutil.Compare's branches, which short-circuits
	// to false when either operand is nil (the field rv is always non-nil here,
	// so only dstVal matters). mathutil.ToInt64(nil) would NOT error (returns 0),
	// so reproduce mathutil.Compare's nil guard explicitly to stay equivalent.
	if dstVal == nil {
		return false
	}
	switch rt {
	case float64Type: // concrete float64 -> mathutil case float64
		f2, err := mathutil.ToFloat(dstVal)
		if err != nil {
			return false
		}
		return mathutil.CompFloat(rv.Float(), f2, op)
	case float32Type: // concrete float32 -> mathutil case float32: CompFloat(float64(f), ToFloat(dst))
		f2, err := mathutil.ToFloat(dstVal)
		if err != nil {
			return false
		}
		// rv.Float() widens the float32 to its exact float64(fVal), same as mathutil.
		return mathutil.CompFloat(rv.Float(), f2, op)
	case intType, int8Type, int16Type, int32Type, int64Type:
		// concrete intX -> mathutil default: CompInt64(ToInt64(first), ToInt64(dst)).
		// ToInt64(intX) == rv.Int() for these concrete types.
		i2, err := mathutil.ToInt64(dstVal)
		if err != nil {
			return false
		}
		return mathutil.CompInt64(rv.Int(), i2, op)
	case uintType, uint8Type, uint16Type, uint32Type, uint64Type:
		// concrete uintX -> mathutil default: ToInt64(uintX) == int64(rv.Uint())
		// (incl. uint64 overflow wrap to negative int64, matching int64(uint64) in ToInt64).
		i2, err := mathutil.ToInt64(dstVal)
		if err != nil {
			return false
		}
		return mathutil.CompInt64(int64(rv.Uint()), i2, op)
	}

	// everything else (named numeric / named string / bool / NilObject / other
	// concrete types) -> exact fallback (boxing, not on the 0-alloc hot path).
	return compareAny(rv.Interface(), dstVal, op)
}
