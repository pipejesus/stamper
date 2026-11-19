package lando

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func parse(t *testing.T, s string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(s), &n); err != nil {
		t.Fatal(err)
	}
	return &n
}

func encode(t *testing.T, n *yaml.Node) string {
	t.Helper()
	f := &File{Doc: n}
	b, err := f.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestApplyKeepsUserContent(t *testing.T) {
	doc := parse(t, `
name: app
recipe: wordpress
config:
  php: "8.2"
  mine: keep # user comment
services:
  appserver:
    build_as_root:
      - apt-get install -y less
      - install-php-extensions old # stamper
  redis:
    type: redis:7
  custom:
    type: node:22
events:
  post-start:
    - appserver: echo hi
    - appserver: bash /app/.lando/stamper/wp-setup.sh
`)
	managed := Map(
		"name", "app",
		"recipe", "wordpress",
		"config", Map("php", "8.4"),
		"services", Map("appserver", Map("build_as_root", Seq(Tagged("install-php-extensions new")))),
		"events", Map("post-start", Seq(Map("appserver", "bash /app/.lando/stamper/wp-setup.sh"))),
	)
	owned := []Owned{
		{Path: Path{"name"}}, {Path: Path{"recipe"}}, {Path: Path{"config", "php"}},
		{Path: Path{"services", "appserver", "build_as_root"}, Items: true},
		{Path: Path{"services", "redis"}},
		{Path: Path{"events", "post-start"}, Items: true},
	}
	Apply(doc, managed, owned)
	out := encode(t, doc)

	for _, want := range []string{
		`php: "8.4"`, "mine: keep # user comment", "apt-get install -y less",
		"install-php-extensions new # stamper", "custom:", "appserver: echo hi",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, gone := range []string{"redis", "install-php-extensions old"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q should be removed:\n%s", gone, out)
		}
	}
	if n := strings.Count(out, "wp-setup.sh"); n != 1 {
		t.Errorf("setup event appears %d times, want 1:\n%s", n, out)
	}
}

func TestDeletePrunesEmptyParents(t *testing.T) {
	doc := parse(t, "services:\n  redis:\n    type: redis:8\nname: x\n")
	Delete(doc, Path{"services", "redis"})
	if out := encode(t, doc); strings.Contains(out, "services") {
		t.Errorf("empty services should be pruned:\n%s", out)
	}
}

func TestNodeQuotesNumericStrings(t *testing.T) {
	out := encode(t, &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{Map("php", "8.4", "on", "true", "x", true)}})
	if !strings.Contains(out, `php: "8.4"`) || !strings.Contains(out, `on: "true"`) || !strings.Contains(out, "x: true") {
		t.Errorf("unexpected quoting:\n%s", out)
	}
}
