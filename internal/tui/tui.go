// Package tui renders stamper's interactive forms with charmbracelet/huh and
// shared lipgloss styles. Forms are generated from recipe option schemas, so
// new recipes get a TUI for free.
package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/term"

	"github.com/pipejesus/stamper/internal/recipe"
)

// ErrAborted is returned when the user cancels a form.
var ErrAborted = huh.ErrUserAborted

// Interactive reports whether stdin and stdout are terminals.
func Interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

var theme = huh.ThemeFunc(huh.ThemeCharm)

func run(f *huh.Form) error { return f.WithTheme(theme).Run() }

// ---- styles ----------------------------------------------------------------

var (
	accent  = lipgloss.Color("#B57EDC")
	muted   = lipgloss.Color("#8A8A8A")
	okColor = lipgloss.Color("#3FB950")
	warnCol = lipgloss.Color("#D29922")

	Title   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	Muted   = lipgloss.NewStyle().Foreground(muted)
	OK      = lipgloss.NewStyle().Foreground(okColor)
	Warn    = lipgloss.NewStyle().Foreground(warnCol)
	KeyCol  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	Command = lipgloss.NewStyle().Foreground(lipgloss.Color("#58A6FF"))
	Box     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1)
)

// Banner is shown on interactive entry points.
func Banner(version string) string {
	return Box.Render(Title.Render("stamper") + Muted.Render("  "+version+" · local environments, stamped"))
}

// Print writes styled text to w (lipgloss downsamples colours for the terminal).
func Print(w io.Writer, s string) { lipgloss.Fprintln(w, s) }

// KV renders an aligned key/value table.
func KV(rows [][2]string) string {
	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(KeyCol.Render(fmt.Sprintf("%-*s", width, r[0])) + "  " + r[1] + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ---- recipe form -------------------------------------------------------------

// RecipeForm asks for every option of rc, prefilled with v. Options that
// share a Group render on one page; options with a Visible rule get their own
// page so they can appear and disappear as answers change.
func RecipeForm(rc recipe.Recipe, v recipe.Values) (recipe.Values, error) {
	opts := rc.Options()
	strs := map[string]*string{}
	multis := map[string]*[]string{}
	bools := map[string]*bool{}
	// current returns live values while the form runs (for Visible rules).
	current := func() recipe.Values {
		out := v.Clone()
		for k, p := range strs {
			out[k] = *p
		}
		for k, p := range multis {
			out[k] = strings.Join(*p, ",")
		}
		for k, p := range bools {
			out[k] = fmt.Sprint(*p)
		}
		return rc.Normalize(out)
	}

	var groups []*huh.Group
	var page []huh.Field
	var pageTitle string
	flush := func() {
		if len(page) > 0 {
			groups = append(groups, huh.NewGroup(page...).Title(pageTitle))
			page = nil
		}
	}
	for _, o := range opts {
		field := buildField(o, v, strs, multis, bools)
		if o.Visible != nil {
			flush()
			vis := o.Visible
			groups = append(groups, huh.NewGroup(field).Title(o.Group).WithHideFunc(func() bool { return !vis(current()) }))
			pageTitle = ""
			continue
		}
		if o.Group != pageTitle {
			flush()
			pageTitle = o.Group
		}
		page = append(page, field)
	}
	flush()

	if err := run(huh.NewForm(groups...)); err != nil {
		return nil, err
	}
	return current(), nil
}

func buildField(o recipe.Option, v recipe.Values, strs map[string]*string, multis map[string]*[]string, bools map[string]*bool) huh.Field {
	desc := o.Help
	switch o.Kind {
	case recipe.Select:
		val := v.Get(o.Key)
		if val == "" {
			val = o.Default
		}
		strs[o.Key] = &val
		sel := huh.NewSelect[string]().Title(o.Title).Description(desc).Value(strs[o.Key])
		var hopts []huh.Option[string]
		for _, c := range o.Choices {
			hopts = append(hopts, huh.NewOption(c.Display(), c.Value))
		}
		return sel.Options(hopts...).Height(min(len(hopts)+2, 10))
	case recipe.Multi:
		cur := v.List(o.Key)
		multis[o.Key] = &cur
		var hopts []huh.Option[string]
		for _, c := range o.Choices {
			hopts = append(hopts, huh.NewOption(c.Display(), c.Value).Selected(slices.Contains(cur, c.Value)))
		}
		return huh.NewMultiSelect[string]().Title(o.Title).Description(desc).Options(hopts...).Value(multis[o.Key]).Height(min(len(hopts)+2, 12))
	case recipe.Bool:
		b := v.Bool(o.Key)
		if v.Get(o.Key) == "" {
			b = recipe.Values{"x": o.Default}.Bool("x")
		}
		bools[o.Key] = &b
		return huh.NewConfirm().Title(o.Title).Description(desc).Value(bools[o.Key])
	default:
		val := v.Get(o.Key)
		strs[o.Key] = &val
		in := huh.NewInput().Title(o.Title).Description(desc).Value(strs[o.Key])
		if o.Default != "" {
			in = in.Placeholder(o.Default)
		}
		return in.Validate(func(s string) error {
			s = strings.TrimSpace(s)
			if s == "" {
				if o.Required {
					return errors.New("required")
				}
				return nil
			}
			if o.Validate != nil {
				return o.Validate(s)
			}
			return nil
		})
	}
}

// ---- generic prompts -----------------------------------------------------------

// Choose shows a single-select menu of label→value pairs (in order).
func Choose(title string, items [][2]string) (string, error) {
	var out string
	opts := make([]huh.Option[string], len(items))
	for i, it := range items {
		opts[i] = huh.NewOption(it[0], it[1])
	}
	err := run(huh.NewForm(huh.NewGroup(huh.NewSelect[string]().Title(title).Options(opts...).Value(&out))))
	return out, err
}

// ChooseMany shows a multi-select; preselected values are checked.
func ChooseMany(title, desc string, items [][2]string, preselected []string) ([]string, error) {
	out := slices.Clone(preselected)
	opts := make([]huh.Option[string], len(items))
	for i, it := range items {
		opts[i] = huh.NewOption(it[0], it[1]).Selected(slices.Contains(preselected, it[1]))
	}
	err := run(huh.NewForm(huh.NewGroup(huh.NewMultiSelect[string]().Title(title).Description(desc).Options(opts...).Value(&out))))
	return out, err
}

// Confirm asks a yes/no question.
func Confirm(title, desc string, def bool) (bool, error) {
	ok := def
	err := run(huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(title).Description(desc).Value(&ok))))
	return ok, err
}

// Input asks for one line of text.
func Input(title, desc, def string) (string, error) {
	val := def
	err := run(huh.NewForm(huh.NewGroup(huh.NewInput().Title(title).Description(desc).Value(&val))))
	return strings.TrimSpace(val), err
}

// Summary renders values in option order, skipping hidden options.
func Summary(rc recipe.Recipe, v recipe.Values, changed []string) string {
	var rows [][2]string
	for _, o := range rc.Options() {
		if o.Visible != nil && !o.Visible(v) {
			continue
		}
		val := v.Get(o.Key)
		if val == "" {
			val = Muted.Render("—")
		}
		if slices.Contains(changed, o.Key) {
			val = Warn.Render(val + "  ← changed")
		}
		rows = append(rows, [2]string{o.Key, val})
	}
	return KV(rows)
}

// SortedKeys is a small helper for stable output.
func SortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
