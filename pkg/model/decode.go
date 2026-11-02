package model

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// parseJSON decodes with encoding/json and falls back to the walker when a field needs
// coercion, so input that needs none costs no more than the standard decoder.
func parseJSON[T any](raw []byte) (T, error) {
	var zero, result T

	if err := checkJSONDepth(raw); err != nil {
		return zero, err
	}

	err := json.Unmarshal(raw, &result)
	if err == nil {
		if err := validateParsed(reflect.ValueOf(&result).Elem(), nil, nil); err != nil {
			return zero, err
		}
		return result, nil
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return zero, fmt.Errorf("json parse error: %w", err)
	}

	root, err := parseJSONNode(raw)
	if err != nil {
		return zero, fmt.Errorf("json parse error: %w", err)
	}
	return walkInto[T](root, FormatJSON)
}

// parseYAML always walks: yaml.v3 ignores json tags, so decoding straight into T would
// silently drop any field that only has one.
func parseYAML[T any](raw []byte) (T, error) {
	var zero T

	doc, _, err := decodeYAML(raw)
	if err != nil {
		return zero, err
	}
	return walkInto[T](yamlToNode(doc), FormatYAML)
}

// decodeYAML is the one route from input into yaml.v3's decoder. A merge with a sequence for a
// key panics inside yaml.v3, on 18 bytes of input. No user code runs here, so a recover can only
// be catching yaml.v3.
func decodeYAML(raw []byte) (doc *yaml.Node, generic interface{}, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			doc, generic, err = nil, nil, fmt.Errorf("yaml parse error: %v", recovered)
		}
	}()

	doc = &yaml.Node{}
	if err := yaml.Unmarshal(raw, doc); err != nil {
		return nil, nil, fmt.Errorf("yaml parse error: %w", err)
	}

	if err := checkYAMLKeys(doc); err != nil {
		return nil, nil, err
	}

	// This decode must run even with the depth limit off. yaml.v3 enforces its alias ratio
	// and self-reference checks only while decoding a whole document: a Node target returns
	// before expanding, and Node.Decode at a leaf starts with fresh counters, so the walk
	// has no protection against an alias bomb except this.
	if err := doc.Decode(&generic); err != nil {
		return nil, nil, fmt.Errorf("yaml parse error: %w", err)
	}
	if err := checkStructureDepth(generic); err != nil {
		return nil, nil, err
	}
	return doc, generic, nil
}

// checkYAMLKeys runs before the decode, which compares every pair of keys in a mapping and does
// it again each time an alias replays that mapping. It also formats an error for every
// duplicate pair, so 15KB of repeated keys costs 800MB. Parsing into nodes is linear, so this
// can answer first, in linear time.
func checkYAMLKeys(doc *yaml.Node) error {
	check := yamlKeyCheck{anchored: make(map[*yaml.Node]uint64)}
	comparisons := check.comparisons(doc)
	if check.err != nil {
		return check.err
	}
	if limit := GetMaxYAMLKeyComparisons(); limit > 0 && comparisons > uint64(limit) {
		return fmt.Errorf("yaml parse error: document needs %d key comparisons, over the limit of %d: use SetMaxYAMLKeyComparisons to raise it",
			comparisons, limit)
	}
	return nil
}

type yamlKeyCheck struct {
	anchored map[*yaml.Node]uint64 // Only an anchored node can be reached twice
	err      error
}

// comparisons returns what decoding n costs. An alias costs its target again, as it does in
// yaml.v3, and a merge is an alias. A node in progress answers 0, which stops a cycle: the
// decode rejects those.
func (c *yamlKeyCheck) comparisons(n *yaml.Node) uint64 {
	if n.Kind == yaml.AliasNode {
		return c.comparisons(n.Alias)
	}
	if n.Anchor != "" {
		if total, ok := c.anchored[n]; ok {
			return total
		}
		c.anchored[n] = 0
	}

	var total uint64
	if n.Kind == yaml.MappingNode {
		c.checkDuplicates(n)
		keys := uint64(len(n.Content) / 2) //nolint:gosec // A length is never negative.
		total = keys * (keys - 1) / 2
	}
	for _, child := range n.Content {
		total = addSaturating(total, c.comparisons(child))
	}

	if n.Anchor != "" {
		c.anchored[n] = total
	}
	return total
}

// checkDuplicates uses yaml.v3's definition of a repeated key.
func (c *yamlKeyCheck) checkDuplicates(n *yaml.Node) {
	type key struct {
		kind  yaml.Kind
		value string
	}
	seen := make(map[key]int, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content) && c.err == nil; i += 2 {
		node := n.Content[i]
		k := key{kind: node.Kind, value: node.Value}
		if line, repeated := seen[k]; repeated {
			c.err = fmt.Errorf("yaml parse error: line %d: mapping key %q already defined at line %d", node.Line, node.Value, line)
		}
		seen[k] = node.Line
	}
}

