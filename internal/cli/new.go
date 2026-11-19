package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pipejesus/stamper/internal/lando"
	"github.com/pipejesus/stamper/internal/project"
	"github.com/pipejesus/stamper/internal/recipe"
	"github.com/pipejesus/stamper/internal/tui"
)

type newOpts struct {
	force, start, dryRun bool
}

func newNewCmd() *cobra.Command {
	var o newOpts
	cmd := &cobra.Command{
		Use:     "new [recipe] [url]",
		Aliases: []string{"stamp", "init"},
		Short:   "Stamp a new Lando environment from a recipe",
		Long:    "Stamp a new Lando environment. Pick a recipe subcommand for its flags; without one, an interactive picker runs.",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !interactive(true) {
				return fmt.Errorf("choose a recipe: %s", recipeNames())
			}
			return runNew(cmd, nil, &o)
		},
	}
	addNewFlags(cmd.Flags(), &o)
	for _, rc := range recipe.All() {
		cmd.AddCommand(newRecipeCmd(rc))
	}
	return cmd
}

func addNewFlags(fs *pflag.FlagSet, o *newOpts) {
	fs.BoolVarP(&o.force, "force", "f", false, "overwrite an existing .lando.yml (a .bak copy is kept)")
	fs.BoolVar(&o.start, "start", false, "run `lando start` afterwards")
	fs.BoolVar(&o.dryRun, "dry-run", false, "print the Landofile instead of writing anything")
}

// newRecipeCmd builds `stamper new <recipe>` with one flag per recipe option.
func newRecipeCmd(rc recipe.Recipe) *cobra.Command {
	var o newOpts
	cmd := &cobra.Command{
		Use:   rc.Name() + " [url]",
		Short: rc.Summary(),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v := flagValues(cmd, rc)
			if len(args) == 1 {
				v["url"] = args[0]
			}
			return runNewRecipe(cmd, rc, v, &o)
		},
	}
	addNewFlags(cmd.Flags(), &o)
	addOptionFlags(cmd.Flags(), rc)
	return cmd
}

// addOptionFlags registers --<key> for every option of rc.
func addOptionFlags(fs *pflag.FlagSet, rc recipe.Recipe) {
	for _, opt := range rc.Options() {
		usage := opt.Title
		if len(opt.Choices) > 0 {
			usage += " [" + recipe.ChoiceList(opt) + "]"
		}
		if opt.Kind == recipe.Multi {
			usage += " (comma-separated)"
		}
		if opt.Kind == recipe.Bool {
			fs.BoolP(opt.Key, opt.Short, recipe.Values{"d": opt.Default}.Bool("d"), usage)
			continue
		}
		fs.StringP(opt.Key, opt.Short, opt.Default, usage)
	}
}

// flagValues returns only the option flags the user explicitly set.
func flagValues(cmd *cobra.Command, rc recipe.Recipe) recipe.Values {
	v := recipe.Values{}
	for _, opt := range rc.Options() {
		f := cmd.Flags().Lookup(opt.Key)
		if f != nil && f.Changed {
			v[opt.Key] = f.Value.String()
		}
	}
	return v
}

// runNew is the interactive entry: pick a recipe, then continue.
func runNew(cmd *cobra.Command, _ []string, o *newOpts) error {
	if o == nil {
		o = &newOpts{}
	}
	all := recipe.All()
	rc := all[0]
	if len(all) > 1 {
		items := make([][2]string, len(all))
		for i, r := range all {
			items[i] = [2]string{r.Name() + tui.Muted.Render("  "+r.Summary()), r.Name()}
		}
		name, err := tui.Choose("Which recipe?", items)
		if err != nil {
			return quiet(err)
		}
		if rc, err = recipe.Get(name); err != nil {
			return err
		}
	}
	return runNewRecipe(cmd, rc, recipe.Values{}, o)
}

