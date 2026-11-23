// Package wordpress is stamper's WordPress recipe: Lando's wordpress recipe
// for Apache/Nginx, a custom OpenLiteSpeed stack, plus caches, tools and an
// idempotent WordPress bootstrap.
package wordpress

import (
	"fmt"
	"strings"
	"time"

	"github.com/pipejesus/stamper/internal/lando"
	r "github.com/pipejesus/stamper/internal/recipe"
)

// Recipe implements recipe.Recipe.
type Recipe struct{}

// New returns the WordPress recipe.
func New() *Recipe { return &Recipe{} }

func (*Recipe) Name() string { return "wordpress" }

// revision is when this recipe's version tables (Lando 3.26 / @lando/php 1.12,
// MariaDB, MySQL, Redis, OLS images) or generated files last changed.
// Bump it (date + time) with every such change: projects stamped earlier get
// a `stamper config upgrade` hint.
var revision = time.Date(2026, 11, 23, 11, 45, 0, 0, time.UTC)

// Revision implements recipe.Revisioned.
func (*Recipe) Revision() time.Time { return revision }

func (*Recipe) Summary() string {
	return "WordPress on Apache, Nginx or OpenLiteSpeed with MariaDB/MySQL, object & page caches, mail and dev tools"
}

func (*Recipe) Options() []r.Option { return options() }

// domainSuffixes are stripped when deriving the app name from the URL.
var domainSuffixes = []string{".lndo.site", ".lndo.dev", ".test", ".localhost", ".local", ".site"}

// AppName derives a Lando app name from a hostname.
func AppName(host string) string {
	h := strings.TrimPrefix(strings.ToLower(host), "*.")
	for _, s := range domainSuffixes {
		if strings.HasSuffix(h, s) && len(h) > len(s) {
			h = strings.TrimSuffix(h, s)
			break
		}
	}
	return strings.ReplaceAll(h, ".", "-")
}

func (*Recipe) Normalize(in r.Values) r.Values {
	v := in.Clone()
	v[kURL] = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v[kURL])), "https://"), "http://"), "/")
	if v[kName] == "" && v[kURL] != "" {
		v[kName] = AppName(v[kURL])
	}
	if v[kAdminEmail] == "" && v[kURL] != "" {
		v[kAdminEmail] = "admin@" + v[kURL]
	}
	v[kWebroot] = strings.Trim(strings.TrimPrefix(v[kWebroot], "./"), "/")
	if v[kWebroot] == "" {
		v[kWebroot] = "."
	}
	v[kMemory], v[kUpload] = strings.ToUpper(v[kMemory]), strings.ToUpper(v[kUpload])
	return v
}

func (*Recipe) Check(v r.Values) error {
	var errs []string
	if v.Get(kWebserver) == ols {
		if !olsPHP[v.Get(kPHP)] {
			errs = append(errs, fmt.Sprintf("OpenLiteSpeed images exist for PHP 8.2–8.5, not %s", v.Get(kPHP)))
		}
	}
	if v.Get(kMultisite) != "no" && v.Get(kInstall) != "" && !v.Bool(kInstall) {
		errs = append(errs, "multisite is configured by the WordPress installer; enable --wp-install or convert the site yourself with `lando wp core multisite-convert`")
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// Owns accepts Landofiles using Lando's wordpress recipe or stamper's OLS stack.
func (*Recipe) Owns(f *lando.File) bool {
	root := f.Root()
	if lando.Scalar(root, "recipe") == "wordpress" {
		return true
	}
	img := lando.Scalar(root, "services", "appserver", "services", "image")
	return strings.Contains(img, "openlitespeed")
}

// Read recovers what the Landofile itself expresses; stamper's state file
// provides the rest.
func (*Recipe) Read(f *lando.File) r.Values {
	root := f.Root()
	v := r.Values{kName: lando.Scalar(root, "name")}
	if lando.Scalar(root, "recipe") == "wordpress" {
		via := lando.Scalar(root, "config", "via")
		switch {
		case strings.HasPrefix(via, "nginx"):
			v[kWebserver] = nginx
		case via != "":
			v[kWebserver] = apache
		}
		v[kPHP] = lando.Scalar(root, "config", "php")
		v[kDatabase] = lando.Scalar(root, "config", "database")
		v[kWebroot] = lando.Scalar(root, "config", "webroot")
		switch x := lando.Scalar(root, "config", "xdebug"); x {
		case "", "false":
			v[kXdebug] = "off"
		case "true":
			v[kXdebug] = "debug"
		default:
			v[kXdebug] = strings.ReplaceAll(x, " ", "")
		}
	} else if img := lando.Scalar(root, "services", "appserver", "services", "image"); strings.Contains(img, "openlitespeed") {
		v[kWebserver] = ols
		if i := strings.Index(img, "lsphp"); i >= 0 && len(img) >= i+7 {
			d := img[i+5 : i+7]
			v[kPHP] = d[:1] + "." + d[1:]
		}
		v[kDatabase] = lando.Scalar(root, "services", "database", "type")
	}
	return v
}
