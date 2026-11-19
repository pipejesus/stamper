// Package lando reads, builds and patches Landofiles (.lando.yml) while
// preserving everything stamper does not own: user keys, ordering, comments.
package lando

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Marker is the line comment stamper puts on sequence items it owns, so it
// can find and replace its own entries inside lists users also edit.
const Marker = "stamper"

// SupportPath is where stamper keeps support files; list items referencing
// it (e.g. post-start events) are recognised as stamper's even when YAML
// comments were lost or moved by hand edits.
const SupportPath = ".lando/stamper/"

// IsOwnedItem reports whether a sequence item belongs to stamper.
func IsOwnedItem(n *yaml.Node) bool {
	if n == nil {
		return false
	}
	if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(n.LineComment), "#")) == Marker {
		return true
	}
	switch n.Kind {
	case yaml.ScalarNode:
		return strings.Contains(n.Value, SupportPath)
	case yaml.MappingNode, yaml.SequenceNode:
		for _, c := range n.Content {
			if IsOwnedItem(c) {
				return true
			}
		}
	}
	return false
}

// Path addresses a key inside a mapping tree, e.g. {"services", "redis"}.
type Path []string

func (p Path) String() string {
	s := ""
	for i, k := range p {
		if i > 0 {
			s += "."
		}
		s += k
	}
	return s
}

// Owned declares a part of the Landofile a recipe controls.
// With Items=false the whole value at Path is owned (set or deleted).
// With Items=true only the sequence items at Path tagged with Marker are owned;
// untagged user items in the same list are kept.
type Owned struct {
	Path  Path
	Items bool
}

// Map builds an ordered mapping node from alternating key/value pairs.
// A nil value skips the pair, which keeps conditional builders terse.
func Map(kv ...any) *yaml.Node {
	if len(kv)%2 != 0 {
		panic("lando.Map: odd number of arguments")
	}
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for i := 0; i < len(kv); i += 2 {
		v := Node(kv[i+1])
		if v == nil {
			continue
		}
		n.Content = append(n.Content, Str(fmt.Sprint(kv[i])), v)
	}
	return n
}

// Seq builds a sequence node; nil items are skipped.
func Seq(items ...any) *yaml.Node {
	n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, it := range items {
		if v := Node(it); v != nil {
			n.Content = append(n.Content, v)
		}
	}
	return n
}

// Tagged returns a copy of v marked as stamper-owned (for Owned{Items:true}).
func Tagged(v any) *yaml.Node {
	n := Node(v)
	if n == nil {
		return nil
	}
	c := *n
	if c.Kind == yaml.ScalarNode {
		// Only scalars carry the marker comment reliably through re-encoding;
		// mappings are recognised by referencing SupportPath (see IsOwnedItem).
		c.LineComment = Marker
	}
	return &c
}

// Str builds a plain string scalar.
func Str(s string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s} }

// Node converts common Go values to YAML nodes.
func Node(v any) *yaml.Node {
	switch t := v.(type) {
	case nil:
		return nil
	case *yaml.Node:
		return t
	case string:
		n := Str(t)
		if looksNonString(t) {
			n.Style = yaml.DoubleQuotedStyle // keep "8.4" a string, never a float
		}
		return n
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(t)}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(t)}
	case []string:
		s := Seq()
		for _, x := range t {
			s.Content = append(s.Content, Node(x))
		}
		return s
	case map[string]string:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		m := Map()
		for _, k := range keys {
			m.Content = append(m.Content, Str(k), Node(t[k]))
		}
		return m
	default:
		panic(fmt.Sprintf("lando.Node: unsupported type %T", v))
	}
}

func looksNonString(s string) bool {
	var probe any
	if err := yaml.Unmarshal([]byte(s), &probe); err != nil {
		return true
	}
	_, isStr := probe.(string)
	return !isStr
}

// Lookup returns the node at path inside mapping m, or nil.
func Lookup(m *yaml.Node, path Path) *yaml.Node {
	cur := unwrap(m)
	for _, key := range path {
		if cur == nil || cur.Kind != yaml.MappingNode {
			return nil
		}
		cur = mapGet(cur, key)
	}
	return cur
}

// Scalar returns the scalar string at path, or "".
func Scalar(m *yaml.Node, path ...string) string {
	n := Lookup(m, path)
	if n == nil || n.Kind != yaml.ScalarNode {
		return ""
	}
	return n.Value
}