func addSaturating(a, b uint64) uint64 {
	if sum := a + b; sum >= a {
		return sum
	}
	return math.MaxUint64
}

func walkInto[T any](root *node, format Format) (T, error) {
	var zero, result T

	w := walker{format: format}
	out := reflect.ValueOf(&result).Elem()
	w.assign(root, out, nil)

	if err := validateParsed(out, w.errs, failedPaths(w.errs, "")); err != nil {
		return zero, err
	}
	return result, nil
}

// checkJSONDepth finds the nesting depth with a byte scan, so the depth limit does not cost
// a full decode of the input.
func checkJSONDepth(raw []byte) error {
	maxDepth := GetMaxStructureDepth()
	if maxDepth <= 0 {
		return nil
	}

	depth := 0
	inString := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if inString {
			switch c {
			case '\\':
				i++
			case '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			depth++
			if depth > maxDepth {
				return fmt.Errorf("structure depth %d exceeds maximum allowed depth of %d", depth, maxDepth)
			}
		case '}', ']':
			depth--
		}
	}
	return nil
}

type nodeKind int

const (
	nodeNull nodeKind = iota
	nodeScalar
	nodeObject
	nodeArray
)

// node is one value of a parsed document. The walker takes structure from it and hands leaves
// back to the format's own decoder through raw or yml, so a value is never re-encoded and
// RawMessage bytes and number spellings survive.
type node struct {
	kind   nodeKind
	scalar interface{}
	keys   []string
	values []*node
	raw    []byte
	yml    *yaml.Node

	keyNodes []*yaml.Node // Parallel to keys, for YAML mappings

	repeatedKey string // A key repeated through an alias, which yaml.v3 refuses on a struct
	hasMerge    bool
}

// isNullKey reports a YAML null key. Its text is whatever the document spelled it as, so it must
// not be taken for a string.
func (n *node) isNullKey(i int) bool {
	return i < len(n.keyNodes) && n.keyNodes[i].ShortTag() == "!!null"
}

func (n *node) hasNullKey() bool {
	for i := range n.keyNodes {
		if n.isNullKey(i) {
			return true
		}
	}
	return false
}

// hasLossyFloatKey reports a float key yaml.v3 would change, as it does a value: it truncates a
// fraction into an integer key, and does not range-check a float32 key.
func (n *node) hasLossyFloatKey(keyType reflect.Type) bool {
	if !isIntegerKind(keyType.Kind()) && keyType.Kind() != reflect.Float32 {
		return false
	}
	for _, key := range n.keyNodes {
		if key.ShortTag() != "!!float" {
			continue
		}
		var value float64
		if keyType.Kind() == reflect.Float32 || key.Decode(&value) != nil || value != math.Trunc(value) {
			return true
		}
	}
	return false
}

func isIntegerKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

func (n *node) decodeInto(out interface{}) error {
	if n.yml != nil {
		return n.yml.Decode(out)
	}
	return json.Unmarshal(n.raw, out)
}

// value returns the node as a plain Go value for coercion and error reporting.
func (n *node) value() interface{} {
	if n.kind == nodeScalar && n.yml == nil {
		return n.scalar
	}
	var v interface{}
	if err := n.decodeInto(&v); err != nil {
		return nil
	}
	return v
}

func (n *node) describe() string {
	switch n.kind {
	case nodeObject:
		return "object"
	case nodeArray:
		return "array"
	case nodeNull:
		return "null"
	default:
		return fmt.Sprintf("%T", n.value())
	}
}

func parseJSONNode(raw []byte) (*node, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	b := jsonNodeBuilder{dec: dec, raw: raw}
	return b.value()
}

type jsonNodeBuilder struct {
	dec *json.Decoder
	raw []byte
}

// start returns where the next value begins. InputOffset stops at the end of the last token,
// which leaves separators and whitespace still to skip.
func (b *jsonNodeBuilder) start() int {
	i := int(b.dec.InputOffset())
	for i < len(b.raw) {
		switch b.raw[i] {
		case ' ', '\t', '\r', '\n', ',', ':':
			i++
		default:
			return i
		}
	}
	return i
}

func (b *jsonNodeBuilder) value() (*node, error) {
	start := b.start()
	token, err := b.dec.Token()
	if err != nil {
		return nil, err
	}

	switch t := token.(type) {
	case json.Delim:
		if t == '{' {
			return b.object(start)
		}
		return b.array(start)
	case nil:
		return &node{kind: nodeNull, raw: b.raw[start:b.dec.InputOffset()]}, nil
	default:
		return &node{kind: nodeScalar, scalar: t, raw: b.raw[start:b.dec.InputOffset()]}, nil
	}
}

