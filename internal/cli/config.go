package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pipejesus/stamper/internal/lando"
	"github.com/pipejesus/stamper/internal/project"
	"github.com/pipejesus/stamper/internal/recipe"
	"github.com/pipejesus/stamper/internal/tui"
)

type applyOpts struct {
	rebuild, dryRun bool
}

func newConfigCmd() *cobra.Command {
	var ao applyOpts
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or change the current project's Lando config",
		Long: `Show or change a stamped environment. stamper reconciles only the keys it owns in
.lando.yml (your own services, tooling, overrides and comments are kept) and rewrites its
support files under .lando/stamper/. Without a subcommand: interactive editor in a terminal,
otherwise 'show'.`,
		Example: `  stamper config                       # interactive editor
  stamper config show
  stamper config get php
  stamper config set php=8.3 webserver=nginx object-cache=redis:8 --rebuild
  stamper config set php-extensions=igbinary,pcov xdebug=debug
  stamper config upgrade               # re-render with the newest recipe`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tui.Interactive() && !gf.yes {
				return runConfigEdit(cmd)
			}
			return runConfigShow(cmd, false)
		},
	}

	var asJSON bool
	show := &cobra.Command{
		Use:   "show",
		Short: "Print the current configuration",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runConfigShow(cmd, asJSON) },
	}
	show.Flags().BoolVar(&asJSON, "json", false, "print as JSON")

	get := &cobra.Command{
		Use:   "get <key>",
		Short: "Print one configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, rc, v, err := loadConfig()
			if err != nil {
				return err
			}
			_ = p
			if _, ok := recipe.Find(rc, args[0]); !ok {
				return fmt.Errorf("unknown key %q; see `stamper config options`", args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), v.Get(args[0]))
			return nil
		},
	}

	set := &cobra.Command{
		Use:   "set <key=value>...",
		Short: "Change configuration values",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, rc, cur, err := loadConfig()
			if err != nil {
				return err
			}
			next := cur.Clone()
			for _, a := range args {
				k, val, ok := strings.Cut(a, "=")
				if !ok {
					return fmt.Errorf("%q: expected key=value", a)
				}
				k = strings.TrimPrefix(strings.TrimSpace(k), "--")
				if _, ok := recipe.Find(rc, k); !ok {
					return fmt.Errorf("unknown key %q; see `stamper config options`", k)
				}
				next[k] = strings.TrimSpace(val)
			}
			return applyConfig(cmd, p, rc, cur, next, ao, false)
		},
	}

	edit := &cobra.Command{
		Use:   "edit",
		Short: "Edit the configuration interactively",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runConfigEdit(cmd) },
	}

	upgrade := &cobra.Command{
		Use:   "upgrade",
		Short: "Re-render the Landofile and support files with this stamper version's recipe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, rc, cur, err := loadConfig()
			if err != nil {
				return err
			}
			return applyConfig(cmd, p, rc, cur, cur, ao, true)
		},
	}

	options := &cobra.Command{
		Use:   "options",
		Short: "List configurable keys and allowed values",
		RunE: func(cmd *cobra.Command, _ []string) error {
			rc, err := currentRecipe()
			if err != nil {
				return err
			}
			printOptions(cmd, rc)
			return nil
		},
	}

	for _, c := range []*cobra.Command{set, edit, upgrade} {
		c.Flags().BoolVar(&ao.rebuild, "rebuild", false, "run `lando rebuild -y` after applying")
		c.Flags().BoolVar(&ao.dryRun, "dry-run", false, "print the resulting .lando.yml without writing")
	}
	cmd.AddCommand(show, get, set, edit, upgrade, options)
	return cmd
}

func currentRecipe() (recipe.Recipe, error) {
	p, err := project.Open(gf.dir)
	if err != nil {
		return nil, err
	}
	return p.Recipe()
}

func loadConfig() (*project.Project, recipe.Recipe, recipe.Values, error) {
	p, err := project.Open(gf.dir)
	if err != nil {
		return nil, nil, nil, err
	}
	rc, err := p.Recipe()
	if err != nil {
		return nil, nil, nil, err
	}
	return p, rc, p.Values(rc), nil
}

