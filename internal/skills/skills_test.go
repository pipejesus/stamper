package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLifecycle(t *testing.T) {
	all, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Skip("no bundled skills yet")
	}
	s := all[0]
	if s.Description == "" {
		t.Errorf("%s: SKILL.md frontmatter has no description", s.Name)
	}
	dir := t.TempDir()

	if st := Check(dir, s); st != Missing {
		t.Fatalf("fresh dir: %v", st)
	}
	if _, err := Install(dir, s, false); err != nil {
		t.Fatal(err)
	}
	if st := Check(dir, s); st != Current {
		t.Fatalf("after install: %v", st)
	}

	// Local edits are detected and protected.
	skillMD := filepath.Join(dir, s.Name, "SKILL.md")
	if err := os.WriteFile(skillMD, []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := Check(dir, s); st != Modified {
		t.Fatalf("after edit: %v", st)
	}
	if _, err := Install(dir, s, false); err == nil {
		t.Fatal("install over a modified skill must need --force")
	}
	if _, err := Install(dir, s, true); err != nil {
		t.Fatal(err)
	}

	// Simulate an older stamper release having installed it.
	stale := *s0(s)
	stale.Hash = "0000000000000000"
	if st := Check(dir, stale); st != Outdated {
		t.Fatalf("hash mismatch should read as outdated, got %v", st)
	}

	if err := Uninstall(dir, s, false); err != nil {
		t.Fatal(err)
	}
	if st := Check(dir, s); st != Missing {
		t.Fatalf("after uninstall: %v", st)
	}
}

func s0(s Skill) *Skill { return &s }

func TestForeignSkillIsProtected(t *testing.T) {
	all, _ := All()
	if len(all) == 0 {
		t.Skip("no bundled skills yet")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, all[0].Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if st := Check(dir, all[0]); st != Foreign {
		t.Fatalf("got %v", st)
	}
	if _, err := Install(dir, all[0], false); err == nil {
		t.Fatal("must not overwrite a foreign skill without force")
	}
}
