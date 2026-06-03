package validate

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// elemClass classifies a field's (de-pointered) element kind for the purpose
// of static-vs-dynamic rule expansion in later phases.
//
//   - elemLeaf:          plain leaf field (string/int/bool/...), no recursion
//   - elemStruct:        struct field (statically expandable, recursed at build)
//   - elemSliceOfStruct: []Struct / [N]Struct — needs per-value expansion (P3 dynamic)
//   - elemMapOfStruct:   map[K]Struct — needs per-value expansion (P3 dynamic)
//   - elemOther:         slice/map/array of non-struct, or anything else
type elemClass uint8

const (
	elemLeaf elemClass = iota
	elemStruct
	elemSliceOfStruct
	elemMapOfStruct
	elemOther
)

// fieldMeta holds the type-level (value-independent) metadata of a single
// struct field. Built once per type and shared read-only afterwards.
type fieldMeta struct {
	// Index is the full path for reflect.Value.FieldByIndex, O(1) access.
	// For a top-level direct field this is a single-element slice {i}; for a
	// nested field it is the parent chain appended with the field index.
	Index []int
	// Name is the plain field name (last segment).
	Name string
	// Path is the dotted path, e.g. "Parent.Child"; for a top-level field it
	// equals Name.
	Path string
	// Kind is the de-pointered reflect.Kind of the field type.
	Kind reflect.Kind
	// IsPtr reports whether the declared field type is a pointer.
	IsPtr bool
	// Elem classifies the field for static/dynamic rule handling.
	Elem elemClass

	// Type-level tag strings, read once at build time via Tag.Get. Consumed by
	// P3 (rule template pre-parsing); P2 only populates them.
	ValidateRule string
	FilterRule   string
	OutputName   string
	Label        string
	MessageRaw   string
}

// typeMeta holds all type-level metadata for one struct type. It is built once
// and, after being stored into the cache, is treated as immutable (read-only),
// so it is safe to share across goroutines.
type typeMeta struct {
	Type reflect.Type
	// Fields are the directly-built field metas (top-level + statically
	// recursed sub-struct fields), in build order.
	Fields []*fieldMeta
	// byName maps a field Path -> its meta, O(1) lookup. key = fieldMeta.Path.
	byName map[string]*fieldMeta

	// dynamicFields collects fields needing per-value expansion
	// (slice/map-of-struct). Reserved for P3; populated as a marker only.
	dynamicFields []*fieldMeta

	// isStatic reports whether this type's collected rule set is fully
	// value-independent (no ptr-to-struct, slice-of-struct or map-of-struct in
	// its field graph). STATIC types use a cached rule template (instantiateStatic);
	// DYNAMIC types keep the original per-value parse path (parseRulesFromTag).
	isStatic bool
	// tpl is the lazily-built, immutable rule template for a STATIC type. Guarded
	// by tplOnce so concurrent first-time validators build it exactly once.
	tplOnce sync.Once
	tpl     *ruleTemplate

	// One-shot Implements results, computed at build time so each instance does
	// not pay three reflect Implements calls in StructData.Create.
	implConfig     bool // implements ConfigValidationFace
	implTranslates bool // implements FieldTranslatorFace
	implMessages   bool // implements CustomMessagesFace
}

// typeKey is the cache key. tagVer is folded in so that a global tag-name
// change (via Config/ResetOption) naturally invalidates all previously cached
// metas without clearing the map.
type typeKey struct {
	rt     reflect.Type
	tagVer uint32
}

// typeCache caches *typeMeta keyed by typeKey. Read-mostly; occasional
// duplicate builds under races are acceptable since the stored value is
// immutable.
var typeCache sync.Map // map[typeKey]*typeMeta

// tagVer is the tag-config version. Bumped when the global tag option names
// change, so cached metas built with the old tag names become unreachable.
var tagVer uint32

// getTypeMeta returns the cached *typeMeta for the given struct type, building
// and storing it on first miss. rt must be a struct type (FromStruct already
// passes the de-pointered elem type).
func getTypeMeta(rt reflect.Type) *typeMeta {
	key := typeKey{rt: rt, tagVer: atomic.LoadUint32(&tagVer)}
	if v, ok := typeCache.Load(key); ok {
		return v.(*typeMeta)
	}

	tm := buildTypeMeta(rt)
	// LoadOrStore guards against a concurrent builder having stored first; in
	// that case we drop ours and use the already-stored (equivalent) one.
	actual, _ := typeCache.LoadOrStore(key, tm)
	return actual.(*typeMeta)
}

// ResetTypeCache clears the whole type metadata cache. Intended for tests and
// special cases where forcing a rebuild is desired. Implemented via Range+Delete
// (instead of swapping the sync.Map pointer) to stay concurrency-safe.
func ResetTypeCache() {
	typeCache.Range(func(key, _ any) bool {
		typeCache.Delete(key)
		return true
	})
}

