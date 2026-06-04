package validate

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gookit/goutil/reflects"
)

/*************************************************************
 * Static rule template (P3b): build once per STATIC type and
 * clone into each Validation, avoiding the per-value tag walk.
 *************************************************************/

// ruleTemplate is the immutable, value-independent rule-collection snapshot of
// a STATIC struct type. It is produced once (lazily) by running the existing
// parseRulesFromTag against a fresh zero-value instance, then captured here.
//
// Because a STATIC type's rule set does not depend on any concrete value (it
// has no ptr-to-struct, slice-of-struct or map-of-struct fields), the snapshot
// taken from a zero value is byte-for-byte identical to what any value would
// produce. The snapshot is read-only after build and shared across goroutines.
type ruleTemplate struct {
	rules       []*Rule
	filterRules []*FilterRule
	optionals   map[string]int8
	defValues   map[string]any
	fieldNames  map[string]int8

	// translation tables (relative to a fresh Translator):
	labelMap map[string]string // trans.labelMap
	fieldMap map[string]string // trans.fieldMap (output names)
	messages map[string]string // ONLY custom messages added during collection
}

// computeIsStatic reports whether rt's rule set is value-independent.
//
// A struct type is DYNAMIC iff anywhere in its field graph (recursing ONLY into
// non-pointer struct fields) there is a field that is:
//   - a pointer to a struct (non time.Time), OR
//   - a slice/array whose element is a struct (non time.Time), OR
//   - a map whose value is a struct (non time.Time).
//
// Otherwise it is STATIC. Recursion only descends through non-pointer struct
// fields; Go forbids a non-pointer struct from (transitively) containing itself
// without a pointer, so this terminates. Any cycle necessarily goes through a
// pointer-to-struct, which is caught by the first rule above (DYNAMIC) before
// recursing. The ancestors guard is a defensive backstop.
func computeIsStatic(rt reflect.Type) bool {
	var scan func(t reflect.Type, ancestors map[reflect.Type]bool) bool
	scan = func(t reflect.Type, ancestors map[reflect.Type]bool) bool {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			name := sf.Name
			// mirror parseRulesFromTag's unexported-field skip.
			if name[0] >= 'a' && name[0] <= 'z' {
				if !gOpt.ValidatePrivateFields {
					continue
				}
			}

			st := sf.Type
			switch st.Kind() {
			case reflect.Ptr:
				// ptr-to-struct (non time.Time): nil vs non-nil changes whether
				// sub-rules are collected -> DYNAMIC.
				et := removeTypePtr(st)
				if et.Kind() == reflect.Struct && et != timeType {
					return false
				}
			case reflect.Struct:
				if st == timeType {
					continue
				}
				if ancestors[st] {
					// only reachable via a non-ptr struct cycle, which Go
					// disallows; treat defensively as dynamic.
					return false
				}
				ancestors[st] = true
				if !scan(st, ancestors) {
					return false
				}
				delete(ancestors, st)
			case reflect.Array, reflect.Slice:
				et := removeTypePtr(st.Elem())
				if et.Kind() == reflect.Struct && et != timeType {
					return false
				}
			case reflect.Map:
				et := removeTypePtr(st.Elem())
				if et.Kind() == reflect.Struct && et != timeType {
					return false
				}
			default:
				// leaf field: no effect on static-ness.
			}
		}
		return true
	}
	return scan(rt, map[reflect.Type]bool{rt: true})
}

// staticTemplate returns the cached rule template for a STATIC type, building
// it once via sync.Once. Safe for concurrent callers (multiple goroutines
// validating the same type).
func (m *typeMeta) staticTemplate() *ruleTemplate {
	m.tplOnce.Do(func() {
		m.tpl = buildRuleTemplate(m.Type)
	})
	return m.tpl
}

// buildRuleTemplate constructs the immutable rule snapshot for a STATIC struct
// type by reusing the existing parseRulesFromTag over a fresh zero-value
// instance. This guarantees the snapshot equals the live result byte-for-byte
// (same parsing code path), while only paying the cost once per type.
func buildRuleTemplate(rt reflect.Type) *ruleTemplate {
	// temporary zero-value StructData + empty Validation to run the real parser.
	zero := reflect.New(rt).Elem()
	td := &StructData{
		src:         zero.Interface(),
		value:       zero,
		valueTyp:    rt,
		ValidateTag: gOpt.ValidateTag,
		FilterTag:   gOpt.FilterTag,
		fieldNames:  make(map[string]int8),
		fieldValues: make(map[string]reflect.Value),
	}
	tv := newEmpty()
	tv.data = td

	td.parseRulesFromTag(tv)

	tpl := &ruleTemplate{
		rules:       tv.rules,
		filterRules: tv.filterRules,
		optionals:   tv.optionals,
		defValues:   tv.defValues,
		fieldNames:  td.fieldNames,
		labelMap:    tv.trans.labelMap,
		fieldMap:    tv.trans.fieldMap,
	}

	// keep only custom messages (those differing from the builtin defaults),
	// matching the dimension captured by the golden regression snapshot.
	if len(tv.trans.messages) > 0 {
		custom := make(map[string]string)
		for k, val := range tv.trans.messages {
			if base, ok := builtinMessages[k]; !ok || base != val {
				custom[k] = val
			}
		}
		if len(custom) > 0 {
			tpl.messages = custom
		}
	}

	// P3a: pre-convert each rule's string args to the validator-signature types
	// once per STATIC type. This moves the per-validate convertArgsType cost to
	// build time. tv reuses the same validatorMeta lookup the runtime path uses.
	preConvertTemplateArgs(tpl.rules, tv)

	return tpl
}

