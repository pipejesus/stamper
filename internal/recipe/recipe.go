// Package recipe defines the pluggable recipe system. A recipe declares its
// options once; the CLI turns them into flags, the TUI into forms, and the
// recipe turns chosen values into a Landofile plus support files.
package recipe

import (
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pipejesus/stamper/internal/lando"

	"go.yaml.in/yaml/v3"
)

// Kind is the input type of an option.
type Kind int

const (
	String Kind = iota // free text
	Select             // exactly one of Choices
	Multi              // any subset of Choices (stored comma-separated)
	Bool               // "true" / "false"
)

// Choice is one allowed value of a Select or Multi option.
type Choice struct {
	Value string
	Label string // human label; Value is used when empty
	Note  string // short hint shown next to the label (e.g. "experimental")
}

// Display renders the choice for menus.
func (c Choice) Display() string {
	l := c.Label
	if l == "" {
		l = c.Value
	}
	if c.Note != "" {
		l += "  (" + c.Note + ")"
	}
	return l
}

// Option is a single configurable setting of a recipe.
type Option struct {
	Key      string // flag name and config key, kebab-case
	Short    string // optional one-letter flag shorthand
	Title    string
	Help     string
	Group    string // TUI page title; options sharing it render together
	Kind     Kind
	Choices  []Choice
	Default  string
	Required bool
	// Validate checks a single (non-empty) value beyond choice membership.
	Validate func(string) error
	// Visible hides the option in the TUI when false (values still validate).
	Visible func(Values) bool
	// ChangeWarning is shown when `stamper config` changes this option on an
	// existing environment (e.g. database engine switches need a fresh volume).
	ChangeWarning string
}

// Values holds chosen option values keyed by Option.Key.
type Values map[string]string

// Get returns the raw value.
func (v Values) Get(k string) string { return v[k] }

// Bool interprets a value as boolean.
func (v Values) Bool(k string) bool {
	switch strings.ToLower(v[k]) {
	case "1", "true", "yes", "y", "on":
		return true
	}
	return false
}

// List splits a Multi value.
func (v Values) List(k string) []string {
	var out []string
	for _, s := range strings.Split(v[k], ",") {
		if s = strings.TrimSpace(s); s != "" && s != "none" {
			out = append(out, s)
		}
	}
	return out
}

// Has reports whether Multi option k contains item.
func (v Values) Has(k, item string) bool { return slices.Contains(v.List(k), item) }

// Clone returns a shallow copy.
func (v Values) Clone() Values {
	c := make(Values, len(v))
	for k, x := range v {
		c[k] = x
	}
	return c
}

// File is a support file a recipe writes next to the Landofile.
type File struct {
	Path    string // relative to the project dir
	Content []byte
	Mode    fs.FileMode
	Policy  Policy
}

// ManagedMarker must appear in every file stamper may overwrite or remove
// outside its own .lando/stamper/ directory.
const ManagedMarker = "Managed by stamper"

// Policy decides what happens when a support file already exists.
type Policy int

const (
	Always    Policy = iota // stamper's own files: always rewritten
	IfManaged               // rewritten only while it still carries ManagedMarker
	Once                    // created if missing, then owned by the user
)

// Plan is everything needed to materialise (or update) an environment.
type Plan struct {
	Landofile *yaml.Node    // managed top-level mapping
	Owned     []lando.Owned // parts of the Landofile this recipe controls
	Files     []File        // support files, always rewritten
	Remove    []string      // support files to delete if present
	Warnings  []string
	Notes     []string // shown after a successful stamp
}

// Recipe is implemented by every environment type stamper can produce.
type Recipe interface {
	Name() string
	Summary() string
	Options() []Option
	// Normalize fills defaults and derives values (e.g. app name from URL).
	Normalize(Values) Values
	// Check validates cross-option constraints after per-option validation.
	Check(Values) error
	// Plan renders the environment for the given (normalized, valid) values.
	Plan(Values) (*Plan, error)
	// Read recovers values from an existing Landofile, for `stamper config`
	// on projects stamped without state or edited by hand.
	Read(*lando.File) Values
	// Owns reports whether an existing Landofile looks like this recipe's.
	Owns(*lando.File) bool
}

// Revisioned is implemented by recipes that track when their knowledge of
// upstream versions (Lando images, PHP, databases) was last refreshed.
// Projects remember the revision they were stamped with, so stamper can
// suggest `stamper config upgrade` after the recipe improves.
type Revisioned interface {
	Revision() time.Time
}

// RevisionOf returns r's revision, or the zero time.
func RevisionOf(r Recipe) time.Time {
	if rv, ok := r.(Revisioned); ok {
		return rv.Revision()
	}
	return time.Time{}
}

// Defaults returns every option's default value.
func Defaults(r Recipe) Values {
	v := Values{}
	for _, o := range r.Options() {
		v[o.Key] = o.Default
	}
	return v
}

// Find returns the option with key k.
func Find(r Recipe, k string) (Option, bool) {
	for _, o := range r.Options() {
		if o.Key == k {
			return o, true
		}
	}
	return Option{}, false
}

// Validate checks each value against its option, then the recipe's
// cross-option rules. It reports all problems at once.
func Validate(r Recipe, v Values) error {
	var errs []string
	for _, o := range r.Options() {
		val := strings.TrimSpace(v[o.Key])
		if val == "" {
			if o.Required {
				errs = append(errs, fmt.Sprintf("--%s is required", o.Key))
			}
			continue
		}
		if err := checkOption(o, val); err != nil {
			errs = append(errs, fmt.Sprintf("--%s: %v", o.Key, err))
		}
	}
	for k := range v {
		if _, ok := Find(r, k); !ok {
			errs = append(errs, fmt.Sprintf("unknown option %q for recipe %s", k, r.Name()))
		}
	}
	if len(errs) == 0 {
		if err := r.Check(v); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func checkOption(o Option, val string) error {
	allowed := func(s string) bool {
		return slices.ContainsFunc(o.Choices, func(c Choice) bool { return c.Value == s })
	}
	switch o.Kind {
	case Select:
		if !allowed(val) {
			return fmt.Errorf("%q is not one of: %s", val, ChoiceList(o))
		}
	case Multi:
		for _, s := range strings.Split(val, ",") {
			if s = strings.TrimSpace(s); s != "" && s != "none" && !allowed(s) {
				return fmt.Errorf("%q is not one of: %s", s, ChoiceList(o))
			}
		}
	case Bool:
		switch strings.ToLower(val) {
		case "1", "0", "true", "false", "yes", "no", "y", "n", "on", "off":
		default:
			return fmt.Errorf("%q is not a boolean", val)
		}
	}
	if o.Validate != nil {
		return o.Validate(val)
	}
	return nil
}

// ChoiceList renders allowed values for help text and errors.
func ChoiceList(o Option) string {
	vals := make([]string, len(o.Choices))
	for i, c := range o.Choices {
		vals[i] = c.Value
	}
	return strings.Join(vals, ", ")
}
