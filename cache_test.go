package validate

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/gookit/goutil/x/assert"
)

type cacheUserSub struct {
	City string `validate:"required" label:"城市"`
	Zip  string `validate:"required|minLen:3"`
}

type cacheUser struct {
	Name string `validate:"required|minLen:3" filter:"trim" json:"name" label:"用户名" message:"required:name is required"`
	Age  int    `validate:"required|min:1"`
	Sub  cacheUserSub
	age  int // unexported, ignored unless ValidatePrivateFields
}

func TestGetTypeMeta_hitSameInstance(t *testing.T) {
	defer func() {
		ResetTypeCache()
	}()

	rt := reflect.TypeOf(cacheUser{})
	m1 := getTypeMeta(rt)
	m2 := getTypeMeta(rt)
	assert.NotNil(t, m1)
	assert.Same(t, m1, m2)
}

func TestGetTypeMeta_ptrAndValueShareMeta(t *testing.T) {
	defer ResetTypeCache()

	// FromStruct uses reflects.Elem, so the cache key for *T and T is the same
	// elem type. Mirror that here.
	mv := getTypeMeta(reflect.TypeOf(cacheUser{}))
	mp := getTypeMeta(reflect.TypeOf(&cacheUser{}).Elem())
	assert.Same(t, mv, mp)
}

func TestGetTypeMeta_fields(t *testing.T) {
	defer ResetTypeCache()
	m := getTypeMeta(reflect.TypeOf(cacheUser{}))

	t.Run("top fields", func(t *testing.T) {
		name := m.byName["Name"]
		assert.NotNil(t, name)
		assert.Eq(t, []int{0}, name.Index)
		assert.Eq(t, "Name", name.Path)
		assert.Eq(t, reflect.String, name.Kind)
		assert.Eq(t, "required|minLen:3", name.ValidateRule)
		assert.Eq(t, "trim", name.FilterRule)
		assert.Eq(t, "name", name.OutputName)
		assert.Eq(t, "用户名", name.Label)
		assert.Eq(t, "required:name is required", name.MessageRaw)
		assert.Eq(t, elemLeaf, name.Elem)

		age := m.byName["Age"]
		assert.NotNil(t, age)
		assert.Eq(t, reflect.Int, age.Kind)
	})

	t.Run("nested struct field", func(t *testing.T) {
		sub := m.byName["Sub"]
		assert.NotNil(t, sub)
		assert.Eq(t, elemStruct, sub.Elem)
		assert.Eq(t, []int{2}, sub.Index)

		city := m.byName["Sub.City"]
		assert.NotNil(t, city)
		assert.Eq(t, "Sub.City", city.Path)
		// Index must be parent chain (Sub=index 2) + field index (City=0)
		assert.Eq(t, []int{2, 0}, city.Index)
		assert.Eq(t, "城市", city.Label)

		zip := m.byName["Sub.Zip"]
		assert.NotNil(t, zip)
		assert.Eq(t, []int{2, 1}, zip.Index)
		assert.Eq(t, "required|minLen:3", zip.ValidateRule)
	})

	t.Run("unexported field skipped by default", func(t *testing.T) {
		_, ok := m.byName["age"]
		assert.False(t, ok)
	})

	t.Run("Index resolves correct field value", func(t *testing.T) {
		u := cacheUser{Name: "tom", Sub: cacheUserSub{City: "NYC"}}
		rv := reflect.ValueOf(u)
		assert.Eq(t, "tom", rv.FieldByIndex(m.byName["Name"].Index).String())
		assert.Eq(t, "NYC", rv.FieldByIndex(m.byName["Sub.City"].Index).String())
	})
}

func TestGetTypeMeta_implements(t *testing.T) {
	defer ResetTypeCache()

	t.Run("plain struct implements nothing", func(t *testing.T) {
		m := getTypeMeta(reflect.TypeOf(cacheUser{}))
		assert.False(t, m.implConfig)
		assert.False(t, m.implTranslates)
		assert.False(t, m.implMessages)
	})
}

