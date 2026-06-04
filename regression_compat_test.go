package validate

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// dumpRuleSet serializes the rule-collection result of a Validation into a
// stable, comparable string. It is the golden-regression snapshot used by P3
// to guarantee the build/instantiate refactor produces a byte-identical rule
// set, translation tables and optional/filter state.
//
// Captured dimensions:
//   - rules:     each *Rule as "path | realName | validator | nameNotRequired |
//     optional | skipEmpty | args(%#v)"; sorted for stable output.
//     NOTE: args use the Go-syntax verb %#v (not %v) so that argument
//     element TYPES are part of the snapshot. This is load-bearing for
//     P3b (rule-parse refactor): the collection phase stores rule args
//     as STRINGS (e.g. "5" not int 5), and the enum/notIn branch stores
//     a single []string element, while regexp stores one raw string.
//     %v would print string "5" and int 5 identically; %#v makes the
//     difference visible so an accidental type change is caught.
//   - filters:   each *FilterRule as "fields | filters | filterArgs"; sorted.
//   - optionals: field -> flag (v.optionals), sorted by field.
//   - labels:    v.trans.labelMap (field -> label), sorted by field.
//   - fieldMap:  v.trans.fieldMap (field -> output name), sorted by field.
//   - messages:  ONLY the entries that differ from the builtin defaults (i.e.
//     custom messages added during rule collection), sorted by key.
//     The builtin baseline (~150 entries copied by Translator.Reset)
//     is intentionally excluded so the snapshot focuses on what rule
//     collection produced.
//   - defValues: v.defValues (field -> default), sorted by field.
//
// All collections are sorted before emission; map iteration order, map-of-struct
// element path order, etc. are non-deterministic in Go, so sorting is mandatory
// to keep the snapshot reproducible across runs (verified with -count=3).
func dumpRuleSet(v *Validation) string {
	var b strings.Builder

	// --- rules ---
	ruleLines := make([]string, 0, len(v.rules))
	for _, r := range v.rules {
		// fields may contain multiple comma-joined names; join with "," for a
		// single stable path token (rule-collection from tags always yields one
		// field per rule, but be defensive).
		path := strings.Join(r.fields, ",")
		ruleLines = append(ruleLines, fmt.Sprintf(
			"%s | real=%s | validator=%s | notRequired=%v | optional=%v | skipEmpty=%v | args=%#v",
			path, r.realName, r.validator, r.nameNotRequired, r.optional, r.skipEmpty, r.arguments,
		))
	}
	sort.Strings(ruleLines)
	b.WriteString("== rules ==\n")
	for _, ln := range ruleLines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}

	// --- filter rules ---
	filterLines := make([]string, 0, len(v.filterRules))
	for _, fr := range v.filterRules {
		// filterArgs is map[int]string; emit sorted by index for stability.
		idxs := make([]int, 0, len(fr.filterArgs))
		for i := range fr.filterArgs {
			idxs = append(idxs, i)
		}
		sort.Ints(idxs)
		args := make([]string, 0, len(idxs))
		for _, i := range idxs {
			args = append(args, fmt.Sprintf("%d:%s", i, fr.filterArgs[i]))
		}
		filterLines = append(filterLines, fmt.Sprintf(
			"%s | filters=%s | args=[%s]",
			strings.Join(fr.fields, ","), strings.Join(fr.filters, ","), strings.Join(args, ","),
		))
	}
	sort.Strings(filterLines)
	b.WriteString("== filters ==\n")
	for _, ln := range filterLines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}

	// --- optionals ---
	b.WriteString("== optionals ==\n")
	for _, line := range sortedMapInt8(v.optionals) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// --- labels (trans.labelMap) ---
	b.WriteString("== labels ==\n")
	for _, line := range sortedMapStr(v.trans.labelMap) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// --- field output map (trans.fieldMap) ---
	b.WriteString("== fieldMap ==\n")
	for _, line := range sortedMapStr(v.trans.fieldMap) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// --- custom messages (trans.messages minus builtin defaults) ---
	b.WriteString("== messages ==\n")
	custom := make(map[string]string)
	for k, val := range v.trans.messages {
		if base, ok := builtinMessages[k]; !ok || base != val {
			custom[k] = val
		}
	}
	for _, line := range sortedMapStr(custom) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	// --- default values ---
	b.WriteString("== defValues ==\n")
	if len(v.defValues) > 0 {
		keys := make([]string, 0, len(v.defValues))
		for k := range v.defValues {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("%s=%v\n", k, v.defValues[k]))
		}
	}

	return b.String()
}

func sortedMapStr(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return out
}

func sortedMapInt8(m map[string]int8) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return out
}

