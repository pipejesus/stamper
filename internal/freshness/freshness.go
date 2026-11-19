// Package freshness runs stamper's boot-time checks: how old the bundled
// skills and recipe knowledge are, whether the current project has stale
// skills or a Landofile rendered by an older recipe, and whether a newer
// stamper release exists. It is offline except for a cached daily release
// lookup.
package freshness

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pipejesus/stamper/internal/project"
	"github.com/pipejesus/stamper/internal/recipe"
	"github.com/pipejesus/stamper/internal/selfupdate"
	"github.com/pipejesus/stamper/internal/skills"
)

// Thresholds after which bundled knowledge is considered stale.
const (
	SkillsMaxAge = 90 * 24 * time.Hour
	RecipeMaxAge = 120 * 24 * time.Hour
)

// Hint is one suggestion shown at boot.
type Hint struct {
	Message string
	Command string
}

// Check gathers hints for the project in dir. version is the running build.
func Check(dir, version string, now time.Time) []Hint {
	var hints []Hint
	days := func(t time.Time) int { return int(now.Sub(t).Hours() / 24) }

	if latest := selfupdate.CachedLatest(now); latest != "" && selfupdate.Newer(latest, version) {
		hints = append(hints, Hint{fmt.Sprintf("stamper %s is available (you run %s)", latest, version), "stamper self-update"})
	}

	if t := skills.LastUpdated(); !t.IsZero() && now.Sub(t) > SkillsMaxAge {
		hints = append(hints, Hint{fmt.Sprintf("bundled skills were last researched %d days ago (%s)", days(t), t.Format(time.DateOnly)), "stamper self-update"})
	}
	for _, rc := range recipe.All() {
		if rev := recipe.RevisionOf(rc); !rev.IsZero() && now.Sub(rev) > RecipeMaxAge {
			hints = append(hints, Hint{fmt.Sprintf("the %s recipe's Lando/PHP/DB version tables are %d days old", rc.Name(), days(rev)), "stamper self-update"})
		}
	}

	// Installed skills in this project and in $HOME.
	bundled, _ := skills.All()
	roots := []string{dir}
	if home, err := os.UserHomeDir(); err == nil && home != dir {
		roots = append(roots, home)
	}
	for _, root := range roots {
		for _, a := range skills.Agents {
			skDir := filepath.Join(root, a.Dir)
			for _, s := range bundled {
				if skills.Check(skDir, s) == skills.Outdated {
					cmd := "stamper skills update"
					if root != dir {
						cmd += " --global"
					}
					hints = append(hints, Hint{fmt.Sprintf("skill %s in %s has a newer bundled version", s.Name, skDir), cmd})
				}
			}
		}
	}

	// Landofile rendered by an older recipe revision.
	if p, err := project.Open(dir); err == nil && p.State != nil {
		if rc, err := recipe.Get(p.State.Recipe); err == nil {
			rev := recipe.RevisionOf(rc)
			stamped, err := time.Parse(time.RFC3339, p.State.Revision)
			if err != nil {
				stamped, _ = time.Parse(time.DateOnly, p.State.Revision)
			}
			if !rev.IsZero() && stamped.Before(rev) {
				hints = append(hints, Hint{fmt.Sprintf("this project's Lando config was rendered by an older %s recipe (%s → %s)", rc.Name(), orUnknown(p.State.Revision), rev.UTC().Format(time.RFC3339)), "stamper config upgrade"})
			}
		}
	}
	return hints
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
