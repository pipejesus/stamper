package wordpress

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/pipejesus/stamper/internal/lando"
	r "github.com/pipejesus/stamper/internal/recipe"

	"go.yaml.in/yaml/v3"
)

//go:embed templates/*
var tmplFS embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"join": strings.Join,
	// sh single-quotes a value for safe use in bash.
	"sh": func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" },
	"v":  func(v r.Values, k string) string { return v[k] },
}).ParseFS(tmplFS, "templates/*"))

// Support files live under one directory so they are easy to spot and ignore.
const (
	supportDir = ".lando/stamper"
	phpIni     = supportDir + "/php.ini"
	nginxConf  = supportDir + "/nginx.conf"
	setupSh    = supportDir + "/wp-setup.sh"
	setupCmd   = "bash /app/" + setupSh
	wpCLIYml   = "wp-cli.yml"
)

// owned lists every Landofile location this recipe may write. Locations a
// given configuration does not produce are removed on `stamper config`, so
// switching e.g. Apache → OpenLiteSpeed or Redis → none cleans up after itself.
var owned = []lando.Owned{
	{Path: lando.Path{"name"}},
	{Path: lando.Path{"recipe"}},
	{Path: lando.Path{"config", "php"}},
	{Path: lando.Path{"config", "via"}},
	{Path: lando.Path{"config", "webroot"}},
	{Path: lando.Path{"config", "database"}},
	{Path: lando.Path{"config", "xdebug"}},
	{Path: lando.Path{"config", "config", "php"}},
	{Path: lando.Path{"config", "config", "vhosts"}},
	{Path: lando.Path{"proxy", "appserver"}, Items: true},
	{Path: lando.Path{"proxy", "appserver_nginx"}, Items: true},
	{Path: lando.Path{"proxy", "edge"}},
	{Path: lando.Path{"proxy", "mailpit"}},
	{Path: lando.Path{"proxy", "mailhog"}},
	{Path: lando.Path{"proxy", "pma"}},
	{Path: lando.Path{"services", "appserver", "type"}},
	{Path: lando.Path{"services", "appserver", "ssl"}},
	{Path: lando.Path{"services", "appserver", "services"}},
	{Path: lando.Path{"services", "appserver", "build_as_root"}, Items: true},
	{Path: lando.Path{"services", "database", "type"}},
	{Path: lando.Path{"services", "database", "portforward"}},
	{Path: lando.Path{"services", "database", "creds"}},
	{Path: lando.Path{"services", "redis"}},
	{Path: lando.Path{"services", "valkey"}},
	{Path: lando.Path{"services", "memcached"}},
	{Path: lando.Path{"services", "edge"}},
	{Path: lando.Path{"services", "mailpit"}},
	{Path: lando.Path{"services", "mailhog"}},
	{Path: lando.Path{"services", "pma"}},
	{Path: lando.Path{"services", "node"}},
	{Path: lando.Path{"tooling", "wp"}},
	{Path: lando.Path{"tooling", "php"}},
	{Path: lando.Path{"tooling", "composer"}},
	{Path: lando.Path{"tooling", "db-import <file>"}},
	{Path: lando.Path{"tooling", "db-export [file]"}},
	{Path: lando.Path{"tooling", "mysql"}},
	{Path: lando.Path{"tooling", "mariadb"}},
	{Path: lando.Path{"tooling", "redis-cli"}},
	{Path: lando.Path{"tooling", "valkey-cli"}},
	{Path: lando.Path{"tooling", "varnishadm"}},
	{Path: lando.Path{"tooling", "npm"}},
	{Path: lando.Path{"tooling", "npx"}},
	{Path: lando.Path{"tooling", "node"}},
	{Path: lando.Path{"tooling", "wp-setup"}},
	{Path: lando.Path{"events", "post-start"}, Items: true},
}

// cache describes the object-cache selection.
type cache struct{ kind, version string }

func parseCache(s string) cache {
	k, ver, _ := strings.Cut(s, ":")
	return cache{k, ver}
}

