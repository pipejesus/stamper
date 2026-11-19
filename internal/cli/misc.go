package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pipejesus/stamper/internal/freshness"
	"github.com/pipejesus/stamper/internal/lando"
	"github.com/pipejesus/stamper/internal/recipe"
	"github.com/pipejesus/stamper/internal/selfupdate"
	"github.com/pipejesus/stamper/internal/skills"
	"github.com/pipejesus/stamper/internal/tui"
)

func newRecipesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recipes [name]",
		Short: "List recipes, or show one recipe's options",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 1 {
				rc, err := recipe.Get(args[0])
				if err != nil {
					return err
				}
				tui.Print(out, tui.Title.Render(rc.Name())+"  "+rc.Summary())
				printOptions(cmd, rc)
				return nil
			}
			for _, rc := range recipe.All() {
				rev := ""
				if t := recipe.RevisionOf(rc); !t.IsZero() {
					rev = tui.Muted.Render("  (versions checked " + t.Format(time.DateOnly) + ")")
				}
				tui.Print(out, tui.KeyCol.Render(rc.Name())+"  "+rc.Summary()+rev)
			}
			return nil
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Lando, Docker, plugins and stamper's freshness",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runDoctor(cmd) },
	}
}

func runDoctor(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	check := func(ok bool, label, detail string) {
		mark := tui.OK.Render("✓")
		if !ok {
			mark = tui.Warn.Render("✗")
		}
		tui.Print(out, mark+" "+label+tui.Muted.Render("  "+detail))
	}
	ver, err := lando.Version()
	check(err == nil, "lando", strings.TrimSpace(ver+" "+errText(err, "install from https://docs.lando.dev")))
	dv, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	check(err == nil, "docker engine", strings.TrimSpace(string(dv)+" "+errText(err, "not reachable — is Docker running?")))
	check(lando.PluginInstalled("@lando/mailpit"), "@lando/mailpit plugin", "needed for --mail mailpit (lando plugin-add @lando/mailpit)")

	all, _ := skills.All()
	last := skills.LastUpdated()
	check(len(all) > 0, "bundled skills", fmt.Sprintf("%d skills, researched %s", len(all), last.Format(time.DateOnly)))
	for _, rc := range recipe.All() {
		if t := recipe.RevisionOf(rc); !t.IsZero() {
			check(time.Since(t) < freshness.RecipeMaxAge, "recipe "+rc.Name(), "version tables from "+t.Format(time.DateOnly))
		}
	}
	src := selfupdate.Repo
	if src == "" {
		src = "not configured (built from source)"
	}
	check(selfupdate.Repo != "", "self-update source", src)

	hints := freshness.Check(gf.dir, Version, time.Now())
	if len(hints) > 0 {
		tui.Print(out, "\n"+tui.Title.Render("Suggestions"))
		for _, h := range hints {
			tui.Print(out, "  "+h.Message+tui.Muted.Render(" → ")+tui.Command.Render(h.Command))
		}
	}
	return nil
}

func errText(err error, msg string) string {
	if err != nil {
		return msg
	}
	return ""
}

func newSelfUpdateCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update stamper (and with it the bundled skills and recipes) to the latest release",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			rel, err := selfupdate.Latest(ctx)
			if err != nil {
				return err
			}
			if !selfupdate.Newer(rel.Tag, Version) {
				tui.Print(out, tui.OK.Render("✓ ")+"stamper "+Version+" is the latest release")
				return nil
			}
			tui.Print(out, "stamper "+rel.Tag+" is available (current "+Version+")")
			if check {
				return nil
			}
			exe, err := selfupdate.Apply(ctx, rel)
			if err != nil {
				return err
			}
			tui.Print(out, tui.OK.Render("✓ ")+"updated "+exe+" to "+rel.Tag)
			tui.Print(out, "Refresh installed skills with "+tui.Command.Render("stamper skills update")+
				" and project configs with "+tui.Command.Render("stamper config upgrade"))
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "only report whether an update exists")
	return cmd
}