func TestGetTypeMeta_tagVerInvalidation(t *testing.T) {
	// IMPORTANT: this test mutates gOpt + tagVer + cache; restore at the end.
	defer func() {
		ResetOption()
		ResetTypeCache()
	}()

	rt := reflect.TypeOf(cacheUser{})
	m1 := getTypeMeta(rt)

	// changing a tag name bumps tagVer -> new cache key -> rebuild
	Config(func(o *GlobalOption) { o.ValidateTag = "valid" })
	m2 := getTypeMeta(rt)
	assert.NotSame(t, m1, m2)

	// after restore, key reverts to original tagVer; but the cache was bumped
	// twice (Config + ResetOption), so the original m1 entry is unreachable and
	// a fresh build happens. Just assert it builds without panic.
	ResetOption()
	m3 := getTypeMeta(rt)
	assert.NotNil(t, m3)
}

func TestResetTypeCache(t *testing.T) {
	defer ResetTypeCache()

	rt := reflect.TypeOf(cacheUser{})
	m1 := getTypeMeta(rt)
	ResetTypeCache()
	m2 := getTypeMeta(rt)
	// after clearing, a fresh meta instance is built
	assert.NotSame(t, m1, m2)
}

func TestGetTypeMeta_privateFields(t *testing.T) {
	defer func() {
		ResetOption()
		ResetTypeCache()
	}()

	Config(func(o *GlobalOption) { o.ValidatePrivateFields = true })
	ResetTypeCache() // ensure rebuild under the new option

	m := getTypeMeta(reflect.TypeOf(cacheUser{}))
	_, ok := m.byName["age"]
	assert.True(t, ok, "unexported field should be present when ValidatePrivateFields=true")
}

// recNode is self-referential; recA/recB are mutually recursive. buildTypeMeta
// walks the TYPE tree, so these must not recurse forever.
type recNode struct {
	Name string `validate:"required"`
	Next *recNode
}
type recA struct {
	Name string `validate:"required"`
	B    *recB
}
type recB struct {
	Title string `validate:"required"`
	A     *recA
}

// TestGetTypeMeta_cyclicType locks the fix for the type-cycle stack overflow:
// a recursive struct type previously worked (value-tree walk stops at nil) but
// would infinitely recurse once metadata is built from the type tree.
func TestGetTypeMeta_cyclicType(t *testing.T) {
	defer ResetTypeCache()

	t.Run("self-referential", func(t *testing.T) {
		m := getTypeMeta(reflect.TypeOf(recNode{}))
		assert.NotNil(t, m)
		// the cyclic field is recorded but marked dynamic (not statically expanded)
		fm, ok := m.byName["Next"]
		assert.True(t, ok)
		assert.Eq(t, elemStruct, fm.Elem)
	})

	t.Run("mutually-recursive", func(t *testing.T) {
		m := getTypeMeta(reflect.TypeOf(recA{}))
		assert.NotNil(t, m)
	})

	t.Run("validate path does not hang", func(t *testing.T) {
		// the regression being locked: Struct() on a recursive type must not
		// stack-overflow at metadata build time. The pre-existing library
		// behavior (preserved by P2) is that parseRulesFromTag recurses one
		// level into the nil sub-struct pointer's type and registers
		// "Next.Name" as required — so a nil Next fails. We assert that exact
		// behavior to prove the path completes and is unchanged.
		v := Struct(&recNode{Name: "a"})
		assert.False(t, v.Validate())
		assert.StrContains(t, v.Errors.One(), "Next.Name")
	})
}

func TestGetTypeMeta_concurrent(t *testing.T) {
	defer ResetTypeCache()
	ResetTypeCache()

	rt := reflect.TypeOf(cacheUser{})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := getTypeMeta(rt)
			if m == nil || m.byName["Name"] == nil {
				t.Error("concurrent getTypeMeta returned incomplete meta")
			}
		}()
	}
	wg.Wait()

	// after the storm, all callers must observe the single stored instance
	final := getTypeMeta(rt)
	assert.Same(t, final, getTypeMeta(rt))
}

// ---- P3b: static/dynamic classification + static-template caching ----

type scStaticTime struct {
	Name string    `validate:"required"`
	At   time.Time `validate:"required"`
}

type scLeafContainers struct {
	Name  string         `validate:"required"`
	Tags  []string       `validate:"required"`
	Nums  [3]int         `validate:"required"`
	Attrs map[string]int `validate:"required"`
	Ptr   *int           `validate:"required"`
}

type scNested2 struct {
	Name string `validate:"required"`
	Sub  cacheUserSub
}