func runConfigShow(cmd *cobra.Command, asJSON bool) error {
	p, rc, v, err := loadConfig()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"recipe": rc.Name(), "dir": p.Dir, "values": v})
	}
	tui.Print(out, tui.Title.Render(rc.Name())+tui.Muted.Render("  "+p.Dir))
	if p.State == nil {
		tui.Print(out, tui.Warn.Render("! ")+"no stamper state found; values were read from .lando.yml and defaults")
	}
	tui.Print(out, tui.Summary(rc, v, nil))
	return nil
}

func runConfigEdit(cmd *cobra.Command) error {
	p, rc, cur, err := loadConfig()
	if err != nil {
		return err
	}
	if !tui.Interactive() {
		return fmt.Errorf("`stamper config edit` needs a terminal; use `stamper config set key=value`")
	}
	tui.Print(cmd.OutOrStdout(), tui.Banner(Version))
	next, err := tui.RecipeForm(rc, cur)
	if err != nil {
		return quiet(err)
	}
	return applyConfig(cmd, p, rc, cur, next, applyOpts{}, false)
}

// applyConfig validates, previews, writes and optionally rebuilds.
func applyConfig(cmd *cobra.Command, p *project.Project, rc recipe.Recipe, cur, next recipe.Values, ao applyOpts, force bool) error {
	out := cmd.OutOrStdout()
	next = rc.Normalize(next)
	if err := recipe.Validate(rc, next); err != nil {
		return err
	}
	changed := project.Diff(cur, next)
	if len(changed) == 0 && !force {
		tui.Print(out, tui.Muted.Render("nothing changed"))
		return nil
	}
	plan, err := rc.Plan(next)
	if err != nil {
		return err
	}
	if ao.dryRun {
		if p.Landofile == nil {
			p.Landofile = lando.New(p.Dir, plan.Landofile)
		} else {
			lando.Apply(p.Landofile.Doc, plan.Landofile, plan.Owned)
		}
		data, err := p.Landofile.Bytes()
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	}

	tui.Print(out, tui.Summary(rc, next, changed))
	printWarnings(out, plan.Warnings)
	var cautions []string
	for _, k := range changed {
		if o, ok := recipe.Find(rc, k); ok && o.ChangeWarning != "" {
			cautions = append(cautions, o.ChangeWarning)
		}
	}
	printWarnings(out, cautions)

	isTUI := tui.Interactive() && !gf.yes
	if isTUI {
		ok, err := tui.Confirm("Apply these changes?", "", true)
		if err != nil || !ok {
			return quiet(err)
		}
	}
	res, err := p.Apply(rc, next, plan, Version)
	if err != nil {
		return err
	}
	reportApply(out, p.Dir, res)

	rebuild := ao.rebuild
	if isTUI && !rebuild && lando.Installed() {
		if rebuild, err = tui.Confirm("Run `lando rebuild -y` now?", "Lando applies service changes on rebuild.", true); err != nil {
			return quiet(err)
		}
	}
	if rebuild {
		return lando.Run(p.Dir, "rebuild", "-y")
	}
	tui.Print(out, "\nApply with: "+tui.Command.Render("lando rebuild -y"))
	return nil
}

func printOptions(cmd *cobra.Command, rc recipe.Recipe) {
	out := cmd.OutOrStdout()
	group := ""
	for _, o := range rc.Options() {
		if o.Group != group {
			group = o.Group
			tui.Print(out, "\n"+tui.Title.Render(group))
		}
		line := "  " + tui.KeyCol.Render(o.Key)
		if o.Default != "" {
			line += tui.Muted.Render(" (default " + o.Default + ")")
		}
		line += "  " + o.Title
		tui.Print(out, line)
		if len(o.Choices) > 0 {
			tui.Print(out, tui.Muted.Render("      "+recipe.ChoiceList(o)))
		}
	}
}