func (b *jsonNodeBuilder) object(start int) (*node, error) {
	n := &node{kind: nodeObject}
	for b.dec.More() {
		key, err := b.dec.Token()
		if err != nil {
			return nil, err
		}
		value, err := b.value()
		if err != nil {
			return nil, err
		}
		n.keys = append(n.keys, key.(string))
		n.values = append(n.values, value)
	}
	if _, err := b.dec.Token(); err != nil {
		return nil, err
	}
	n.raw = b.raw[start:b.dec.InputOffset()]
	return n, nil
}

func (b *jsonNodeBuilder) array(start int) (*node, error) {
	n := &node{kind: nodeArray}
	for b.dec.More() {
		value, err := b.value()
		if err != nil {
			return nil, err
		}
		n.values = append(n.values, value)
	}
	if _, err := b.dec.Token(); err != nil {
		return nil, err
	}
	n.raw = b.raw[start:b.dec.InputOffset()]
	return n, nil
}

// yamlToNode expands aliases and merge keys. It must only see a document that passed the
// whole-document decode in parseYAML, which is what rejects cycles and alias bombs.
func yamlToNode(y *yaml.Node) *node {
	switch y.Kind {
	case yaml.DocumentNode:
		if len(y.Content) == 0 {
			return &node{kind: nodeNull}
		}
		return yamlToNode(y.Content[0])
	case yaml.AliasNode:
		return yamlToNode(y.Alias)
	case yaml.ScalarNode:
		if y.ShortTag() == "!!null" {
			return &node{kind: nodeNull, yml: y}
		}
		return &node{kind: nodeScalar, yml: y}
	case yaml.SequenceNode:
		n := &node{kind: nodeArray, yml: y, values: make([]*node, 0, len(y.Content))}
		for _, item := range y.Content {
			n.values = append(n.values, yamlToNode(item))
		}
		return n
	case yaml.MappingNode:
		n := &node{kind: nodeObject, yml: y}
		addYAMLMapping(n, y, map[string]int{}, false)
		return n
	default:
		return &node{kind: nodeNull}
	}
}

// addYAMLMapping follows yaml.v3's merge rules: explicit keys beat merged ones at every level,
// a later << replaces an earlier one, and earlier sources in a merge list win. Keys already
// seen are skipped before their value is visited, so this expands no more than the
// whole-document decode that vetted it.
// addYAMLMapping flattens a mapping and its merges. A key repeated in the mapping itself, which
// only an alias can do, takes the later value. A merged key never replaces one already there.
func addYAMLMapping(n *node, y *yaml.Node, seen map[string]int, merged bool) {
	var merge *yaml.Node
	for i := 0; i+1 < len(y.Content); i += 2 {
		key, value := y.Content[i], y.Content[i+1]
		if isYAMLMerge(key) {
			merge = value
			continue
		}
		if key.Kind == yaml.AliasNode {
			key = key.Alias
		}
		// A null key is not the string spelled the same way.
		id := key.Value
		if key.ShortTag() == "!!null" {
			id = "\x00null"
		}
		if at, repeated := seen[id]; repeated {
			if !merged {
				n.repeatedKey = key.Value
				n.keyNodes[at], n.values[at] = key, yamlToNode(value)
			}
			continue
		}
		seen[id] = len(n.keys)
		n.keys = append(n.keys, key.Value)
		n.keyNodes = append(n.keyNodes, key)
		n.values = append(n.values, yamlToNode(value))
	}

	if merge == nil {
		return
	}
	n.hasMerge = true
	sources := []*yaml.Node{merge}
	if merge.Kind == yaml.SequenceNode {
		sources = merge.Content
	}
	for _, source := range sources {
		if source.Kind == yaml.AliasNode {
			source = source.Alias
		}
		if source.Kind == yaml.MappingNode {
			addYAMLMapping(n, source, seen, true)
		}
	}
}

func isYAMLMerge(key *yaml.Node) bool {
	return key.Kind == yaml.ScalarNode && key.Value == "<<" &&
		(key.Tag == "" || key.Tag == "!" || key.ShortTag() == "!!merge")
}

var (
	timeType            = reflect.TypeOf(time.Time{})
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	yamlUnmarshalerType = reflect.TypeOf((*yaml.Unmarshaler)(nil)).Elem()

	// yaml.v3 still honours the func-signature form, through an unexported interface.
	yamlFuncUnmarshalerType = reflect.TypeOf((*interface {
		UnmarshalYAML(func(interface{}) error) error
	})(nil)).Elem()
	yamlNodeType = reflect.TypeOf(yaml.Node{})

	jsonNumberType = reflect.TypeOf(json.Number(""))
)

