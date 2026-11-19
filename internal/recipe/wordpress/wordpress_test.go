package wordpress

import (
	"strings"
	"testing"

	"github.com/pipejesus/stamper/internal/lando"
	r "github.com/pipejesus/stamper/internal/recipe"
)

func values(t *testing.T, kv ...string) r.Values {
	t.Helper()
	rc := New()
	v := r.Defaults(rc)
	for i := 0; i < len(kv); i += 2 {
		v[kv[i]] = kv[i+1]
	}
	v = rc.Normalize(v)
	if err := r.Validate(rc, v); err != nil {
		t.Fatal(err)
	}
	return v
}

func renderPlan(t *testing.T, v r.Values) (string, *r.Plan) {
	t.Helper()
	p, err := New().Plan(v)
	if err != nil {
		t.Fatal(err)
	}
	b, err := lando.New(t.TempDir(), p.Landofile).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return string(b), p
}

func TestAppName(t *testing.T) {
	for in, want := range map[string]string{
		"myshop.lndo.site":   "myshop",
		"a.b.lndo.site":      "a-b",
		"shop.example.test":  "shop-example",
		"*.multi.lndo.site":  "multi",
		"plainhost.internal": "plainhost-internal",
	} {
		if got := AppName(in); got != want {
			t.Errorf("AppName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	rc := New()
	v := rc.Normalize(r.Values{"url": "not a host", "php": "9.9"})
	err := r.Validate(rc, v)
	if err == nil || !strings.Contains(err.Error(), "--url") || !strings.Contains(err.Error(), "--php") {
		t.Fatalf("want url and php errors, got %v", err)
	}
	v = rc.Normalize(merge(r.Defaults(rc), r.Values{"url": "x.lndo.site", "webserver": "openlitespeed", "php": "7.4"}))
	if err := r.Validate(rc, v); err == nil {
		t.Fatal("OLS with PHP 7.4 should be rejected")
	}
}

func merge(a, b r.Values) r.Values {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func TestPlanApacheDefaults(t *testing.T) {
	out, p := renderPlan(t, values(t, "url", "myshop.lndo.site"))
	for _, want := range []string{"recipe: wordpress", `php: "8.4"`, "via: apache:2.4", "database: mariadb:11.4", "xdebug: false", "wp-setup.sh"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "proxy:\n  appserver") {
		t.Errorf("default lndo.site host should not need a proxy entry:\n%s", out)
	}
	files := map[string]string{}
	for _, f := range p.Files {
		files[f.Path] = string(f.Content)
	}
	if !strings.Contains(out, "webroot: public_html") {
		t.Errorf("default web root should be public_html:\n%s", out)
	}
	if !strings.Contains(files[wpCLIYml], "path: public_html") {
		t.Errorf("wp-cli.yml missing or wrong: %q", files[wpCLIYml])
	}
	setup := files[setupSh]
	for _, want := range []string{"mv wp-config.php ../wp-config.php", "WP_DEBUG_LOG /app/logs/debug.log", "# BEGIN stamper", "wp-content/uploads/.htaccess"} {
		if !strings.Contains(setup, want) {
			t.Errorf("setup script lacks %q", want)
		}
	}
	if !strings.Contains(files[".gitignore"], "/public_html/*") {
		t.Errorf(".gitignore not tuned for public_html:\n%s", files[".gitignore"])
	}
}

func TestPlanFlatWebroot(t *testing.T) {
	_, p := renderPlan(t, values(t, "url", "flat.lndo.site", "webroot", "."))
	for _, f := range p.Files {
		if f.Path == wpCLIYml {
			t.Error("flat layout must not write wp-cli.yml")
		}
		if f.Path == setupSh && strings.Contains(string(f.Content), "mv wp-config.php") {
			t.Error("flat layout must keep wp-config.php in place")
		}
	}
}

func TestPlanNginxRedisVarnish(t *testing.T) {
	out, p := renderPlan(t, values(t, "url", "shop.lndo.site", "webserver", "nginx", "object-cache", "redis:8", "page-cache", "varnish:6", "php-extensions", "igbinary,pcov", "multisite", "subdirectory"))
	for _, want := range []string{"via: nginx:1.29", "vhosts: .lando/stamper/nginx.conf", "type: redis:8", "appserver_nginx", "install-php-extensions igbinary pcov", "edge.shop.lndo.site"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	var conf, setup string
	for _, f := range p.Files {
		switch f.Path {
		case nginxConf:
			conf = string(f.Content)
		case setupSh:
			setup = string(f.Content)
		}
	}
	if !strings.Contains(conf, "rewrite ^(/[^/]+)?(/wp-.*) $2 last;") {
		t.Error("nginx conf lacks multisite rewrites")
	}
	if !strings.Contains(setup, "wp core multisite-install") || !strings.Contains(setup, "WP_REDIS_HOST redis") {
		t.Errorf("setup script misses multisite install or redis wiring:\n%s", setup)
	}
}

func TestPlanOpenLiteSpeed(t *testing.T) {
	out, _ := renderPlan(t, values(t, "url", "ols.lndo.site", "webserver", "openlitespeed", "php", "8.3", "database", "mysql:8.4"))
	for _, want := range []string{"litespeedtech/openlitespeed:1.9.4-lsphp83", "type: mysql:8.4", "ols.lndo.site:80", "db-import <file>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "recipe: wordpress") {
		t.Error("OLS stack must not use Lando's wordpress recipe")
	}
}

func TestReadRoundTrip(t *testing.T) {
	v := values(t, "url", "rt.lndo.site", "webserver", "nginx", "php", "8.2", "database", "mysql:8.0", "xdebug", "develop,debug")
	p, _ := New().Plan(v)
	f := lando.New(t.TempDir(), p.Landofile)
	got := New().Read(f)
	for _, k := range []string{"webserver", "php", "database", "xdebug", "name"} {
		if got[k] != v[k] {
			t.Errorf("Read %s = %q, want %q", k, got[k], v[k])
		}
	}
	if !New().Owns(f) {
		t.Error("recipe should own its Landofile")
	}
}