// preConvertTemplateArgs walks the static template rules and, for each rule
// backed by a BUILTIN validator, converts its string args to the validator's
// signature types via convertRuleArgs. On success the rule is marked argsReady
// so valueValidate skips the runtime conversion; on failure (or for
// non-builtin / unknown validators) the args are left untouched for the runtime
// path to handle. Only builtin validators are pre-converted so a later
// AddValidator override (different signature) can never use a stale typed arg.
func preConvertTemplateArgs(rules []*Rule, tv *Validation) {
	for _, r := range rules {
		// resolve funcMeta exactly as the runtime does (valueValidate uses the
		// rule's realName as the validator name; template rules have no
		// checkFuncMeta, so this falls through to validatorMeta lookup).
		name := r.realName
		fm := r.checkFuncMeta
		if fm == nil {
			fm = tv.validatorMeta(name)
		}
		// only pre-convert builtin validators (design §4.6).
		if fm == nil || !fm.builtin {
			continue
		}

		// compute addNum the same way valueValidate does: +1 for the value arg,
		// +1 more when the validator's first arg is DataFace.
		ft := fm.fv.Type()
		addNum := 1
		if ft.In(0) == dataFaceType {
			addNum++
		}

		// convertRuleArgs converts in place; only mark argsReady on success so a
		// failed conversion stays string and is retried (and reported) at runtime.
		if err := convertRuleArgs(fm, "", r.arguments, addNum); err == nil {
			r.argsReady = true
		}
	}
}

// instantiateStatic clones the cached STATIC template into the real Validation
// v and StructData d, producing a rule set identical to what parseRulesFromTag
// would have built for this value — but with zero reflection over the value.
func (d *StructData) instantiateStatic(v *Validation) {
	if d.ValidateTag == "" {
		d.ValidateTag = gOpt.ValidateTag
	}
	if d.FilterTag == "" {
		d.FilterTag = gOpt.FilterTag
	}

	tpl := d.meta.staticTemplate()

	// --- rules: clone each rule with its OWN args slice ---
	// convertArgsType (validating.go) mutates r.arguments in place at validate
	// time (string->typed). Sharing the template's args slice would corrupt the
	// template / race across instances, so each instance gets a fresh copy.
	if len(tpl.rules) > 0 {
		// keep this instance's OWN backing array (cap=len) so Reset/复用 + AddRule
		// re-append never overwrites the shared template backing array.
		v.rules = make([]*Rule, 0, len(tpl.rules))
		for _, tr := range tpl.rules {
			if tr.argsReady {
				// argsReady 模板规则在校验期完全只读(valueValidate 跳过 convertArgsType,
				// 全仓库无任何校验期对 Rule 字段的写入),直接共享模板 *Rule 指针,免每实例
				// cloneRule 分配;跨实例/跨 goroutine 共享安全。
				v.rules = append(v.rules, tr)
			} else {
				// 非 argsReady 规则运行期仍会原地转换 args,每实例必须独立拷贝。
				v.rules = append(v.rules, cloneRule(tr))
			}
		}
	}

	// --- filter rules: clone (filters slice + filterArgs map copied) ---
	if len(tpl.filterRules) > 0 {
		v.filterRules = make([]*FilterRule, 0, len(tpl.filterRules))
		for _, tfr := range tpl.filterRules {
			v.filterRules = append(v.filterRules, cloneFilterRule(tfr))
		}
	}

	// --- optionals ---
	for k, val := range tpl.optionals {
		v.optionals[k] = val
	}

	// --- default values ---
	for k, val := range tpl.defValues {
		v.SetDefValue(k, val)
	}

	// --- field names (TryGet/Set rely on these) ---
	for k, val := range tpl.fieldNames {
		d.fieldNames[k] = val
	}

	// --- translation tables: replay via the public helpers ---
	for field, label := range tpl.labelMap {
		v.trans.addLabelName(field, label)
	}
	if len(tpl.fieldMap) > 0 {
		v.trans.AddFieldMap(tpl.fieldMap)
	}
	for key, msg := range tpl.messages {
		v.trans.AddMessage(key, msg)
	}
}