// decodesItself reports whether typ owns its decoding, which makes it a leaf whatever its Kind.
func decodesItself(typ reflect.Type, n *node) bool {
	ptr := reflect.PointerTo(typ)
	if ptr.Implements(textUnmarshalerType) {
		return true
	}
	if n.yml != nil {
		return typ == yamlNodeType || ptr.Implements(yamlUnmarshalerType) || ptr.Implements(yamlFuncUnmarshalerType)
	}
	return ptr.Implements(jsonUnmarshalerType)
}

// walker decodes a node tree into a value, directed by the target type. It is the only code
// path for input the standard decoders reject.
type walker struct {
	format Format
	errs   ErrorList
}

func (w *walker) fail(path *fieldPath, n *node, typ reflect.Type, message string) {
	// Sensitivity is judged on the path down to n, so keys inside n must not reach Value.
	var value interface{} = n.describe()
	if n.kind == nodeScalar {
		value = n.value()
	}
	w.report(path, value, typ.String(), message)
}

func (w *walker) report(path *fieldPath, value interface{}, typ, message string) {
	parseErr := NewParseError(path.String(), value, typ, message)
	if path.sensitive() {
		parseErr.redact()
	}
	w.errs.addCapped(parseErr)
}

func (w *walker) assign(n *node, out reflect.Value, path *fieldPath) {
	if w.errs.full() {
		return
	}

	typ := out.Type()
	if n.kind == nodeNull {
		w.assignNull(n, out, path)
		return
	}

	if out.Kind() == reflect.Ptr {
		if out.IsNil() {
			out.Set(reflect.New(typ.Elem()))
		}
		w.assign(n, out.Elem(), path)
		return
	}

	if decodesItself(typ, n) {
		w.assignSelfDecoding(n, out, path)
		return
	}

	switch out.Kind() {
	case reflect.Struct:
		w.assignStruct(n, out, path)
	case reflect.Slice:
		w.assignSlice(n, out, path)
	case reflect.Array:
		w.assignArray(n, out, path)
	case reflect.Map:
		w.assignMap(n, out, path)
	case reflect.Interface:
		if err := n.decodeInto(out.Addr().Interface()); err != nil {
			w.fail(path, n, typ, err.Error())
		}
	default:
		w.assignScalar(n, out, path)
	}
}

// assignNull follows the decoders: an addressable UnmarshalJSON sees the null, a pointer is
// cleared without calling it, and a yaml.Node keeps the null node. The format has to be read
// from the walker, since an empty YAML document is a null with no yaml.Node behind it.
func (w *walker) assignNull(n *node, out reflect.Value, path *fieldPath) {
	seesNull := w.format == FormatJSON && out.Kind() != reflect.Ptr && reflect.PointerTo(out.Type()).Implements(jsonUnmarshalerType)
	if seesNull || (n.yml != nil && out.Type() == yamlNodeType) {
		w.assignSelfDecoding(n, out, path)
		return
	}
	switch out.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		out.Set(reflect.Zero(out.Type()))
	}
}

func (w *walker) assignSelfDecoding(n *node, out reflect.Value, path *fieldPath) {
	err := n.decodeInto(out.Addr().Interface())
	if err == nil {
		return
	}
	// time.Time's own decoder accepts one layout; coercion accepts several, and numbers.
	if out.Type() == timeType && n.kind == nodeScalar {
		w.coerceScalar(n, out, path)
		return
	}
	w.fail(path, n, out.Type(), err.Error())
}

// assignScalar lets the format's decoder go first, so coercion only sees what it rejected.
func (w *walker) assignScalar(n *node, out reflect.Value, path *fieldPath) {
	if n.kind != nodeScalar {
		w.fail(path, n, out.Type(), fmt.Sprintf("cannot coerce %s to %s", n.describe(), out.Type()))
		return
	}
	if n.yml != nil && out.Type() == jsonNumberType {
		w.assignYAMLNumber(n, out, path)
		return
	}
	if !yamlTruncates(n, out) {
		if err := n.decodeInto(out.Addr().Interface()); err == nil {
			return
		}
	}
	w.coerceScalar(n, out, path)
}

// yamlTruncates picks out the case yaml.v3 gets wrong: it truncates a float into an integer and
// does not range-check a float. Its other number handling has to stay, since it also parses
// "30s" into a time.Duration.
func yamlTruncates(n *node, out reflect.Value) bool {
	return n.yml != nil && (out.CanInt() || out.CanUint() || out.CanFloat()) && n.yml.ShortTag() == "!!float"
}