// tplData feeds the support-file templates.
type tplData struct {
	r.Values
	Webserver, PHP, Webroot, URL, SiteURL string
	Multisite, Subdomains, Install        bool
	CacheKind, CacheHost, CachePort       string
	CachePlugin, Mail                     bool
	OpcacheOn, OpcacheValidate, JIT       bool
	OpcacheFreq                           int
	Hash                                  string
	Nested                                bool // WordPress lives in a subfolder (e.g. public_html)
	Marker                                string
}

func (rc *Recipe) Plan(v r.Values) (*r.Plan, error) {
	ws, name, url := v.Get(kWebserver), v.Get(kName), v.Get(kURL)
	oc := parseCache(v.Get(kObjectCache))
	defaultHost := name + ".lndo.site"
	proxyService := "appserver"
	if ws == nginx {
		proxyService = "appserver_nginx"
	}
	p := &r.Plan{Owned: owned}

	// ---- proxy ------------------------------------------------------------
	var hosts []any
	if ws == ols || url != defaultHost {
		hosts = append(hosts, lando.Tagged(hostPort(url, ws)))
	}
	if ws == ols && url != defaultHost {
		hosts = append(hosts, lando.Tagged(hostPort(defaultHost, ws)))
	}
	if v.Get(kMultisite) == "subdomain" {
		hosts = append(hosts, lando.Tagged(hostPort("*."+url, ws)))
	}
	proxy := lando.Map()
	if len(hosts) > 0 {
		proxy = lando.Map(proxyService, lando.Seq(hosts...))
	}

	// ---- services ---------------------------------------------------------
	services := lando.Map()
	tooling := lando.Map()
	add := func(m *yaml.Node, k string, val any) { m.Content = append(m.Content, lando.Str(k), lando.Node(val)) }

	appserver := lando.Map()
	if ws == ols {
		phpNoDot := strings.ReplaceAll(v.Get(kPHP), ".", "")
		add(appserver, "type", "lando")
		add(appserver, "ssl", true)
		add(appserver, "services", lando.Map(
			"image", fmt.Sprintf("litespeedtech/openlitespeed:%s-lsphp%s", olsImageVersion, phpNoDot),
			"command", "/entrypoint.sh true",
			"environment", map[string]string{"WP_CLI_CACHE_DIR": "/tmp/wp-cli-cache"},
		))
		iniDir := fmt.Sprintf("/usr/local/lsws/lsphp%s/etc/php/%s/mods-available", phpNoDot, v.Get(kPHP))
		docRoot := "/app/"
		if wr := v.Get(kWebroot); wr != "." {
			docRoot += wr + "/"
		}
		add(appserver, "build_as_root", lando.Seq(
			lando.Tagged(fmt.Sprintf("ln -sf /app/%s %s/99-stamper.ini", phpIni, iniDir)),
			lando.Tagged(fmt.Sprintf(`sed -i -e 's#^vhRoot.*#vhRoot /app/#' -e 's#docRoot .*#docRoot %s#' /usr/local/lsws/conf/templates/docker.conf /usr/local/lsws/.conf/templates/docker.conf`, docRoot)),
			lando.Tagged("rm -rf /usr/local/lsws/conf/vhosts/localhost && /usr/local/lsws/bin/lswsctrl restart"),
		))
		db := parseCache(v.Get(kDatabase)) // reuse "type:version" splitter
		services.Content = append(services.Content,
			lando.Str("database"), lando.Map(
				"type", v.Get(kDatabase),
				"portforward", true,
				"creds", lando.Map("user", "wordpress", "password", "wordpress", "database", "wordpress"),
			))
		for _, t := range olsTooling(db.kind) {
			tooling.Content = append(tooling.Content, t...)
		}
		if len(v.List(kExtensions)) > 0 {
			p.Warnings = append(p.Warnings, "php-extensions are not applied on OpenLiteSpeed (LSPHP packages differ); the image already has redis, memcached, imagick, intl and opcache")
		}
		if v.Get(kXdebug) != "off" {
			p.Warnings = append(p.Warnings, "xdebug is not available on the OpenLiteSpeed stack and was ignored")
		}
		p.Warnings = append(p.Warnings, "OpenLiteSpeed is experimental: it runs the official litespeedtech image as a custom Lando service. WebAdmin listens on port 7080 inside the container.")
	} else {
		if exts := v.List(kExtensions); len(exts) > 0 {
			// One non-fatal step per extension: Lando aborts the whole build on
			// the first failing command, and the recipe's wp-cli download runs
			// after build_as_root, so one broken extension used to leave the
			// container without `wp`.
			steps := make([]any, 0, len(exts))
			for _, e := range exts {
				steps = append(steps, lando.Tagged(fmt.Sprintf(
					"install-php-extensions %s || echo 'stamper: PHP extension %s failed to build, skipped'", e, e)))
			}
			add(appserver, "build_as_root", lando.Seq(steps...))
			if slices.Contains(exts, "memcache") && v.Get(kPHP) == "8.5" {
				p.Warnings = append(p.Warnings, "the memcache extension does not compile on PHP 8.5 yet and will be skipped; use PHP 8.4 or the built-in memcached extension")
			}
		}
	}
	if len(appserver.Content) > 0 {
		services.Content = append([]*yaml.Node{lando.Str("appserver"), appserver}, services.Content...)
	}

	switch oc.kind {
	case "redis":
		add(services, "redis", lando.Map("type", "redis:"+oc.version))
		add(tooling, "redis-cli", lando.Map("service", "redis"))
	case "valkey":
		add(services, "valkey", lando.Map(
			"type", "compose",
			"services", lando.Map("image", "valkey/valkey:"+oc.version, "command", "valkey-server --save '' --appendonly no"),
		))
		add(tooling, "valkey-cli", lando.Map("service", "valkey", "cmd", "valkey-cli"))
	case "memcached":
		add(services, "memcached", lando.Map("type", "memcached:"+oc.version))
		p.Notes = append(p.Notes, "Memcached runs at memcached:11211. Install the same drop-in production uses (W3 Total Cache, Object Cache Pro, or the `memcached` plugin + --php-extensions memcache).")
	case "apcu":
		p.Notes = append(p.Notes, "APCu is enabled for the web SAPI and CLI. Add an APCu object-cache drop-in matching production.")
	}

	if pc := parseCache(v.Get(kPageCache)); pc.kind == "varnish" {
		backend := "appserver"
		if ws == nginx {
			backend = "appserver_nginx"
		}
		add(services, "edge", lando.Map("type", "varnish:"+pc.version, "backends", []string{backend}, "ssl", true))
		add(proxy, "edge", []string{"edge." + url})
		add(tooling, "varnishadm", lando.Map("service", "edge", "user", "root"))
		p.Notes = append(p.Notes, "Varnish fronts the site at https://edge."+url+" (direct, uncached: https://"+url+").")
	}

	switch v.Get(kMail) {
	case "mailpit":
		add(services, "mailpit", lando.Map("type", "mailpit:1.29", "mailFrom", []string{"appserver"}))
		add(proxy, "mailpit", []string{"mail." + url})
		if !lando.PluginInstalled("@lando/mailpit") {
			p.Warnings = append(p.Warnings, "Mailpit needs the Lando plugin: run `lando plugin-add @lando/mailpit` (or choose --mail mailhog)")
		}
	case "mailhog":
		add(services, "mailhog", lando.Map("type", "mailhog", "hogfrom", []string{"appserver"}))
		add(proxy, "mailhog", []string{"mail." + url})
	}
	if v.Get(kMail) != "none" && ws == ols {
		p.Warnings = append(p.Warnings, "mail catchers wire sendmail for Lando PHP images; on OpenLiteSpeed point an SMTP plugin at the mail service on port 1025")
	}

	if v.Get(kDBAdmin) == "phpmyadmin" {
		add(services, "pma", lando.Map("type", "phpmyadmin:5.2", "hosts", []string{"database"}))
		add(proxy, "pma", []string{"pma." + url})
	}

	if n := v.Get(kNode); n != "none" && n != "" {
		add(services, "node", lando.Map("type", "node:"+n))
		for _, c := range []string{"npm", "npx", "node"} {
			add(tooling, c, lando.Map("service", "node"))
		}
	}

	// ---- setup script & tooling --------------------------------------------
	install := v.Bool(kInstall)
	add(tooling, "wp-setup", lando.Map(
		"service", "appserver",
		"description", "Re-run stamper's WordPress setup (install, constants, cache wiring)",
		"cmd", setupCmd,
	))

	// ---- top level ----------------------------------------------------------
	root := lando.Map("name", name)
	if ws != ols {
		via := "apache:2.4"
		if ws == nginx {
			via = "nginx:1.29"
		}
		cfg := lando.Map(
			"php", v.Get(kPHP),
			"via", via,
			"webroot", v.Get(kWebroot),
			"database", v.Get(kDatabase),
			"xdebug", xdebugValue(v.Get(kXdebug)),
		)
		files := lando.Map("php", phpIni)
		if ws == nginx {
			add(files, "vhosts", nginxConf)
		}
		add(cfg, "config", files)
		root = lando.Map("name", name, "recipe", "wordpress", "config", cfg)
	}
	if len(proxy.Content) > 0 {
		add(root, "proxy", proxy)
	}
	if len(services.Content) > 0 {
		add(root, "services", services)
	}
	add(root, "tooling", tooling)
	add(root, "events", lando.Map("post-start", lando.Seq(lando.Tagged(lando.Map("appserver", setupCmd)))))
	p.Landofile = root

	// ---- support files ------------------------------------------------------
	d := tplData{
		Values: v, Webserver: ws, PHP: v.Get(kPHP), Webroot: v.Get(kWebroot), URL: url,
		Nested: v.Get(kWebroot) != ".", Marker: r.ManagedMarker,
		SiteURL:   "https://" + url,
		Multisite: v.Get(kMultisite) != "no", Subdomains: v.Get(kMultisite) == "subdomain",
		Install:   install,
		CacheKind: oc.kind, CachePlugin: v.Bool(kCachePlugin) && (oc.kind == "redis" || oc.kind == "valkey"),
		Mail: v.Get(kMail) != "none",
	}
	switch oc.kind {
	case "redis", "valkey":
		d.CacheHost, d.CachePort = oc.kind, "6379"
	case "memcached":
		d.CacheHost, d.CachePort = "memcached", "11211"
	}
	switch v.Get(kOpcache) {
	case "dev":
		d.OpcacheOn, d.OpcacheValidate, d.OpcacheFreq = true, true, 0
	case "prod":
		d.OpcacheOn, d.OpcacheValidate, d.OpcacheFreq = true, true, 60
	case "prod-jit":
		d.OpcacheOn, d.OpcacheValidate, d.OpcacheFreq, d.JIT = true, true, 60, true
	}
	if v.Get(kOpcache) != "dev" {
		p.Notes = append(p.Notes, "OPcache revalidates every 60s: code edits can take up to a minute to show (or run `lando rebuild`).")
	}

	ini, err := render("php.ini.tmpl", d)
	if err != nil {
		return nil, err
	}
	p.Files = append(p.Files, r.File{Path: phpIni, Content: ini})
	if ws == nginx {
		conf, err := render("nginx.conf.tmpl", d)
		if err != nil {
			return nil, err
		}
		p.Files = append(p.Files, r.File{Path: nginxConf, Content: conf})
	} else {
		p.Remove = append(p.Remove, nginxConf)
	}
	// The script carries a hash of its inputs so repeated starts are no-ops.
	d.Hash = valuesHash(v)
	sh, err := render("wp-setup.sh.tmpl", d)
	if err != nil {
		return nil, err
	}
	p.Files = append(p.Files, r.File{Path: setupSh, Content: sh, Mode: 0o755})
	p.Files = append(p.Files, r.File{Path: supportDir + "/.gitignore", Content: []byte("# stamper runtime markers\n*.applied\n")})

	// Project-root files: wp-cli.yml lets `lando wp` work from the project
	// root; .gitignore is a one-time starting point the user then owns.
	if d.Nested {
		yml, err := render("wp-cli.yml.tmpl", d)
		if err != nil {
			return nil, err
		}
		p.Files = append(p.Files, r.File{Path: wpCLIYml, Content: yml, Policy: r.IfManaged})
	} else {
		p.Remove = append(p.Remove, wpCLIYml)
	}
	gi, err := render("gitignore.tmpl", d)
	if err != nil {
		return nil, err
	}
	p.Files = append(p.Files, r.File{Path: ".gitignore", Content: gi, Policy: r.Once})

	// ---- notes ----------------------------------------------------------------
	if !strings.HasSuffix(url, ".lndo.site") {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%s is not under lndo.site: add `127.0.0.1 %s` to /etc/hosts (and subdomains if used)", url, url))
	}
	if v.Get(kDatabase) == "mysql:5.7" {
		p.Warnings = append(p.Warnings, "MySQL 5.7 is end-of-life; only keep it if production still runs it")
	}
	if d.Multisite && ws == nginx {
		p.Notes = append(p.Notes, "Nginx vhost includes WordPress multisite rewrites.")
	}
	if install {
		p.Notes = append(p.Notes, fmt.Sprintf("After `lando start`: %s/wp-admin  (user %s / password %s)", d.SiteURL, v.Get(kAdminUser), v.Get(kAdminPass)))
	}
	return p, nil
}