func runNewRecipe(cmd *cobra.Command, rc recipe.Recipe, given recipe.Values, o *newOpts) error {
	out := cmd.OutOrStdout()
	p, err := project.Open(gf.dir)
	if err != nil {
		return err
	}
	if p.Landofile != nil && !o.force && !o.dryRun {
		return fmt.Errorf("%s already exists — change it with `stamper config`, or pass --force to start over", p.Landofile.Path)
	}

	v := recipe.Defaults(rc)
	for k, x := range given {
		v[k] = x
	}
	isTUI := interactive(v["url"] == "")
	if isTUI {
		if v["url"] == "" {
			v["url"] = suggestURL(p.Dir)
		}
		// Derived values (app name, admin e-mail) stay empty so the form's
		// final URL decides them.
		tui.Print(out, tui.Banner(Version))
		if v, err = tui.RecipeForm(rc, v); err != nil {
			return quiet(err)
		}
	}
	v = rc.Normalize(v)
	if err := recipe.Validate(rc, v); err != nil {
		return err
	}
	plan, err := rc.Plan(v)
	if err != nil {
		return err
	}

	if o.dryRun {
		data, err := lando.New(p.Dir, plan.Landofile).Bytes()
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		printWarnings(cmd.ErrOrStderr(), plan.Warnings)
		return err
	}

	if isTUI {
		tui.Print(out, "\n"+tui.Title.Render("Summary")+"\n"+tui.Summary(rc, v, nil))
		printWarnings(out, plan.Warnings)
		ok, err := tui.Confirm("Write the environment to "+p.Dir+"?", "", true)
		if err != nil || !ok {
			return quiet(err)
		}
	}

	if p.Landofile != nil { // --force: keep a backup, start from scratch
		if err := backup(p.Landofile.Path); err != nil {
			return err
		}
		p.Landofile = nil
	}
	res, err := p.Apply(rc, v, plan, Version)
	if err != nil {
		return err
	}
	reportApply(out, p.Dir, res)
	if !isTUI {
		printWarnings(out, plan.Warnings)
	}
	for _, n := range plan.Notes {
		tui.Print(out, tui.Muted.Render("• ")+n)
	}

	start := o.start
	if isTUI && !start && lando.Installed() {
		if start, err = tui.Confirm("Run `lando start` now?", "First start pulls images and installs WordPress; it can take a few minutes.", true); err != nil {
			return quiet(err)
		}
	}
	if isTUI {
		if ok, _ := tui.Confirm("Install agent skills into this project too?", "Gutenberg blocks, modern CSS and modern PHP skills for Claude Code / Codex.", false); ok {
			if err := runSkillsInstall(cmd, nil, skillsOpts{}); err != nil {
				return err
			}
		}
	}
	if start {
		return lando.Run(p.Dir, "start")
	}
	next := "lando start"
	if d := relDir(p.Dir); d != "." {
		next = "cd " + d + " && " + next
	}
	tui.Print(out, "\nNext: "+tui.Command.Render(next))
	return nil
}

// suggestURL proposes <folder>.lndo.site.
func suggestURL(dir string) string {
	base := strings.ToLower(filepath.Base(dir))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "mysite"
	}
	return name + ".lndo.site"
}

func backup(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0o644)
}

func reportApply(w io.Writer, dir string, res *project.Result) {
	tui.Print(w, tui.OK.Render("✓ ")+"wrote "+relTo(dir, res.Landofile))
	for _, f := range res.Written {
		tui.Print(w, tui.OK.Render("✓ ")+"wrote "+f)
	}
	for _, f := range res.Removed {
		tui.Print(w, tui.Muted.Render("− removed "+f))
	}
	for _, f := range res.Kept {
		tui.Print(w, tui.Muted.Render("= kept your "+f))
	}
}

func printWarnings(w io.Writer, warnings []string) {
	for _, x := range warnings {
		tui.Print(w, tui.Warn.Render("! ")+x)
	}
}

func relTo(dir, p string) string {
	if r, err := filepath.Rel(dir, p); err == nil {
		return r
	}
	return p
}

func relDir(dir string) string {
	wd, _ := os.Getwd()
	if r, err := filepath.Rel(wd, dir); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return dir
}

func recipeNames() string {
	var n []string
	for _, r := range recipe.All() {
		n = append(n, r.Name())
	}
	return strings.Join(n, ", ")
}