// cloneRule makes a shallow copy of an immutable template Rule.
//
// For an argsReady rule (P3a pre-converted), its args are already typed AND the
// runtime no longer mutates them (valueValidate skips convertArgsType), so the
// template's args slice is immutable and can be SHARED directly — saving a
// per-instance copy. For a non-argsReady rule the runtime still converts args in
// place, so each instance needs its own copy. nil args stay nil in both cases.
func cloneRule(tr *Rule) *Rule {
	r := *tr // shallow copy of all scalar/ref fields
	if tr.arguments != nil && !tr.argsReady {
		args := make([]any, len(tr.arguments))
		copy(args, tr.arguments)
		r.arguments = args
	} // else: argsReady (share immutable template args) OR nil (keep nil)
	return &r
}

// cloneFilterRule deep-copies a template FilterRule's mutable-shaped fields
// (filters slice + filterArgs map) so per-instance use never touches the
// shared template.
func cloneFilterRule(tfr *FilterRule) *FilterRule {
	fr := &FilterRule{}
	if tfr.fields != nil {
		fr.fields = make([]string, len(tfr.fields))
		copy(fr.fields, tfr.fields)
	}
	if tfr.filters != nil {
		fr.filters = make([]string, len(tfr.filters))
		copy(fr.filters, tfr.filters)
	}
	fr.filterArgs = make(map[int]string, len(tfr.filterArgs))
	for i, a := range tfr.filterArgs {
		fr.filterArgs[i] = a
	}
	return fr
}

// parse and collect rules from struct tags.
func (d *StructData) parseRulesFromTag(v *Validation) {
	if d.ValidateTag == "" {
		d.ValidateTag = gOpt.ValidateTag
	}

	if d.FilterTag == "" {
		d.FilterTag = gOpt.FilterTag
	}

	fOutMap := make(map[string]string)
	var recursiveFunc func(vv reflect.Value, vt reflect.Type, preStrName string, parentIsAnonymous bool)

	vv := d.value
	vt := d.valueTyp
	// preStrName - the parent field name.
	recursiveFunc = func(vv reflect.Value, vt reflect.Type, parentFName string, parentIsAnonymous bool) {
		for i := 0; i < vt.NumField(); i++ {
			fv := vt.Field(i)
			// skip don't exported field
			name := fv.Name
			if name[0] >= 'a' && name[0] <= 'z' {
				if !gOpt.ValidatePrivateFields {
					continue
				}
			}

			if parentFName == "" {
				d.fieldNames[name] = fieldAtTopStruct
			} else {
				name = parentFName + "." + name
				if parentIsAnonymous {
					d.fieldNames[name] = fieldAtAnonymous
				} else {
					d.fieldNames[name] = fieldAtSubStruct
				}
			}

			// validate rule
			vRule := fv.Tag.Get(d.ValidateTag)
			if vRule != "" {
				v.StringRule(name, vRule)
			}

			// filter rule
			fRule := fv.Tag.Get(d.FilterTag)
			if fRule != "" {
				v.FilterRule(name, fRule)
			}

			// load field output name by FieldTag. eg: `json:"user_name"`
			outName := ""
			if gOpt.FieldTag != "" {
				outName = fv.Tag.Get(gOpt.FieldTag)
				outName = strings.SplitN(outName, ",", 2)[0]
			}

			// add pre field display name to fName
			if outName != "" {
				if parentFName != "" {
					if pOutName, ok := fOutMap[parentFName]; ok {
						outName = pOutName + "." + outName
					}
				}

				fOutMap[name] = outName
			}

			// load field translate name
			// preferred to use label tag name. eg: `label:"display name"`
			// and then use field output name. eg: `json:"user_name"`
			if gOpt.LabelTag != "" {
				v.trans.addLabelName(name, fv.Tag.Get(gOpt.LabelTag))
			}

			// load custom error messages.
			// eg: `message:"required:name is required|minLen:name min len is %d"`
			if gOpt.MessageTag != "" {
				errMsg := fv.Tag.Get(gOpt.MessageTag)
				if errMsg != "" {
					d.loadMessagesFromTag(v.trans, name, vRule, errMsg)
				}
			}

			ft := removeTypePtr(vt.Field(i).Type)

			// collect rules from sub-struct and from arrays/slices elements
			if ft != timeType && removeValuePtr(vv).IsValid() {

				// feat: only collect sub-struct rule on current field has rule.
				if vRule == "" && gOpt.CheckSubOnParentMarked {
					continue
				}

				fValue := removeValuePtr(vv).Field(i)

				switch ft.Kind() {
				case reflect.Struct:
					recursiveFunc(fValue, ft, name, fv.Anonymous)

				case reflect.Array, reflect.Slice:
					fValue = removeValuePtr(fValue)

					// Check if the reflect.Value is valid and not a nil pointer
					if !fValue.IsValid() || (ft.Kind() == reflect.Slice && fValue.IsNil()) {
						continue
					}
					// perf: skip parse on elements is simple kind
					if reflects.IsSimpleKind(ft.Elem().Kind()) {
						continue
					}

					for j := 0; j < fValue.Len(); j++ {
						elemValue := removeValuePtr(fValue.Index(j))
						elemType := removeTypePtr(elemValue.Type())

						arrayName := fmt.Sprintf("%s.%d", name, j)
						if outName != "" {
							fOutMap[arrayName] = fmt.Sprintf("%s.%d", outName, j)
						}
						if elemType.Kind() == reflect.Struct {
							recursiveFunc(elemValue, elemType, arrayName, fv.Anonymous)
						}
					}

				case reflect.Map:
					fValue = removeValuePtr(fValue)

					// Check if the reflect.Value is valid and not a nil pointer
					if !fValue.IsValid() || fValue.IsNil() {
						continue
					}

					for _, key := range fValue.MapKeys() {
						key = removeValuePtr(key)
						elemValue := removeValuePtr(fValue.MapIndex(key))
						elemType := removeTypePtr(elemValue.Type())

						format := "%s."
						kind := key.Kind()
						val := key.Interface()
						switch {
						case kind == reflect.String:
							format += "%s"
							val = strings.ReplaceAll(key.String(), "\"", "")
						case kind >= reflect.Int && kind <= reflect.Uint64:
							format += "%d"
						case kind >= reflect.Float32 && kind <= reflect.Complex128:
							format += "%f"
						default:
							format += "%#v"
						}

						arrayName := fmt.Sprintf(format, name, val)
						if outName != "" {
							fOutMap[arrayName] = fmt.Sprintf(format, outName, val)
						}
						if elemType.Kind() == reflect.Struct {
							recursiveFunc(elemValue, elemType, arrayName, fv.Anonymous)
						}
					}
				case reflect.Ptr:
					// If the field is a pointer type and is nil, and has validation rules, initialize the pointer
					if fValue.IsNil() && vRule != "" {
						// Create an instance of the type pointed to by the pointer
						newValue := reflect.New(ft.Elem())
						// Set the field value
						removeValuePtr(vv).Field(i).Set(newValue)
						// Update fValue to the newly created value
						fValue = newValue
					}

					// Continue processing the type pointed to by the pointer
					if fValue.IsValid() && !fValue.IsNil() && removeTypePtr(ft).Kind() == reflect.Struct {
						recursiveFunc(removeValuePtr(fValue), removeTypePtr(ft), name, fv.Anonymous)
					}
				default:
					// do nothing
				}
			}
		}
	}

	recursiveFunc(removeValuePtr(vv), vt, "", false)

	if len(fOutMap) > 0 {
		v.Trans().AddFieldMap(fOutMap)
	}
}