// assignYAMLNumber keeps the digits as written, which is the reason json.Number exists.
// yaml.v3 would store any scalar in it, since it is a string kind.
func (w *walker) assignYAMLNumber(n *node, out reflect.Value, path *fieldPath) {
	if isJSONNumber(n.yml.Value) {
		out.SetString(n.yml.Value)
		return
	}
	// Spellings only YAML has, like 0x10, go through their value.
	if tag := n.yml.ShortTag(); tag == "!!int" || tag == "!!float" {
		w.coerceScalar(n, out, path)
		return
	}
	w.fail(path, n, out.Type(), fmt.Sprintf("%q is not a number", n.yml.Value))
}

func (w *walker) coerceScalar(n *node, out reflect.Value, path *fieldPath) {
	coerced, err := CoerceValueWithFormat(n.value(), out.Type(), "", w.format)
	if err != nil {
		var parseErr *ParseError
		if errors.As(err, &parseErr) {
			parseErr.Field = path.String()
			if path.sensitive() {
				parseErr.redact()
			}
		}
		w.errs.addCapped(err)
		return
	}
	setCoerced(out, coerced)
}

func setCoerced(out reflect.Value, coerced interface{}) {
	switch out.Kind() {
	case reflect.String:
		out.SetString(coerced.(string))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		out.SetInt(coerced.(int64))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out.SetUint(coerced.(uint64))
	case reflect.Float32, reflect.Float64:
		out.SetFloat(coerced.(float64))
	case reflect.Bool:
		out.SetBool(coerced.(bool))
	default:
		out.Set(reflect.ValueOf(coerced))
	}
}

func (w *walker) assignStruct(n *node, out reflect.Value, path *fieldPath) {
	if n.kind != nodeObject {
		w.fail(path, n, out.Type(), fmt.Sprintf("cannot coerce %s to %s", n.describe(), out.Type()))
		return
	}

	table := fieldsOf(out.Type(), w.format)
	if n.repeatedKey != "" && table.lookup(n.repeatedKey, w.format) != nil {
		w.fail(path, n, out.Type(), fmt.Sprintf("field %q is set twice", n.repeatedKey))
		return
	}
	for i, key := range n.keys {
		if w.errs.full() {
			return
		}
		if n.isNullKey(i) {
			if table.inlineMap != nil {
				w.report(table.inlineField(path), nil, out.Type().String(), "a null key cannot go in this map")
			}
			continue
		}
		entry := table.lookup(key, w.format)
		if entry == nil {
			w.assignInline(table, key, n.values[i], out, path)
			continue
		}
		here := &fieldPath{parent: path, kind: pathField, name: entry.path, key: key, hidden: entry.hidden}
		dst, err := fieldByIndex(out, entry.index)
		if err != nil {
			w.report(here, nil, out.Type().String(), err.Error())
			continue
		}
		value := n.values[i]
		if entry.quoted {
			value = unquoted(value)
		}
		w.assign(value, dst, here)
	}
}

// unquoted undoes the json ",string" option, which wraps a scalar in a JSON string. Anything
// else stays as it is, for coercion to judge. The depth scan skips string contents, so nesting
// here must be refused before it reaches the recursive node builder.
func unquoted(n *node) *node {
	text, isString := n.scalar.(string)
	if n.kind != nodeScalar || !isString {
		return n
	}
	raw := []byte(text)
	if len(raw) == 0 || len(bytes.TrimSpace(raw)) != len(raw) || raw[0] == '{' || raw[0] == '[' || !json.Valid(raw) {
		return n
	}
	inner, err := parseJSONNode(raw)
	if err != nil {
		return n
	}
	return inner
}

// isQuoted reports the ",string" option, which encoding/json honours only on scalars.
func isQuoted(sf reflect.StructField, target reflect.Type) bool {
	_, options, _ := strings.Cut(sf.Tag.Get("json"), ",")
	if !strings.Contains(","+options+",", ",string,") {
		return false
	}
	switch target.Kind() {
	case reflect.Bool, reflect.String, reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	default:
		return false
	}
}

// assignInline routes a key no field claimed into a yaml `,inline` map, if the type has one.
func (w *walker) assignInline(table *fieldTable, key string, value *node, out reflect.Value, path *fieldPath) {
	if table.inlineMap == nil {
		return
	}
	here := table.inlineField(path)
	dst, err := fieldByIndex(out, table.inlineMap)
	if err != nil {
		w.report(here, nil, out.Type().String(), err.Error())
		return
	}
	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}
	w.assignMapEntry(key, value, dst, here)
}

