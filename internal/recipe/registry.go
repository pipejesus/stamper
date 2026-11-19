package recipe

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pipejesus/stamper/internal/lando"
)

var (
	mu       sync.RWMutex
	registry = map[string]Recipe{}
)

// Register adds a recipe. Recipes call it from their package's init or the
// CLI wires them explicitly; registering a name twice panics.
func Register(r Recipe) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[r.Name()]; dup {
		panic("recipe: duplicate registration of " + r.Name())
	}
	registry[r.Name()] = r
}

// Get returns a registered recipe by name.
func Get(name string) (Recipe, error) {
	mu.RLock()
	defer mu.RUnlock()
	if r, ok := registry[name]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("unknown recipe %q", name)
}

// All returns registered recipes sorted by name.
func All() []Recipe {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Recipe, 0, len(registry))
	for _, r := range registry {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Detect finds the recipe that owns an existing Landofile.
func Detect(f *lando.File) (Recipe, bool) {
	for _, r := range All() {
		if r.Owns(f) {
			return r, true
		}
	}
	return nil, false
}
