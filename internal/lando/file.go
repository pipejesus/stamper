package lando

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// FileName is the Landofile stamper manages.
const FileName = ".lando.yml"

// Header is placed at the top of newly stamped Landofiles.
const Header = `Stamped by stamper. Change it with "stamper config" or edit by hand:
stamper only rewrites the keys it manages and keeps everything else.`

// File is a parsed Landofile.
type File struct {
	Path string
	Doc  *yaml.Node // DocumentNode
}

// ErrNoLandofile is returned when dir has no .lando.yml.
var ErrNoLandofile = errors.New("no " + FileName + " found")

// Load reads dir/.lando.yml.
func Load(dir string) (*File, error) {
	p := filepath.Join(dir, FileName)
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w in %s", ErrNoLandofile, dir)
	}
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	if doc.Kind == 0 { // empty file
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{Map()}}
	}
	if unwrap(&doc).Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: top level must be a mapping", p)
	}
	return &File{Path: p, Doc: &doc}, nil
}

// New returns an in-memory Landofile for dir with root as content.
func New(dir string, root *yaml.Node) *File {
	root.HeadComment = Header
	return &File{
		Path: filepath.Join(dir, FileName),
		Doc:  &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}},
	}
}

// Root returns the top-level mapping.
func (f *File) Root() *yaml.Node { return unwrap(f.Doc) }

// Bytes encodes the Landofile with 2-space indentation.
func (f *File) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(f.Doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Save writes the Landofile atomically.
func (f *File) Save() error {
	data, err := f.Bytes()
	if err != nil {
		return err
	}
	return WriteAtomic(f.Path, data, 0o644)
}

// WriteAtomic writes data via a temp file + rename so a crash never leaves
// a half-written config behind.
func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Installed reports whether the lando binary is on PATH.
func Installed() bool {
	_, err := exec.LookPath("lando")
	return err == nil
}

// Run executes `lando args...` in dir, streaming output to the terminal.
func Run(dir string, args ...string) error {
	cmd := exec.Command("lando", args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// Version returns `lando version` output, trimmed.
func Version() (string, error) {
	out, err := exec.Command("lando", "version").Output()
	return string(bytes.TrimSpace(out)), err
}

// PluginInstalled reports whether a Lando plugin such as "@lando/mailpit"
// exists in the user's plugin directory (~/.lando/plugins).
func PluginInstalled(name string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".lando", "plugins", name))
	return err == nil
}
