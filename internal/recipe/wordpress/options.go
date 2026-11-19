package wordpress

import (
	"fmt"
	"regexp"
	"strings"

	r "github.com/pipejesus/stamper/internal/recipe"
)

// Option keys.
const (
	kURL         = "url"
	kName        = "name"
	kWebroot     = "webroot"
	kMultisite   = "multisite"
	kWebserver   = "webserver"
	kPHP         = "php"
	kDatabase    = "database"
	kObjectCache = "object-cache"
	kPageCache   = "page-cache"
	kOpcache     = "opcache"
	kExtensions  = "php-extensions"
	kXdebug      = "xdebug"
	kMemory      = "memory-limit"
	kUpload      = "upload-max"
	kMail        = "mail"
	kDBAdmin     = "db-admin"
	kNode        = "node"
	kInstall     = "wp-install"
	kWPVersion   = "wp-version"
	kLocale      = "wp-locale"
	kAdminUser   = "admin-user"
	kAdminPass   = "admin-password"
	kAdminEmail  = "admin-email"
	kCachePlugin = "cache-plugin"
)

// Webservers.
const (
	apache = "apache"
	nginx  = "nginx"
	ols    = "openlitespeed"
)

var (
	hostRe = regexp.MustCompile(`^(\*\.)?([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z][a-z0-9-]*[a-z0-9]$`)
	nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	sizeRe = regexp.MustCompile(`^(-1|[0-9]+[KMG]?)$`)
)

func validHost(s string) error {
	if !hostRe.MatchString(strings.ToLower(s)) {
		return fmt.Errorf("%q is not a valid hostname (e.g. myshop.lndo.site)", s)
	}
	return nil
}

func validSize(s string) error {
	if !sizeRe.MatchString(strings.ToUpper(s)) {
		return fmt.Errorf("%q must look like 512M, 2G or -1", s)
	}
	return nil
}

// olsPHP lists PHP versions published as litespeedtech/openlitespeed images.
var olsPHP = map[string]bool{"8.2": true, "8.3": true, "8.4": true, "8.5": true}

const olsImageVersion = "1.9.4"

func notOLS(v r.Values) bool     { return v.Get(kWebserver) != ols }
func installing(v r.Values) bool { return v.Bool(kInstall) }
func redisLike(v r.Values) bool {
	c := v.Get(kObjectCache)
	return strings.HasPrefix(c, "redis") || strings.HasPrefix(c, "valkey")
}
func cachePluginVis(v r.Values) bool { return installing(v) && redisLike(v) }