// eg: `message:"required:name is required|minLen:name min len is %d"`
func (d *StructData) loadMessagesFromTag(trans *Translator, field, vRule, vMsg string) {
	var msgKey, vName string
	var vNames []string

	// only one message, use for first validator.
	// eg: `message:"name is required"`
	if !strings.ContainsRune(vMsg, '|') {
		// eg: `message:"required:name is required"`
		if strings.ContainsRune(vMsg, ':') {
			nodes := strings.SplitN(vMsg, ":", 2)
			vName = strings.TrimSpace(nodes[0])
			vNames = []string{vName}
			// first is validator name
			vMsg = strings.TrimSpace(nodes[1])
		}

		if vName == "" {
			// eg `validate:"required|date"`
			vNames = []string{vRule}
			if strings.ContainsRune(vRule, '|') {
				vNames = strings.Split(vRule, "|")
			}

			for i, node := range vNames {
				// has params for validator: "minLen:5"
				if strings.ContainsRune(node, ':') {
					tmp := strings.SplitN(node, ":", 2)
					vNames[i] = tmp[0]
				}
			}
		}

		// if rName, has := validatorAliases[validator]; has {
		// 	msgKey = field + "." + rName
		// } else {

		for _, name := range vNames {
			msgKey = field + "." + name
			trans.AddMessage(msgKey, vMsg)
		}

		return
	}

	// multi message for validators
	// eg: `message:"required:name is required | minLen:name min len is %d"`
	for _, validatorWithMsg := range strings.Split(vMsg, "|") {
		// validatorWithMsg eg: "required:name is required"
		nodes := strings.SplitN(validatorWithMsg, ":", 2)

		validator := nodes[0]
		msgKey = field + "." + validator

		trans.AddMessage(msgKey, strings.TrimSpace(nodes[1]))
	}
}