func hostPort(host, ws string) string {
	if ws == ols {
		return host + ":80"
	}
	return host
}

func xdebugValue(mode string) any {
	if mode == "" || mode == "off" {
		return false
	}
	return mode
}

func render(name string, d tplData) ([]byte, error) {
	var b bytes.Buffer
	if err := tmpl.ExecuteTemplate(&b, name, d); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return b.Bytes(), nil
}

func valuesHash(v r.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, v[k])
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// olsTooling recreates the tooling Lando's wordpress recipe would provide.
func olsTooling(dbKind string) [][]*yaml.Node {
	hostOpt := lando.Map("host", lando.Map("description", "The database service to use", "default", "database", "alias", []string{"h"}))
	entries := [][]*yaml.Node{
		{lando.Str("wp"), lando.Map("service", "appserver", "cmd", "wp")},
		{lando.Str("php"), lando.Map("service", "appserver", "cmd", "php")},
		{lando.Str("db-import <file>"), lando.Map(
			"service", ":host", "description", "Imports a dump file into a database service",
			"cmd", "/helpers/sql-import.sh", "user", "root",
			"options", lando.Map(
				"host", lando.Map("description", "The database service to use", "default", "database", "alias", []string{"h"}),
				"no-wipe", lando.Map("description", "Do not destroy the existing database before an import", "boolean", true),
			))},
		{lando.Str("db-export [file]"), lando.Map(
			"service", ":host", "description", "Exports database from a database service to a file",
			"cmd", "/helpers/sql-export.sh", "user", "root",
			"options", lando.Map(
				"host", lando.Map("description", "The database service to use", "default", "database", "alias", []string{"h"}),
				"stdout", lando.Map("description", "Dump database to stdout"),
			))},
	}
	cli, cmd := "mysql", "mysql -uroot"
	if dbKind == "mariadb" {
		cli, cmd = "mariadb", "mariadb -uroot"
	}
	entries = append(entries, []*yaml.Node{lando.Str(cli), lando.Map(
		"service", ":host", "description", "Drops into a "+cli+" shell on a database service", "cmd", cmd, "options", hostOpt)})
	return entries
}