// ---- struct fixtures used by the golden cases ----

type rcFlat struct {
	Name  string `validate:"required|minLen:3"`
	Email string `validate:"email"`
	Age   int    `validate:"int|min:1|max:120"`
}

type rcInner struct {
	City string `validate:"required"`
	Zip  string `validate:"required|minLen:3"`
}

type rcNested struct {
	Name string  `validate:"required"`
	Addr rcInner `validate:"required"`
}

type rcPtrNested struct {
	Name string   `validate:"required"`
	Sub  *rcInner `validate:"required"`
}

type rcSliceOfStruct struct {
	Name  string    `validate:"required"`
	Items []rcInner `validate:"required"`
}

type rcMapOfStruct struct {
	Name  string             `validate:"required"`
	Items map[string]rcInner `validate:"required"`
}

// RcEmbedBase is intentionally EXPORTED so the embedded field name starts with
// an uppercase letter and is NOT skipped as unexported. This exercises the real
// anonymous-embed recursion path in parseRulesFromTag.
type RcEmbedBase struct {
	BaseID int `validate:"required|min:1"`
}

type rcEmbed struct {
	RcEmbedBase `validate:""`
	Name        string `validate:"required"`
}

// rcEmbedUnexported embeds an UNEXPORTED-named type; its embed field name starts
// lowercase, so (with ValidatePrivateFields=false) the whole embed and its inner
// fields are skipped. Captures the contrasting v1 behavior.
type rcEmbedBaseLower struct {
	BaseID int `validate:"required|min:1"`
}

type rcEmbedUnexported struct {
	rcEmbedBaseLower
	Name string `validate:"required"`
}

type rcPrivate struct {
	Name string `validate:"required"`
	age  int    `validate:"required|min:1"`
}

type rcFilter struct {
	Name string `validate:"required" filter:"trim|lower"`
}

type rcMsgSingle struct {
	Name string `validate:"required" message:"name is required"`
}

type rcMsgNamed struct {
	Name string `validate:"required|minLen:3" message:"required:name must not be empty"`
}

type rcMsgMulti struct {
	Name string `validate:"required|minLen:3" message:"required:name is required|minLen:name min len is %d"`
}

type rcLabelJSON struct {
	Name string `validate:"required" label:"用户名" json:"user_name"`
}

type rcDefault struct {
	Name string `validate:"default:foo"`
}

type rcSafe struct {
	Name string `validate:"required"`
	Tmp  string `validate:"safe"`
	Skip string `validate:"-"`
}

type rcScene struct {
	Name  string `validate:"required"`
	Email string `validate:"email"`
	Age   int    `validate:"int|min:1"`
}

// ---- fixtures for the parameter-parsing (arg-form) golden cases ----
//
// These pin the EXACT shape of *Rule.arguments produced by the collection phase
// (StringRule's `switch realName`). P3b will move rule parsing to a type-level
// cache; these snapshots guarantee the arg shapes are reproduced byte-for-byte.

// single scalar args: collected as STRING "5"/"10" (not int) — the default
// (comma-split) branch runs them through parseArgString+strings2Args.
type rcArgSingle struct {
	A int `validate:"min:5"`
	B int `validate:"max:10"`
}

// two scalar args via between + its alias range; default branch comma-splits
// into two STRING args.
type rcArgBetween struct {
	A int `validate:"between:1,10"`
	B int `validate:"range:1,10"`
}

// enum / its alias in: realName=="enum" → args MERGED into a single []string
// element (one arg whose value is the []string slice).
type rcArgEnum struct {
	A string `validate:"enum:a,b,c"`
	B int    `validate:"in:1,2,3"`
}

// notIn / its alias not_in: realName=="notIn" → same merge-into-one-[]string.
type rcArgNotIn struct {
	A string `validate:"notIn:x,y,z"`
	B string `validate:"not_in:p,q"`
}

// regexp / its alias regex: realName=="regexp" → args NOT comma-split; the whole
// pattern is one raw STRING arg. Use raw-string (backtick) tags so the backslash
// is literal.
type rcArgRegexp struct {
	A string `validate:"regexp:\\d{4,6}"`
	B string `validate:"regex:^a,b$"`
}

// length / minLen / maxLen: single int param collected as STRING via default
// branch, then PRE-CONVERTED to int by the STATIC template build (P3a), since
// the validators take a concrete `int` arg. The golden args are typed ints.
type rcArgLen struct {
	A string `validate:"length:6"`
	B string `validate:"minLen:3"`
	C string `validate:"maxLen:20"`
}

// requiredIf: requiredXX validator → nameNotRequired=false; multi string args via
// default branch (comma-split into two STRINGs).
type rcArgRequiredIf struct {
	Name string `validate:"requiredIf:other,value"`
}

