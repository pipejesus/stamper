// Package project ties a directory, its Landofile, stamper's saved state and
// a recipe together, and applies recipe plans to disk.
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pipejesus/stamper/internal/lando"
	"github.com/pipejesus/stamper/internal/recipe"

	"go.yaml.in/yaml/v3"
)

// StateFile records which recipe and values stamped the project, relative
// to the project dir. It is the source of truth for `stamper config`.
const StateFile = ".lando/stamper/state.yml"

// State is the persisted stamper metadata.
type State struct {
	Stamper string `yaml:"stamper"`
	Recipe  string `yaml:"recipe"`
	// Revision is the recipe revision (YYYY-MM-DD) the config was rendered with.
	Revision string            `yaml:"revision,omitempty"`
	Values   map[string]string `yaml:"values"`
}

// Project is a directory with (or about to have) a stamped Landofile.
type Project struct {
	Dir       string
	Landofile *lando.File // nil before the first stamp
	State     *State      // nil when the project was not stamped by stamper
}

// Open loads whatever exists in dir. Missing files are not an error.
func Open(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	p := &Project{Dir: abs}
	if p.Landofile, err = lando.Load(abs); err != nil && !errors.Is(err, lando.ErrNoLandofile) {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(abs, StateFile))
	switch {
	case err == nil:
		var s State
		if err := yaml.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parse %s: %w", StateFile, err)
		}
		p.State = &s
	case !errors.Is(err, os.ErrNotExist):
		return nil, err
	}
	return p, nil
}

// Recipe resolves the project's recipe from state, else by detection.
func (p *Project) Recipe() (recipe.Recipe, error) {
	if p.State != nil && p.State.Recipe != "" {
		return recipe.Get(p.State.Recipe)
	}
	if p.Landofile == nil {
		return nil, fmt.Errorf("%w in %s — run `stamper new` first", lando.ErrNoLandofile, p.Dir)
	}
	if r, ok := recipe.Detect(p.Landofile); ok {
		return r, nil
	}
	return nil, fmt.Errorf("%s was not created by a known stamper recipe", p.Landofile.Path)
}

// Values returns the current configuration: defaults, overlaid by what the
// Landofile says (it may have been edited by hand), overlaid by saved state
// for everything the Landofile cannot express.
func (p *Project) Values(r recipe.Recipe) recipe.Values {
	v := recipe.Defaults(r)
	if p.State != nil {
		for k, x := range p.State.Values {
			if _, ok := recipe.Find(r, k); ok {
				v[k] = x
			}
		}
	}
	if p.Landofile != nil {
		for k, x := range r.Read(p.Landofile) {
			if x != "" {
				v[k] = x
			}
		}
	}
	return r.Normalize(v)
}

// Result summarises what Apply wrote.
type Result struct {
	Landofile string
	Written   []string
	Removed   []string
}

// Apply writes the plan. On a fresh project the Landofile is the managed
// tree; on an existing one only owned paths are reconciled.
func (p *Project) Apply(r recipe.Recipe, v recipe.Values, plan *recipe.Plan, version string) (*Result, error) {
	if p.Landofile == nil {
		p.Landofile = lando.New(p.Dir, plan.Landofile)
	} else {
		lando.Apply(p.Landofile.Doc, plan.Landofile, plan.Owned)
	}
	if err := p.Landofile.Save(); err != nil {
		return nil, err
	}
	res := &Result{Landofile: p.Landofile.Path}
	for _, f := range plan.Files {
		mode := f.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := lando.WriteAtomic(filepath.Join(p.Dir, f.Path), f.Content, mode); err != nil {
			return nil, err
		}
		res.Written = append(res.Written, f.Path)
	}
	for _, rel := range plan.Remove {
		err := os.Remove(filepath.Join(p.Dir, rel))
		if err == nil {
			res.Removed = append(res.Removed, rel)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	p.State = &State{Stamper: version, Recipe: r.Name(), Values: v}
	if rev := recipe.RevisionOf(r); !rev.IsZero() {
		p.State.Revision = rev.Format(time.DateOnly)
	}
	data, err := yaml.Marshal(p.State)
	if err != nil {
		return nil, err
	}
	if err := lando.WriteAtomic(filepath.Join(p.Dir, StateFile), data, 0o644); err != nil {
		return nil, err
	}
	res.Written = append(res.Written, StateFile)
	sort.Strings(res.Written)
	return res, nil
}

// Diff lists keys whose values differ between a and b.
func Diff(a, b recipe.Values) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range []recipe.Values{a, b} {
		for k := range m {
			if !seen[k] && a[k] != b[k] {
				out = append(out, k)
			}
			seen[k] = true
		}
	}
	sort.Strings(out)
	return out
}