func (w *walker) assignSlice(n *node, out reflect.Value, path *fieldPath) {
	typ := out.Type()
	if n.kind == nodeScalar {
		// A string can decode into []byte; anything else fails here with a clear message.
		if err := n.decodeInto(out.Addr().Interface()); err != nil {
			w.fail(path, n, typ, fmt.Sprintf("cannot coerce %s to %s", n.describe(), typ))
		}
		return
	}
	if n.kind != nodeArray {
		w.fail(path, n, typ, fmt.Sprintf("cannot coerce %s to %s", n.describe(), typ))
		return
	}

	slice := reflect.MakeSlice(typ, len(n.values), len(n.values))
	for i, item := range n.values {
		w.assign(item, slice.Index(i), &fieldPath{parent: path, kind: pathIndex, index: i})
	}
	out.Set(slice)
}

func (w *walker) assignArray(n *node, out reflect.Value, path *fieldPath) {
	typ := out.Type()
	if n.kind != nodeArray {
		w.fail(path, n, typ, fmt.Sprintf("cannot coerce %s to %s", n.describe(), typ))
		return
	}
	// Like encoding/json: extra elements are ignored and missing ones stay zero. An error here
	// would make the same array valid or not depending on whether a sibling needed coercion.
	for i := 0; i < len(n.values) && i < typ.Len(); i++ {
		w.assign(n.values[i], out.Index(i), &fieldPath{parent: path, kind: pathIndex, index: i})
	}
}

func (w *walker) assignMap(n *node, out reflect.Value, path *fieldPath) {
	typ := out.Type()
	if n.kind != nodeObject {
		w.fail(path, n, typ, fmt.Sprintf("cannot coerce %s to %s", n.describe(), typ))
		return
	}
	// Checked before either path, so the answer does not depend on some other key.
	if typ.Key().Kind() != reflect.Interface && n.hasNullKey() {
		w.report(path, nil, typ.String(), "a null key cannot go in this map")
		return
	}
	if n.yml != nil && !n.hasLossyFloatKey(typ.Key()) && w.assignYAMLMap(n, out, path) {
		return
	}
	if out.IsNil() {
		out.Set(reflect.MakeMapWithSize(typ, len(n.keys)))
	}
	for i, key := range n.keys {
		w.assignMapEntry(key, n.values[i], out, path)
	}
}

// assignYAMLMap lets yaml.v3 settle the keys, by decoding into nodes. It identifies a key by its
// decoded value for the target key type, and merges depend on that, so comparing key text here
// can only approximate it. Values stay on the walker, for coercion, json tags and redaction.
//
// It reports false when yaml.v3 refuses, and the text keys take over. That covers "1" into an
// int key, and a fresh decoder tripping the alias ratio on a large merge. An interface key has no
// such fallback, since text would turn 1 into "1".
func (w *walker) assignYAMLMap(n *node, out reflect.Value, path *fieldPath) bool {
	typ := out.Type()
	decoded := reflect.New(reflect.MapOf(typ.Key(), yamlNodeType))
	if err := n.yml.Decode(decoded.Interface()); err != nil {
		if typ.Key().Kind() != reflect.Interface {
			return false
		}
		w.fail(path, n, typ, err.Error())
		return true
	}
	if !n.hasMerge && decoded.Elem().Len() != len(n.keys) {
		w.fail(path, n, typ, "two keys are the same map key")
		return true
	}

	type entry struct {
		label string
		key   reflect.Value
		value *yaml.Node
	}
	entries := make([]entry, 0, decoded.Elem().Len())
	for iter := decoded.Elem().MapRange(); iter.Next(); {
		value := iter.Value().Interface().(yaml.Node)
		entries = append(entries, entry{label: fmt.Sprint(iter.Key().Interface()), key: iter.Key(), value: &value})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].label < entries[j].label })

	if out.IsNil() {
		out.Set(reflect.MakeMapWithSize(typ, len(entries)))
	}
	for _, e := range entries {
		if w.errs.full() {
			break
		}
		w.setMapEntry(e.key, yamlToNode(e.value), out, &fieldPath{parent: path, kind: pathKey, key: e.label})
	}
	return true
}

func (w *walker) assignMapEntry(key string, value *node, out reflect.Value, path *fieldPath) {
	typ := out.Type()
	here := &fieldPath{parent: path, kind: pathKey, key: key}

	mapKey, err := coerceMapKey(key, typ.Key())
	if err != nil {
		w.report(here, key, typ.Key().String(), err.Error())
		return
	}
	// JSON keeps the last, as encoding/json does.
	if w.format == FormatYAML && out.MapIndex(mapKey).IsValid() {
		w.report(here, key, typ.Key().String(), "two keys are the same map key")
		return
	}
	w.setMapEntry(mapKey, value, out, here)
}