// Set writes value at path, creating intermediate mappings as needed.
// Existing keys keep their position (and comments); new keys are appended.
func Set(m *yaml.Node, path Path, value *yaml.Node) {
	cur := unwrap(m)
	for i, key := range path {
		last := i == len(path)-1
		idx := mapIndex(cur, key)
		switch {
		case last && idx >= 0:
			old := cur.Content[idx+1]
			value.HeadComment, value.FootComment = keep(value.HeadComment, old.HeadComment), keep(value.FootComment, old.FootComment)
			cur.Content[idx+1] = value
		case last:
			cur.Content = append(cur.Content, Str(key), value)
		case idx >= 0 && cur.Content[idx+1].Kind == yaml.MappingNode:
			cur = cur.Content[idx+1]
		default:
			child := Map()
			if idx >= 0 {
				cur.Content[idx+1] = child
			} else {
				cur.Content = append(cur.Content, Str(key), child)
			}
			cur = child
		}
	}
}

// Delete removes the key at path and prunes parent mappings left empty.
func Delete(m *yaml.Node, path Path) {
	if len(path) == 0 {
		return
	}
	parent := Lookup(m, path[:len(path)-1])
	if parent == nil || parent.Kind != yaml.MappingNode {
		return
	}
	if idx := mapIndex(parent, path[len(path)-1]); idx >= 0 {
		parent.Content = append(parent.Content[:idx], parent.Content[idx+2:]...)
	}
	if len(path) > 1 && len(parent.Content) == 0 {
		Delete(m, path[:len(path)-1])
	}
}

// Apply reconciles doc with the freshly rendered managed tree: every owned
// path takes the managed value, or is removed when the recipe no longer
// produces it. Everything else in doc is left untouched.
func Apply(doc, managed *yaml.Node, owned []Owned) {
	for _, o := range owned {
		want := Lookup(managed, o.Path)
		if !o.Items {
			if want == nil {
				Delete(doc, o.Path)
			} else {
				Set(doc, o.Path, want)
			}
			continue
		}
		var items []*yaml.Node
		if cur := Lookup(doc, o.Path); cur != nil && cur.Kind == yaml.SequenceNode {
			for _, it := range cur.Content {
				if !IsOwnedItem(it) {
					items = append(items, it)
				}
			}
		}
		if want != nil && want.Kind == yaml.SequenceNode {
			for _, it := range want.Content {
				if IsOwnedItem(it) {
					items = append(items, it)
				}
			}
		}
		if len(items) == 0 {
			Delete(doc, o.Path)
			continue
		}
		seq := Seq()
		seq.Content = items
		Set(doc, o.Path, seq)
	}
	reorderTop(doc, managed)
}

// reorderTop sorts doc's top-level keys into managed's order. Keys unknown to
// managed (user additions) stay glued behind the key they followed.
func reorderTop(doc, managed *yaml.Node) {
	d, m := unwrap(doc), unwrap(managed)
	if d == nil || m == nil || d.Kind != yaml.MappingNode {
		return
	}
	rank := map[string]int{}
	for i := 0; i+1 < len(m.Content); i += 2 {
		rank[m.Content[i].Value] = i/2 + 1
	}
	type pair struct {
		k, v *yaml.Node
		r    int
	}
	pairs := make([]pair, 0, len(d.Content)/2)
	last := 0
	for i := 0; i+1 < len(d.Content); i += 2 {
		if r, ok := rank[d.Content[i].Value]; ok {
			last = r
		}
		pairs = append(pairs, pair{d.Content[i], d.Content[i+1], last})
	}
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].r < pairs[j].r })
	head := ""
	if len(d.Content) > 0 {
		head, d.Content[0].HeadComment = d.Content[0].HeadComment, ""
	}
	d.Content = d.Content[:0]
	for _, p := range pairs {
		d.Content = append(d.Content, p.k, p.v)
	}
	if len(d.Content) > 0 && d.Content[0].HeadComment == "" {
		d.Content[0].HeadComment = head
	}
}

func unwrap(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

func mapIndex(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func mapGet(m *yaml.Node, key string) *yaml.Node {
	if i := mapIndex(m, key); i >= 0 {
		return m.Content[i+1]
	}
	return nil
}

func keep(newer, older string) string {
	if newer != "" {
		return newer
	}
	return older
}
