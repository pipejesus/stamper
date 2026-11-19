// Package skills ships stamper's agent skills (SKILL.md format, read by both
// Claude Code and OpenAI Codex) and installs them into projects or $HOME.
package skills

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

//go:embed all:library
var library embed.FS

// Agent is a coding agent that reads skills from a known directory.
type Agent struct {
	Key   string // flag value
	Label string
	Dir   string // relative to the project root or $HOME
}

// Agents supported out of the box.
var Agents = []Agent{
	{Key: "claude", Label: "Claude Code", Dir: ".claude/skills"},
	{Key: "codex", Label: "OpenAI Codex (and other agents reading .agents/skills)", Dir: ".agents/skills"},
}

// AgentByKey looks up an agent.
func AgentByKey(k string) (Agent, error) {
	for _, a := range Agents {
		if a.Key == k {
			return a, nil
		}
	}
	return Agent{}, fmt.Errorf("unknown agent %q (use: claude, codex)", k)
}

// Skill is one bundled skill.
type Skill struct {
	Name        string
	Description string
	Updated     time.Time // when its content was last researched
	Files       []string  // paths relative to the skill dir
	Hash        string    // content hash of all files
}

// MarkerFile is written into each installed skill to track its origin.
const MarkerFile = ".stamper-skill.json"

type marker struct {
	Skill   string `json:"skill"`
	Updated string `json:"updated"`
	Hash    string `json:"hash"`
}

type manifest struct {
	Skills map[string]struct {
		Updated string `yaml:"updated"`
	} `yaml:"skills"`
}

// All returns the bundled skills sorted by name.
func All() ([]Skill, error) {
	var mf manifest
	if data, err := library.ReadFile("library/manifest.yml"); err == nil {
		if err := yaml.Unmarshal(data, &mf); err != nil {
			return nil, fmt.Errorf("skills manifest: %w", err)
		}
	}
	entries, err := library.ReadDir("library")
	if err != nil {
		return nil, err
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		root := "library/" + e.Name()
		if _, err := fs.Stat(library, root+"/SKILL.md"); err != nil {
			continue // work in progress, not a skill yet
		}
		s := Skill{Name: e.Name()}
		err := fs.WalkDir(library, root, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				s.Files = append(s.Files, strings.TrimPrefix(p, root+"/"))
			}
			return err
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(s.Files)
		h := sha256.New()
		for _, rel := range s.Files {
			data, err := library.ReadFile(root + "/" + rel)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(h, "%s\x00%d\x00", rel, len(data))
			h.Write(data)
			if rel == "SKILL.md" {
				s.Description = frontmatter(data)["description"]
			}
		}
		s.Hash = hex.EncodeToString(h.Sum(nil))[:16]
		if m, ok := mf.Skills[s.Name]; ok {
			s.Updated, _ = time.Parse(time.DateOnly, m.Updated)
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Get returns a bundled skill by name.
func Get(name string) (Skill, error) {
	all, err := All()
	if err != nil {
		return Skill{}, err
	}
	for _, s := range all {
		if s.Name == name {
			return s, nil
		}
	}
	return Skill{}, fmt.Errorf("unknown skill %q", name)
}

// LastUpdated is the newest Updated date across bundled skills.
func LastUpdated() time.Time {
	all, _ := All()
	var t time.Time
	for _, s := range all {
		if s.Updated.After(t) {
			t = s.Updated
		}
	}
	return t
}

// Status of a skill at a target location.
type Status int

const (
	Missing  Status = iota
	Current         // identical to the bundled version
	Outdated        // installed by stamper from older content, untouched since
	Modified        // installed by stamper, then edited locally
	Foreign         // a same-named skill stamper did not install
)

func (s Status) String() string {
	return [...]string{"not installed", "up to date", "update available", "locally modified", "not managed by stamper"}[s]
}

// Check inspects dir (an agent skills directory) for skill s.
func Check(dir string, s Skill) Status {
	sd := filepath.Join(dir, s.Name)
	if _, err := os.Stat(sd); errors.Is(err, os.ErrNotExist) {
		return Missing
	}
	data, err := os.ReadFile(filepath.Join(sd, MarkerFile))
	if err != nil {
		return Foreign
	}
	var m marker
	if json.Unmarshal(data, &m) != nil {
		return Foreign
	}
	disk, err := hashDir(sd)
	if err != nil || disk != m.Hash {
		return Modified
	}
	if m.Hash != s.Hash {
		return Outdated
	}
	return Current
}

// Install writes skill s into dir/<name>. Without force it refuses to
// overwrite skills that were edited locally or not installed by stamper.
func Install(dir string, s Skill, force bool) (Status, error) {
	st := Check(dir, s)
	if !force && (st == Modified || st == Foreign) {
		return st, fmt.Errorf("%s: %s in %s (use --force to overwrite)", s.Name, st, dir)
	}
	if st == Current {
		return st, nil
	}
	sd := filepath.Join(dir, s.Name)
	tmp := sd + ".stamper-tmp"
	_ = os.RemoveAll(tmp)
	for _, rel := range s.Files {
		data, err := library.ReadFile(path.Join("library", s.Name, rel))
		if err != nil {
			return st, err
		}
		dst := filepath.Join(tmp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return st, err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return st, err
		}
	}
	m, _ := json.MarshalIndent(marker{Skill: s.Name, Updated: s.Updated.Format(time.DateOnly), Hash: s.Hash}, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, MarkerFile), append(m, '\n'), 0o644); err != nil {
		return st, err
	}
	if err := os.RemoveAll(sd); err != nil {
		return st, err
	}
	return st, os.Rename(tmp, sd)
}

// Uninstall removes a stamper-installed skill.
func Uninstall(dir string, s Skill, force bool) error {
	st := Check(dir, s)
	switch {
	case st == Missing:
		return nil
	case !force && (st == Foreign || st == Modified):
		return fmt.Errorf("%s: %s (use --force to remove)", s.Name, st)
	}
	return os.RemoveAll(filepath.Join(dir, s.Name))
}

// hashDir hashes an installed skill the same way All hashes bundled ones.
func hashDir(dir string) (string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel = filepath.ToSlash(rel); rel != MarkerFile {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files) // same ordering as All
	h := sha256.New()
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

// frontmatter parses the simple `key: value` YAML header of a SKILL.md.
func frontmatter(data []byte) map[string]string {
	out := map[string]string{}
	if !bytes.HasPrefix(data, []byte("---")) {
		return out
	}
	end := bytes.Index(data[3:], []byte("\n---"))
	if end < 0 {
		return out
	}
	var m map[string]any
	if yaml.Unmarshal(data[3:3+end], &m) == nil {
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = strings.TrimSpace(s)
			}
		}
		return out
	}
	sc := bufio.NewScanner(bytes.NewReader(data[3 : 3+end]))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), ":"); ok {
			out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return out
}