func (w *walker) setMapEntry(mapKey reflect.Value, value *node, out reflect.Value, here *fieldPath) {
	elem := reflect.New(out.Type().Elem()).Elem()
	w.assign(value, elem, here)
	out.SetMapIndex(mapKey, elem)
}

func coerceMapKey(key string, keyType reflect.Type) (reflect.Value, error) {
	out := reflect.New(keyType).Elem()
	if unmarshaler, ok := out.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return out, unmarshaler.UnmarshalText([]byte(key))
	}
	if keyType.Kind() == reflect.String {
		out.SetString(key)
		return out, nil
	}
	coerced, err := CoerceValue(key, keyType, "")
	if err != nil {
		return out, err
	}
	setCoerced(out, coerced)
	return out, nil
}

func fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for i, step := range index {
		if i > 0 && v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() {
					return v, fmt.Errorf("cannot set embedded pointer to unexported struct %s", v.Type().Elem())
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(step)
	}
	return v, nil
}

// fieldEntry is one decodable key of a struct, possibly promoted from an embedded struct.
type fieldEntry struct {
	name   string
	index  []int
	path   string // Go field names, matching the paths validation reports
	depth  int
	tag    bool
	hidden bool // Kept out of JSON, so set only by YAML
	quoted bool // Has the json ",string" option
}

type fieldTable struct {
	byName    map[string]*fieldEntry
	ordered   []*fieldEntry
	inlineMap []int

	inlinePath   string
	inlineHidden bool // The inline map is kept out of JSON
}

func (t *fieldTable) inlineField(path *fieldPath) *fieldPath {
	return &fieldPath{parent: path, kind: pathField, name: t.inlinePath, hidden: t.inlineHidden}
}

// lookup matches like the format's decoder: encoding/json falls back to case folding, and
// yaml.v3 does not.
func (t *fieldTable) lookup(key string, format Format) *fieldEntry {
	if entry := t.byName[key]; entry != nil {
		return entry
	}
	if format != FormatJSON {
		return nil
	}
	for _, entry := range t.ordered {
		if strings.EqualFold(entry.name, key) {
			return entry
		}
	}
	return nil
}

func fieldsOf(typ reflect.Type, format Format) *fieldTable {
	key := fieldKeyCacheKey{typ: typ, format: format}
	if cached, ok := fieldKeyCache.Load(key); ok {
		return cached.(*fieldTable)
	}

	var table *fieldTable
	if format == FormatYAML {
		table = buildYAMLFields(typ)
	} else {
		table = buildJSONFields(typ)
	}
	fieldKeyCache.Store(key, table)
	return table
}

// embeddedStruct is a struct still to scan while resolving promoted fields.
type embeddedStruct struct {
	typ   reflect.Type
	index []int
	path  string
}

// buildJSONFields resolves fields the way encoding/json does, level by level through embedded
// structs, so precedence and ambiguity match the standard decoder.
func buildJSONFields(typ reflect.Type) *fieldTable {
	var candidates []*fieldEntry
	visited := map[reflect.Type]bool{}
	next := []embeddedStruct{{typ: typ}}

	for depth := 0; len(next) > 0; depth++ {
		level := next
		next = nil
		counts := map[reflect.Type]int{}
		for _, current := range level {
			counts[current.typ]++
		}

		for _, current := range level {
			if visited[current.typ] {
				continue
			}
			visited[current.typ] = true

			for i := 0; i < current.typ.NumField(); i++ {
				sf := current.typ.Field(i)
				name, skip := jsonFieldName(sf)
				if skip {
					continue
				}

				index := append(append([]int(nil), current.index...), i)
				elem := sf.Type
				if elem.Name() == "" && elem.Kind() == reflect.Ptr {
					elem = elem.Elem()
				}

				path := current.path
				if sf.IsExported() {
					path = joinFieldNames(path, sf.Name)
				}

				if name == "" && sf.Anonymous && elem.Kind() == reflect.Struct {
					next = append(next, embeddedStruct{typ: elem, index: index, path: path})
					continue
				}

				entry := &fieldEntry{name: name, index: index, path: path, depth: depth, tag: name != "", quoted: isQuoted(sf, elem)}
				if entry.name == "" {
					entry.name = sf.Name
				}
				candidates = append(candidates, entry)
				if counts[current.typ] > 1 {
					candidates = append(candidates, entry) // embedded twice: annihilates itself
				}
			}
		}
	}

	return newFieldTable(dominantFields(candidates))
}

func jsonFieldName(sf reflect.StructField) (name string, skip bool) {
	if sf.Anonymous {
		elem := sf.Type
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if !sf.IsExported() && elem.Kind() != reflect.Struct {
			return "", true
		}
	} else if !sf.IsExported() {
		return "", true
	}

	tag := sf.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	name, _, _ = strings.Cut(tag, ",")
	return name, false
}