func options() []r.Option {
	return []r.Option{
		// --- Site ---
		{Key: kURL, Short: "u", Group: "Site", Title: "Local URL", Kind: r.String, Required: true,
			Help:     "Hostname of the local site. *.lndo.site resolves automatically; anything else needs a hosts entry.",
			Validate: validHost},
		{Key: kName, Short: "n", Group: "Site", Title: "Lando app name", Kind: r.String,
			Help: "Defaults to the URL without its domain suffix (myshop.lndo.site → myshop).",
			Validate: func(s string) error {
				if !nameRe.MatchString(s) {
					return fmt.Errorf("%q: use lowercase letters, digits and dashes", s)
				}
				return nil
			}},
		{Key: kWebroot, Group: "Site", Title: "Web root", Kind: r.String, Default: ".",
			Help: "Directory (relative to the project) that holds WordPress, e.g. '.', 'wordpress', 'web'.",
			Validate: func(s string) error {
				if strings.HasPrefix(s, "/") || strings.Contains(s, "..") {
					return fmt.Errorf("%q must be a relative path inside the project", s)
				}
				return nil
			}},
		{Key: kMultisite, Group: "Site", Title: "Multisite", Kind: r.Select, Default: "no",
			Choices: []r.Choice{{Value: "no", Label: "Single site"}, {Value: "subdirectory", Label: "Multisite: subdirectories"}, {Value: "subdomain", Label: "Multisite: subdomains", Note: "adds a wildcard proxy"}}},

		// --- Stack ---
		{Key: kWebserver, Short: "w", Group: "Stack", Title: "Web server", Kind: r.Select, Default: apache,
			ChangeWarning: "Switching web server replaces the appserver container: run `lando rebuild -y`.",
			Choices: []r.Choice{
				{Value: apache, Label: "Apache 2.4 + mod_php"},
				{Value: nginx, Label: "Nginx 1.29 + PHP-FPM"},
				{Value: ols, Label: "OpenLiteSpeed " + olsImageVersion + " + LSPHP", Note: "experimental, PHP 8.2–8.5"},
			}},
		{Key: kPHP, Short: "p", Group: "Stack", Title: "PHP version", Kind: r.Select, Default: "8.4",
			Choices: []r.Choice{{Value: "8.5"}, {Value: "8.4", Note: "default"}, {Value: "8.3"}, {Value: "8.2"}, {Value: "8.1", Note: "EOL"}, {Value: "8.0", Note: "EOL"}, {Value: "7.4", Note: "EOL"}},
			Help:    "Match production. Lando 3 ships images up to PHP 8.5."},
		{Key: kDatabase, Short: "d", Group: "Stack", Title: "Database", Kind: r.Select, Default: "mariadb:11.4",
			Choices: []r.Choice{
				{Value: "mariadb:11.8", Label: "MariaDB 11.8", Note: "LTS"},
				{Value: "mariadb:11.4", Label: "MariaDB 11.4", Note: "LTS, default"},
				{Value: "mariadb:10.11", Label: "MariaDB 10.11", Note: "LTS"},
				{Value: "mariadb:10.6", Label: "MariaDB 10.6", Note: "LTS"},
				{Value: "mysql:8.4", Label: "MySQL 8.4", Note: "LTS"},
				{Value: "mysql:8.0", Label: "MySQL 8.0"},
				{Value: "mysql:5.7", Label: "MySQL 5.7", Note: "EOL"},
			},
			Help:          "Match production's engine and major version.",
			ChangeWarning: "Database engine/version changes need a fresh data volume: run `lando db-export` first, then `lando destroy -y && lando start` and `lando db-import <dump>`."},

		// --- Caching ---
		{Key: kObjectCache, Short: "c", Group: "Caching", Title: "Object cache backend", Kind: r.Select, Default: "none",
			Choices: []r.Choice{
				{Value: "none", Label: "None"},
				{Value: "redis:8", Label: "Redis 8"},
				{Value: "redis:7.4", Label: "Redis 7.4"},
				{Value: "valkey:8", Label: "Valkey 8", Note: "Redis-compatible"},
				{Value: "memcached:1.6", Label: "Memcached 1.6"},
				{Value: "apcu", Label: "APCu", Note: "in-process, no service"},
			},
			Help: "The PHP redis, memcached and apcu extensions are always present; this adds the backing service."},
		{Key: kCachePlugin, Group: "Caching", Title: "Install the Redis Object Cache drop-in?", Kind: r.Bool, Default: "true",
			Help:    "Installs and enables the redis-cache plugin and sets WP_REDIS_HOST. Turn off if production uses Object Cache Pro etc.",
			Visible: cachePluginVis},
		{Key: kPageCache, Group: "Caching", Title: "Full-page cache", Kind: r.Select, Default: "none",
			Choices: []r.Choice{{Value: "none", Label: "None"}, {Value: "varnish:6", Label: "Varnish 6", Note: "served on edge.<url>"}}},
		{Key: kOpcache, Group: "Caching", Title: "OPcache profile", Kind: r.Select, Default: "dev",
			Choices: []r.Choice{
				{Value: "dev", Label: "Development", Note: "revalidate on every request"},
				{Value: "prod", Label: "Production-like", Note: "revalidate every 60s"},
				{Value: "prod-jit", Label: "Production-like + JIT", Note: "tracing JIT"},
				{Value: "off", Label: "Disabled"},
			}},

		// --- PHP ---
		{Key: kExtensions, Short: "e", Group: "PHP", Title: "Extra PHP extensions", Kind: r.Multi, Visible: notOLS,
			Help: "Built-ins already include apcu, bcmath, exif, gd, imagick, intl, memcached, opcache, redis, soap, xhprof, zip.",
			Choices: []r.Choice{
				{Value: "igbinary", Note: "compact serializer for object caches"},
				{Value: "msgpack"}, {Value: "zstd"}, {Value: "lz4"},
				{Value: "memcache", Note: "for the Memcache drop-in"},
				{Value: "pcov", Note: "fast coverage"}, {Value: "excimer", Note: "sampling profiler"},
				{Value: "xsl"}, {Value: "tidy"}, {Value: "gmp"}, {Value: "sockets"},
			}},
		{Key: kXdebug, Short: "x", Group: "PHP", Title: "Xdebug mode", Kind: r.Select, Default: "off", Visible: notOLS,
			Choices: []r.Choice{
				{Value: "off", Label: "Off", Note: "fastest"},
				{Value: "debug", Label: "Step debugging"},
				{Value: "develop,debug", Label: "Step debugging + dev helpers"},
				{Value: "profile", Label: "Profiler"},
				{Value: "coverage", Label: "Code coverage"},
				{Value: "trace", Label: "Function trace"},
			},
			Help: "Requests trigger Xdebug via XDEBUG_TRIGGER / browser extension (start_with_request=trigger)."},
		{Key: kMemory, Group: "PHP", Title: "memory_limit", Kind: r.String, Default: "512M", Validate: validSize},
		{Key: kUpload, Group: "PHP", Title: "upload_max_filesize / post_max_size", Kind: r.String, Default: "128M", Validate: validSize},

		// --- Tools ---
		{Key: kMail, Group: "Tools", Title: "Mail catcher", Kind: r.Select, Default: "mailpit",
			Choices: []r.Choice{{Value: "mailpit", Label: "Mailpit", Note: "needs the @lando/mailpit plugin"}, {Value: "mailhog", Label: "MailHog", Note: "legacy, bundled"}, {Value: "none", Label: "None"}}},
		{Key: kDBAdmin, Group: "Tools", Title: "Database UI", Kind: r.Select, Default: "none",
			Choices: []r.Choice{{Value: "none", Label: "None"}, {Value: "phpmyadmin", Label: "phpMyAdmin 5.2", Note: "on pma.<url>"}}},
		{Key: kNode, Group: "Tools", Title: "Node.js service (theme/block builds)", Kind: r.Select, Default: "none",
			Choices: []r.Choice{{Value: "none", Label: "None"}, {Value: "24", Label: "Node 24", Note: "LTS"}, {Value: "22", Label: "Node 22", Note: "LTS"}, {Value: "26", Label: "Node 26"}}},

		// --- WordPress ---
		{Key: kInstall, Group: "WordPress", Title: "Download & install WordPress on first start?", Kind: r.Bool, Default: "true",
			Help: "Runs an idempotent setup script after `lando start` (core download, wp-config, install, cache wiring)."},
		{Key: kWPVersion, Group: "WordPress install", Title: "WordPress version", Kind: r.String, Default: "latest", Visible: installing},
		{Key: kLocale, Group: "WordPress install", Title: "Locale", Kind: r.String, Default: "en_US", Visible: installing},
		{Key: kAdminUser, Group: "WordPress install", Title: "Admin user", Kind: r.String, Default: "admin", Visible: installing},
		{Key: kAdminPass, Group: "WordPress install", Title: "Admin password", Kind: r.String, Default: "admin", Visible: installing},
		{Key: kAdminEmail, Group: "WordPress install", Title: "Admin e-mail", Kind: r.String, Visible: installing,
			Help: "Defaults to admin@<url>."},
	}
}