// requiredWith: variadic validator; default branch comma-splits into N STRING args.
type rcArgRequiredWith struct {
	Name string `validate:"requiredWith:a,b,c"`
}

// mixed pipeline on one field: exercises every arg form coexisting on one field.
type rcArgMixed struct {
	Age int `validate:"required|int|min:1|max:120|enum:1,2,3"`
}

func TestRuleCompat_golden(t *testing.T) {
	t.Run("flat", func(t *testing.T) {
		v := Struct(&rcFlat{})
		want := `== rules ==
Age | real=isInt | validator=int | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Age | real=max | validator=max | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"120"}
Age | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1"}
Email | real=isEmail | validator=email | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Name | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{3}
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("nested", func(t *testing.T) {
		v := Struct(&rcNested{})
		want := `== rules ==
Addr | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Addr.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Addr.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{3}
Addr.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("ptr_nested_nil", func(t *testing.T) {
		v := Struct(&rcPtrNested{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Sub.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("ptr_nested_nonnil", func(t *testing.T) {
		v := Struct(&rcPtrNested{Sub: &rcInner{}})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Sub.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Sub.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("slice_of_struct", func(t *testing.T) {
		v := Struct(&rcSliceOfStruct{Items: []rcInner{{}, {}, {}}})
		want := `== rules ==
Items | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.0.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.0.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Items.0.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.1.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.1.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Items.1.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.2.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.2.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Items.2.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("map_of_struct", func(t *testing.T) {
		v := Struct(&rcMapOfStruct{Items: map[string]rcInner{"home": {}, "work": {}}})
		want := `== rules ==
Items | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.home.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.home.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Items.home.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.work.City | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Items.work.Zip | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"3"}
Items.work.Zip | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("embed_anonymous", func(t *testing.T) {
		v := Struct(&rcEmbed{})
		// NOTE: an anonymous embed of an EXPORTED-named type recurses and the
		// embedded field rules are collected under a dotted path
		// "RcEmbedBase.BaseID" (NOT flattened to "BaseID"). Captures v1 behavior.
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
RcEmbedBase.BaseID | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1"}
RcEmbedBase.BaseID | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("embed_anonymous_unexported", func(t *testing.T) {
		v := Struct(&rcEmbedUnexported{})
		// NOTE: an embed whose field name starts lowercase is skipped as
		// unexported (ValidatePrivateFields=false), so the embedded BaseID rule
		// is NOT collected. Contrasts with the exported-embed case above.
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("private_fields", func(t *testing.T) {
		defer func() {
			ResetOption()
			ResetTypeCache()
		}()
		Config(func(opt *GlobalOption) { opt.ValidatePrivateFields = true })
		v := Struct(&rcPrivate{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
age | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1"}
age | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("filter", func(t *testing.T) {
		v := Struct(&rcFilter{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
Name | filters=trim,lower | args=[]
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("message_single", func(t *testing.T) {
		v := Struct(&rcMsgSingle{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
Name.required=name is required
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("message_named", func(t *testing.T) {
		v := Struct(&rcMsgNamed{})
		want := `== rules ==
Name | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{3}
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
Name.required=name must not be empty
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("message_multi", func(t *testing.T) {
		v := Struct(&rcMsgMulti{})
		want := `== rules ==
Name | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{3}
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
Name.minLen=name min len is %d
Name.required=name is required
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("label_json", func(t *testing.T) {
		v := Struct(&rcLabelJSON{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
Name=用户名
== fieldMap ==
Name=user_name
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("default", func(t *testing.T) {
		v := Struct(&rcDefault{})
		// "default:foo" registers a default value (not a rule); no rule is added.
		want := `== rules ==
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
Name=foo
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("safe_and_dash", func(t *testing.T) {
		v := Struct(&rcSafe{})
		// "safe" and "-" are collected as ordinary rules (validator name = literal
		// "safe" / "-"); their semantics (skip / no-op) are handled at validate time.
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
Skip | real=- | validator=- | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Tmp | real=safe | validator=safe | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("scene", func(t *testing.T) {
		v := Struct(&rcScene{}).WithScenes(SValues{
			"create": []string{"Name", "Email"},
			"update": []string{"Name"},
		}).AtScene("create")
		// scene does not change the collected rule set itself; it only filters at
		// validate time. Capture the full collected rules + scene name.
		want := `== rules ==
Age | real=isInt | validator=int | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Age | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1"}
Email | real=isEmail | validator=email | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
scene=create
`
		assert.Eq(t, want, dumpRuleSet(v)+"scene="+v.Scene()+"\n")
	})

	t.Run("check_sub_on_parent_marked", func(t *testing.T) {
		defer func() {
			ResetOption()
			ResetTypeCache()
		}()
		Config(func(opt *GlobalOption) { opt.CheckSubOnParentMarked = true })
		// Addr has NO validate rule on the parent field, so with
		// CheckSubOnParentMarked=true the sub-struct rules must NOT be collected.
		type parent struct {
			Name string  `validate:"required"`
			Addr rcInner // no rule on parent field
		}
		v := Struct(&parent{})
		want := `== rules ==
Name | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})
}

// TestRuleCompat_argForms pins the exact shape of *Rule.arguments produced by the
// collection phase for every special / general parameter form handled by
// StringRule's `switch realName`. Golden args use %#v so element TYPES are part of
// the snapshot (see dumpRuleSet doc). This is the most P3b-sensitive snapshot.
func TestRuleCompat_argForms(t *testing.T) {
	t.Run("single_scalar", func(t *testing.T) {
		// min:5 / max:10 — default branch: arg collected as STRING "5"/"10", NOT int.
		v := Struct(&rcArgSingle{})
		want := `== rules ==
A | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"5"}
B | real=max | validator=max | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"10"}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("between_and_range_alias", func(t *testing.T) {
		// between:1,10 and alias range:1,10 — default branch: two STRING args.
		// validator keeps the input name (between / range); realName resolves both
		// to "between".
		v := Struct(&rcArgBetween{})
		want := `== rules ==
A | real=between | validator=between | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1", "10"}
B | real=between | validator=range | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1", "10"}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("enum_and_in_alias_merged", func(t *testing.T) {
		// enum:a,b,c / in:1,2,3 — realName=="enum": args MERGED into ONE element
		// whose value is a []string slice.
		v := Struct(&rcArgEnum{})
		want := `== rules ==
A | real=enum | validator=enum | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{[]string{"a", "b", "c"}}
B | real=enum | validator=in | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{[]string{"1", "2", "3"}}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("notIn_and_not_in_alias_merged", func(t *testing.T) {
		// notIn:x,y,z / not_in:p,q — realName=="notIn": same single []string merge.
		v := Struct(&rcArgNotIn{})
		want := `== rules ==
A | real=notIn | validator=notIn | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{[]string{"x", "y", "z"}}
B | real=notIn | validator=not_in | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{[]string{"p", "q"}}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("regexp_and_regex_alias_nosplit", func(t *testing.T) {
		// regexp / regex — realName=="regexp": args NOT comma-split; the WHOLE
		// pattern is one raw STRING arg (note B's pattern contains a comma which is
		// preserved verbatim).
		v := Struct(&rcArgRegexp{})
		want := `== rules ==
A | real=regexp | validator=regexp | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"\\d{4,6}"}
B | real=regexp | validator=regex | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"^a,b$"}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("length_variants", func(t *testing.T) {
		// length:6 / minLen:3 / maxLen:20 — default branch collects STRING args,
		// then the STATIC template build pre-converts them to typed int (P3a).
		v := Struct(&rcArgLen{})
		want := `== rules ==
A | real=length | validator=length | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{6}
B | real=minLength | validator=minLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{3}
C | real=maxLength | validator=maxLen | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{20}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("requiredIf_multi_string_args", func(t *testing.T) {
		// requiredIf:other,value — requiredXX → nameNotRequired=false; default branch
		// comma-splits into two STRING args.
		v := Struct(&rcArgRequiredIf{})
		want := `== rules ==
Name | real=requiredIf | validator=requiredIf | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}{"other", "value"}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("requiredWith_variadic", func(t *testing.T) {
		// requiredWith:a,b,c — variadic validator; default branch: N STRING args.
		v := Struct(&rcArgRequiredWith{})
		want := `== rules ==
Name | real=requiredWith | validator=requiredWith | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}{"a", "b", "c"}
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})

	t.Run("mixed_pipeline_one_field", func(t *testing.T) {
		// required|int|min:1|max:120|enum:1,2,3 on one field — verifies every arg
		// form coexists correctly under a single field path.
		v := Struct(&rcArgMixed{})
		want := `== rules ==
Age | real=enum | validator=enum | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{[]string{"1", "2", "3"}}
Age | real=isInt | validator=int | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}(nil)
Age | real=max | validator=max | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"120"}
Age | real=min | validator=min | notRequired=true | optional=false | skipEmpty=true | args=[]interface {}{"1"}
Age | real=required | validator=required | notRequired=false | optional=false | skipEmpty=true | args=[]interface {}(nil)
== filters ==
== optionals ==
== labels ==
== fieldMap ==
== messages ==
== defValues ==
`
		assert.Eq(t, want, dumpRuleSet(v))
	})
}