// dominantFields applies encoding/json's rules for a name claimed more than once: shallowest
// wins, then tagged, and a remaining tie removes the name entirely.
func dominantFields(candidates []*fieldEntry) []*fieldEntry {
	groups := map[string][]*fieldEntry{}
	for _, entry := range candidates {
		groups[entry.name] = append(groups[entry.name], entry)
	}

	var fields []*fieldEntry
	for _, group := range groups {
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].depth != group[j].depth {
				return group[i].depth < group[j].depth
			}
			return group[i].tag && !group[j].tag
		})
		if len(group) > 1 && group[0].depth == group[1].depth && group[0].tag == group[1].tag {
			continue
		}
		fields = append(fields, group[0])
	}
	return fields
}

func newFieldTable(fields []*fieldEntry) *fieldTable {
	sort.Slice(fields, func(i, j int) bool {
		a, b := fields[i].index, fields[j].index
		for k := 0; k < len(a) && k < len(b); k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})

	table := &fieldTable{byName: make(map[string]*fieldEntry, len(fields)), ordered: fields}
	for _, entry := range fields {
		table.byName[entry.name] = entry
	}
	return table
}

func joinFieldNames(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// buildYAMLFields resolves keys as yaml tag, then json tag, then lowercased field name.
func buildYAMLFields(typ reflect.Type) *fieldTable {
	table := &fieldTable{}
	var fields []*fieldEntry
	collectYAMLFields(typ, nil, "", false, map[reflect.Type]bool{}, &fields, table)

	built := newFieldTable(fields)
	built.inlineMap = table.inlineMap
	built.inlinePath = table.inlinePath
	built.inlineHidden = table.inlineHidden
	return built
}

func collectYAMLFields(typ reflect.Type, index []int, path string, hidden bool, active map[reflect.Type]bool, fields *[]*fieldEntry, table *fieldTable) {
	if active[typ] {
		return
	}
	active[typ] = true
	defer delete(active, typ)

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		name, inline, skip := yamlFieldName(sf)
		if skip {
			continue
		}

		fieldIndex := append(append([]int(nil), index...), i)
		fieldPath := path
		if sf.IsExported() {
			fieldPath = joinFieldNames(path, sf.Name)
		}
		_, jsonHidden := jsonFieldName(sf)
		jsonHidden = jsonHidden || hidden
		if inline {
			elem := sf.Type
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			switch elem.Kind() {
			case reflect.Struct:
				collectYAMLFields(elem, fieldIndex, fieldPath, jsonHidden, active, fields, table)
			case reflect.Map:
				if sf.IsExported() && sf.Type.Kind() == reflect.Map {
					table.inlineMap = fieldIndex
					table.inlinePath = fieldPath
					table.inlineHidden = jsonHidden
				}
			}
			continue
		}
		*fields = append(*fields, &fieldEntry{name: name, index: fieldIndex, path: fieldPath, hidden: jsonHidden})
	}
}

func yamlFieldName(sf reflect.StructField) (name string, inline, skip bool) {
	tag := sf.Tag.Get("yaml")
	if tag == "-" {
		return "", false, true
	}
	name, options, _ := strings.Cut(tag, ",")
	inline = strings.Contains(","+options+",", ",inline,")

	// As in yaml.v3, only an embedded struct is reachable when unexported.
	if !sf.IsExported() && !(inline && sf.Anonymous) {
		return "", false, true
	}
	if name != "" || inline {
		return name, inline, false
	}

	jsonTag := sf.Tag.Get("json")
	if jsonTag == "-" {
		return "", false, true
	}
	if name, _, _ = strings.Cut(jsonTag, ","); name != "" {
		return name, false, false
	}
	return strings.ToLower(sf.Name), false, false
}

// coerceContainer gives CoerceValue the same semantics as parsing: the value is rendered to
// JSON and walked, so there is no second implementation of container decoding.
func coerceContainer(value interface{}, targetType reflect.Type, fieldName string, format Format) (interface{}, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("cannot coerce %T to %s", value, targetType))
	}
	root, err := parseJSONNode(encoded)
	if err != nil {
		return nil, NewParseError(fieldName, value, targetType.String(), err.Error())
	}

	w := walker{format: format}
	out := reflect.New(targetType).Elem()
	var path *fieldPath
	if fieldName != "" {
		path = &fieldPath{kind: pathField, name: fieldName}
	}
	w.assign(root, out, path)
	err = validateParsed(out, w.errs, failedPaths(w.errs, fieldName))
	if list, isList := err.(ErrorList); isList && len(list) == 1 {
		return nil, list[0]
	}
	if err != nil {
		return nil, err
	}
	return out.Interface(), nil
}