func TestComputeIsStatic(t *testing.T) {
	defer ResetTypeCache()

	t.Run("static forms", func(t *testing.T) {
		assert.True(t, computeIsStatic(reflect.TypeOf(rcFlat{})))           // flat leaves
		assert.True(t, computeIsStatic(reflect.TypeOf(rcNested{})))         // non-ptr struct
		assert.True(t, computeIsStatic(reflect.TypeOf(scNested2{})))        // non-ptr struct
		assert.True(t, computeIsStatic(reflect.TypeOf(rcEmbed{})))          // exported embed (non-ptr struct)
		assert.True(t, computeIsStatic(reflect.TypeOf(scStaticTime{})))     // time.Time is a leaf
		assert.True(t, computeIsStatic(reflect.TypeOf(scLeafContainers{}))) // slice/array/map/ptr of LEAF
	})

	t.Run("dynamic forms", func(t *testing.T) {
		assert.False(t, computeIsStatic(reflect.TypeOf(rcPtrNested{})))     // *Sub
		assert.False(t, computeIsStatic(reflect.TypeOf(rcSliceOfStruct{}))) // []Sub
		assert.False(t, computeIsStatic(reflect.TypeOf(rcMapOfStruct{})))   // map[k]Sub
		assert.False(t, computeIsStatic(reflect.TypeOf(recNode{})))         // *recNode (ptr cycle)
		assert.False(t, computeIsStatic(reflect.TypeOf(recA{})))            // mutual ptr cycle
	})

	t.Run("meta carries isStatic", func(t *testing.T) {
		assert.True(t, getTypeMeta(reflect.TypeOf(rcFlat{})).isStatic)
		assert.False(t, getTypeMeta(reflect.TypeOf(rcPtrNested{})).isStatic)
	})
}

// TestStaticTemplate_concurrent stresses the lazy template build under -race:
// many goroutines validate the same STATIC type at once, all racing on the
// sync.Once-guarded staticTemplate build and on the immutable template clone.
func TestStaticTemplate_concurrent(t *testing.T) {
	defer ResetTypeCache()
	ResetTypeCache()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := Struct(&rcFlat{Name: "tom", Email: "a@b.com", Age: 5})
			_ = v.Validate()
		}()
	}
	wg.Wait()

	// the cached template must be a single shared instance after the storm.
	m := getTypeMeta(reflect.TypeOf(rcFlat{}))
	assert.NotNil(t, m.staticTemplate())
	assert.Same(t, m.staticTemplate(), m.staticTemplate())
}

// TestStaticTemplate_isolation verifies the per-instance arg-slice contract:
//   - the per-instance rules slice itself is always distinct;
//   - argsReady rules (P3a pre-converted, immutable, never mutated at runtime)
//     SHARE the template's args slice (P3b: no per-instance copy);
//   - non-argsReady rules get an INDEPENDENT copy (runtime still converts in
//     place), so one validation never corrupts another or the template.
func TestStaticTemplate_isolation(t *testing.T) {
	defer ResetTypeCache()
	ResetTypeCache()

	v1 := Struct(&rcArgSingle{A: 6, B: 5})
	v2 := Struct(&rcArgSingle{A: 6, B: 5})

	// the rules slices themselves must be distinct per instance.
	assert.NotEq(t, fmt.Sprintf("%p", v1.rules), fmt.Sprintf("%p", v2.rules))

	tpl := getTypeMeta(reflect.TypeOf(rcArgSingle{})).staticTemplate()
	for i := range v1.rules {
		r1, r2 := v1.rules[i], v2.rules[i]
		a1, a2 := r1.arguments, r2.arguments
		if len(a1) == 0 {
			continue
		}
		if r1.argsReady {
			// argsReady: both instances share the SAME immutable template args.
			assert.True(t, r1.argsReady && r2.argsReady)
			assert.Eq(t, fmt.Sprintf("%p", a1), fmt.Sprintf("%p", a2))
			assert.Eq(t, fmt.Sprintf("%p", a1), fmt.Sprintf("%p", tpl.rules[i].arguments))
		} else {
			// non-argsReady: each instance has its own backing array.
			assert.NotEq(t, fmt.Sprintf("%p", a1), fmt.Sprintf("%p", a2))
		}
	}

	// rcArgSingle uses min/max (any params): argsReady but args stay STRING; the
	// shared template args must remain the originally-collected STRING form even
	// after validating an instance (immutability check).
	_ = v1.Validate()
	for _, r := range tpl.rules {
		for _, a := range r.arguments {
			_, isStr := a.(string)
			assert.True(t, isStr)
		}
	}
}
