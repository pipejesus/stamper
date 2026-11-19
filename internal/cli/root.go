// Package cli wires stamper's commands. Every command works from flags alone
// (scriptable, CI-friendly) and falls back to charmbracelet forms when run
// in a terminal without enough information.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"

	"github.com/pipejesus/stamper/internal/freshness"
	"github.com/pipejesus/stamper/internal/lando"
	"github.com/pipejesus/stamper/internal/recipe"
	"github.com/pipejesus/stamper/internal/recipe/wordpress"
	"github.com/pipejesus/stamper/internal/tui"
)

// Build information, injected by main.
var (
	Version = "0.1.0-dev"
	Commit  = ""
)

func init() {
	// The recipe registry: add new recipes here.
	recipe.Register(wordpress.New())
}

type globalFlags struct {
	dir         string
	yes         bool
	interactive bool
	noCheck     bool
}

var gf globalFlags

// Execute runs the CLI.
func Execute() int {
	root := newRoot()
	opts := []fang.Option{fang.WithVersion(Version)}
	if Commit != "" {
		opts = append(opts, fang.WithCommit(Commit))
	}
	if err := fang.Execute(context.Background(), root, opts...); err != nil {
		return 1
	}
	return 0
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "stamper",
		Short: "Stamp out and tune Lando environments, and install agent skills",
		Long: `stamper creates production-matching Lando environments from modular recipes
(WordPress today: Apache, Nginx or OpenLiteSpeed, MariaDB/MySQL, Redis/Valkey/Memcached/APCu,
Varnish, Mailpit, phpMyAdmin, Node) and keeps them easy to change. It also installs curated,
token-efficient agent skills (Gutenberg blocks, modern CSS, modern PHP) for Claude Code and Codex.

Run without arguments in a terminal for the interactive menu.`,
		Example: `  stamper                                   # interactive menu
  stamper new wordpress myshop.lndo.site    # stamp with defaults
  stamper new wordpress shop.lndo.site -w nginx -p 8.3 -d mysql:8.0 -c redis:8 --yes --start
  stamper config set php=8.4 object-cache=valkey:8 --rebuild
  stamper skills install --all --agent claude,codex`,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			bootChecks(cmd)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !tui.Interactive() {
				return cmd.Help()
			}
			return menu(cmd)
		},
	}
	pf := root.PersistentFlags()
	pf.StringVarP(&gf.dir, "dir", "C", ".", "project directory to operate on")
	pf.BoolVarP(&gf.yes, "yes", "y", false, "never prompt; accept defaults and confirmations")
	pf.BoolVarP(&gf.interactive, "interactive", "i", false, "force interactive forms even when flags are complete")
	pf.BoolVar(&gf.noCheck, "no-update-check", false, "skip boot-time freshness checks (or set STAMPER_NO_UPDATE_CHECK=1)")

	root.AddCommand(newNewCmd(), newConfigCmd(), newSkillsCmd(), newRecipesCmd(), newDoctorCmd(), newSelfUpdateCmd())
	return root
}

// interactive decides whether to show forms: forced with -i, suppressed with
// --yes or without a terminal, otherwise only when information is missing.
func interactive(missing bool) bool {
	if gf.yes || !tui.Interactive() {
		return false
	}
	return gf.interactive || missing
}

// bootChecks prints freshness hints: stale skills/recipes, outdated installs
// in this project, Landofiles rendered by older recipes, newer releases.
func bootChecks(cmd *cobra.Command) {
	switch cmd.Name() {
	case "completion", "help", "__complete", "__completeNoDesc", "self-update":
		return
	}
	if gf.noCheck || os.Getenv("STAMPER_NO_UPDATE_CHECK") != "" {
		return
	}
	hints := freshness.Check(gf.dir, Version, time.Now())
	if len(hints) == 0 {
		return
	}
	w := cmd.ErrOrStderr()
	for _, h := range hints {
		tui.Print(w, tui.Warn.Render("↻ ")+h.Message+tui.Muted.Render(" → ")+tui.Command.Render(h.Command))
	}
	tui.Print(w, "")
}

func menu(cmd *cobra.Command) error {
	tui.Print(cmd.OutOrStdout(), tui.Banner(Version))
	items := [][2]string{{"Stamp a new Lando environment", "new"}}
	if _, err := lando.Load(gf.dir); err == nil {
		items = append(items, [2]string{"Change this project's Lando config", "config"})
	}
	items = append(items,
		[2]string{"Install / update agent skills", "skills"},
		[2]string{"Check my setup (doctor)", "doctor"},
		[2]string{"Quit", "quit"},
	)
	choice, err := tui.Choose("What do you want to do?", items)
	if err != nil {
		return quiet(err)
	}
	switch choice {
	case "new":
		return runNew(cmd, nil, nil)
	case "config":
		return runConfigEdit(cmd)
	case "skills":
		return runSkillsInstall(cmd, nil, skillsOpts{})
	case "doctor":
		return runDoctor(cmd)
	}
	return nil
}

// quiet turns a user abort into a clean exit.
func quiet(err error) error {
	if errors.Is(err, tui.ErrAborted) {
		fmt.Fprintln(os.Stderr, tui.Muted.Render("aborted"))
		return nil
	}
	return err
}