// buildTypeMeta does a pure type traversal of the struct type rt — it never
// touches any concrete value. It recurses into struct-of-struct (skipping
// time.Time and, unless gOpt.ValidatePrivateFields, unexported fields), and
// only marks slice/map-of-struct fields (no element expansion).
func buildTypeMeta(rt reflect.Type) *typeMeta {
	tm := &typeMeta{
		Type:   rt,
		byName: make(map[string]*fieldMeta),
	}

	// ancestors tracks struct types currently on the recursion path, to break
	// type cycles (e.g. `type Node struct{ Next *Node }`). buildTypeMeta walks
	// the TYPE tree (not the value tree), so without this guard a self- or
	// mutually-recursive struct type would recurse forever. A type appearing in
	// sibling branches (not as an ancestor) is still fully expanded.
	var walk func(t reflect.Type, parentPath string, parentIndex []int, ancestors map[reflect.Type]bool)
	walk = func(t reflect.Type, parentPath string, parentIndex []int, ancestors map[reflect.Type]bool) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			name := sf.Name

			// skip unexported fields unless explicitly enabled (mirrors
			// parseRulesFromTag: data_source.go).
			if name[0] >= 'a' && name[0] <= 'z' {
				if !gOpt.ValidatePrivateFields {
					continue
				}
			}

			path := name
			if parentPath != "" {
				path = parentPath + "." + name
			}

			// build full FieldByIndex path: parent chain + this field index.
			idx := make([]int, len(parentIndex)+1)
			copy(idx, parentIndex)
			idx[len(parentIndex)] = i

			ft := removeTypePtr(sf.Type)
			fm := &fieldMeta{
				Index: idx,
				Name:  name,
				Path:  path,
				Kind:  ft.Kind(),
				IsPtr: sf.Type.Kind() == reflect.Ptr,
			}

			// read the five type-level tags once.
			if gOpt.ValidateTag != "" {
				fm.ValidateRule = sf.Tag.Get(gOpt.ValidateTag)
			}
			if gOpt.FilterTag != "" {
				fm.FilterRule = sf.Tag.Get(gOpt.FilterTag)
			}
			if gOpt.FieldTag != "" {
				fm.OutputName = sf.Tag.Get(gOpt.FieldTag)
			}
			if gOpt.LabelTag != "" {
				fm.Label = sf.Tag.Get(gOpt.LabelTag)
			}
			if gOpt.MessageTag != "" {
				fm.MessageRaw = sf.Tag.Get(gOpt.MessageTag)
			}

			// classify element kind and recurse statically for struct-of-struct.
			switch ft.Kind() {
			case reflect.Struct:
				if ft == timeType {
					fm.Elem = elemLeaf
				} else {
					fm.Elem = elemStruct
				}
			case reflect.Array, reflect.Slice:
				if removeTypePtr(ft.Elem()).Kind() == reflect.Struct && removeTypePtr(ft.Elem()) != timeType {
					fm.Elem = elemSliceOfStruct
				} else {
					fm.Elem = elemOther
				}
			case reflect.Map:
				if removeTypePtr(ft.Elem()).Kind() == reflect.Struct && removeTypePtr(ft.Elem()) != timeType {
					fm.Elem = elemMapOfStruct
				} else {
					fm.Elem = elemOther
				}
			default:
				fm.Elem = elemLeaf
			}

			tm.Fields = append(tm.Fields, fm)
			tm.byName[path] = fm

			switch fm.Elem {
			case elemStruct:
				if ancestors[ft] {
					// type cycle: depth is value-dependent, so this field cannot
					// be statically expanded. Mark it dynamic (P3 walks it per
					// value, terminating naturally at nil) and stop descending.
					tm.dynamicFields = append(tm.dynamicFields, fm)
				} else {
					ancestors[ft] = true
					walk(ft, path, idx, ancestors)
					delete(ancestors, ft)
				}
			case elemSliceOfStruct, elemMapOfStruct:
				// mark for P3 per-value expansion; do NOT expand elements here.
				tm.dynamicFields = append(tm.dynamicFields, fm)
			}
		}
	}

	walk(rt, "", nil, map[reflect.Type]bool{rt: true})

	// one-shot interface checks. Match the previous per-instance behavior in
	// StructData.Create EXACTLY: it called d.valueTyp.Implements(...), where
	// d.valueTyp is the de-pointered (value) struct type — i.e. only the value
	// method set is considered, never the pointer method set. Keep it identical.
	tm.implConfig = rt.Implements(cvFaceType)
	tm.implTranslates = rt.Implements(ftFaceType)
	tm.implMessages = rt.Implements(cmFaceType)

	// classify static vs dynamic for rule-template caching (P3b). Computed via a
	// dedicated type scan (see computeIsStatic) so the criteria are explicit and
	// independent of the dynamicFields markers above.
	tm.isStatic = computeIsStatic(rt)

	return tm
}
