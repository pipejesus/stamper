package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pipejesus/stamper/internal/skills"
	"github.com/pipejesus/stamper/internal/tui"
)

type skillsOpts struct {
	agents []string
	global bool
	force  bool
	all    bool
}

func newSkillsCmd() *cobra.Command {
	var so skillsOpts
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Install curated agent skills for Claude Code and Codex",
		Long: `Bundled skills use the open SKILL.md format: Claude Code reads .claude/skills/,
Codex (and other agents) read .agents/skills/. Skills go into the project directory
(-C/--dir, default: the current folder, e.g. your theme) or, with --global, into $HOME.`,
	}
	flags := func(c *cobra.Command) {
		c.Flags().StringSliceVarP(&so.agents, "agent", "a", nil, "target agents: claude, codex (default: both)")
		c.Flags().BoolVarP(&so.global, "global", "g", false, "use ~/.claude/skills and ~/.agents/skills instead of the project")
		c.Flags().BoolVarP(&so.force, "force", "f", false, "overwrite skills edited locally or not installed by stamper")
	}

	list := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "status"},
		Short:   "List bundled skills and where they are installed",
		RunE:    func(cmd *cobra.Command, _ []string) error { return runSkillsList(cmd, so) },
	}
	install := &cobra.Command{
		Use:   "install [skill...]",
		Short: "Install skills (interactive picker when none are named)",
		Example: `  stamper skills install                          # pick interactively
  stamper skills install --all                    # everything, both agents, current folder
  stamper skills install php-modern css-modern -a claude -C wp-content/themes/mytheme`,
		RunE: func(cmd *cobra.Command, args []string) error { return runSkillsInstall(cmd, args, so) },
	}
	install.Flags().BoolVar(&so.all, "all", false, "install every bundled skill")
	update := &cobra.Command{
		Use:   "update",
		Short: "Update every stamper-installed skill that has a newer bundled version",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runSkillsUpdate(cmd, so) },
	}
	uninstall := &cobra.Command{
		Use:     "uninstall <skill...>",
		Aliases: []string{"remove", "rm"},
		Short:   "Remove stamper-installed skills",
		Args:    cobra.MinimumNArgs(1),
		RunE:    func(cmd *cobra.Command, args []string) error { return runSkillsUninstall(cmd, args, so) },
	}
	for _, c := range []*cobra.Command{list, install, update, uninstall} {
		flags(c)
	}
	cmd.AddCommand(list, install, update, uninstall)
	cmd.RunE = list.RunE
	return cmd
}

func skillRoot(global bool) (string, error) {
	if global {
		return os.UserHomeDir()
	}
	return filepath.Abs(gf.dir)
}

func resolveAgents(keys []string) ([]skills.Agent, error) {
	if len(keys) == 0 {
		return skills.Agents, nil
	}
	var out []skills.Agent
	for _, k := range keys {
		a, err := skills.AgentByKey(strings.TrimSpace(k))
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func runSkillsList(cmd *cobra.Command, so skillsOpts) error {
	all, err := skills.All()
	if err != nil {
		return err
	}
	root, err := skillRoot(so.global)
	if err != nil {
		return err
	}
	agents, err := resolveAgents(so.agents)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if len(all) == 0 {
		tui.Print(out, tui.Warn.Render("no skills are bundled in this build"))
		return nil
	}
	tui.Print(out, tui.Muted.Render("target: "+root))
	for _, s := range all {
		updated := "unknown"
		if !s.Updated.IsZero() {
			updated = s.Updated.Format(time.DateOnly)
		}
		tui.Print(out, "\n"+tui.Title.Render(s.Name)+tui.Muted.Render("  researched "+updated+" · "+fmt.Sprint(len(s.Files))+" files"))
		tui.Print(out, "  "+wrap(s.Description, 92, "  "))
		for _, a := range agents {
			st := skills.Check(filepath.Join(root, a.Dir), s)
			tui.Print(out, "  "+tui.KeyCol.Render(fmt.Sprintf("%-7s", a.Key))+statusStyle(st))
		}
	}
	return nil
}

func statusStyle(st skills.Status) string {
	switch st {
	case skills.Current:
		return tui.OK.Render(st.String())
	case skills.Missing:
		return tui.Muted.Render(st.String())
	default:
		return tui.Warn.Render(st.String())
	}
}

func runSkillsInstall(cmd *cobra.Command, names []string, so skillsOpts) error {
	all, err := skills.All()
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return errors.New("no skills are bundled in this build")
	}
	out := cmd.OutOrStdout()
	agentKeys := so.agents
	pick := len(names) == 0 && !so.all
	if pick && !interactive(true) {
		return errors.New("name the skills to install or pass --all (see `stamper skills list`)")
	}
	if so.all {
		for _, s := range all {
			names = append(names, s.Name)
		}
	}
	if pick {
		items := make([][2]string, len(all))
		var pre []string
		for i, s := range all {
			items[i] = [2]string{s.Name + tui.Muted.Render("  "+short(s.Description, 70)), s.Name}
			pre = append(pre, s.Name)
		}
		if names, err = tui.ChooseMany("Which skills?", "space toggles, enter confirms", items, pre); err != nil {
			return quiet(err)
		}
		if len(agentKeys) == 0 {
			ag := make([][2]string, len(skills.Agents))
			var preA []string
			for i, a := range skills.Agents {
				ag[i] = [2]string{a.Label + tui.Muted.Render("  → "+a.Dir), a.Key}
				preA = append(preA, a.Key)
			}
			if agentKeys, err = tui.ChooseMany("For which agents?", "", ag, preA); err != nil {
				return quiet(err)
			}
		}
		if !so.global {
			abs, _ := filepath.Abs(gf.dir)
			dir, err := tui.Input("Install into which folder?", "Usually your theme or plugin root. Use ~ for global (all projects).", abs)
			if err != nil {
				return quiet(err)
			}
			if dir == "~" {
				so.global = true
			} else {
				gf.dir = dir
			}
		}
	}
	if len(names) == 0 {
		tui.Print(out, tui.Muted.Render("nothing selected"))
		return nil
	}
	agents, err := resolveAgents(agentKeys)
	if err != nil {
		return err
	}
	root, err := skillRoot(so.global)
	if err != nil {
		return err
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}

	var failed []string
	for _, name := range names {
		s, err := skills.Get(name)
		if err != nil {
			return err
		}
		for _, a := range agents {
			dir := filepath.Join(root, a.Dir)
			before, err := skills.Install(dir, s, so.force)
			rel := relTo(root, filepath.Join(dir, s.Name))
			switch {
			case err != nil:
				failed = append(failed, err.Error())
			case before == skills.Current:
				tui.Print(out, tui.Muted.Render("= "+rel+" already up to date"))
			case before == skills.Missing:
				tui.Print(out, tui.OK.Render("✓ ")+"installed "+rel)
			default:
				tui.Print(out, tui.OK.Render("↻ ")+"updated "+rel+tui.Muted.Render(" (was "+before.String()+")"))
			}
		}
	}
	if len(failed) > 0 {
		printWarnings(out, failed)
		return errors.New("some skills were not installed")
	}
	tui.Print(out, tui.Muted.Render("Agents pick up new skills in their next session."))
	return nil
}

func runSkillsUpdate(cmd *cobra.Command, so skillsOpts) error {
	all, err := skills.All()
	if err != nil {
		return err
	}
	root, err := skillRoot(so.global)
	if err != nil {
		return err
	}
	agents, err := resolveAgents(so.agents)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	n := 0
	for _, s := range all {
		for _, a := range agents {
			dir := filepath.Join(root, a.Dir)
			st := skills.Check(dir, s)
			if st != skills.Outdated && !(so.force && st == skills.Modified) {
				continue
			}
			if _, err := skills.Install(dir, s, so.force); err != nil {
				return err
			}
			tui.Print(out, tui.OK.Render("↻ ")+"updated "+relTo(root, filepath.Join(dir, s.Name)))
			n++
		}
	}
	if n == 0 {
		tui.Print(out, tui.Muted.Render("all installed skills are up to date in "+root))
	}
	return nil
}

func runSkillsUninstall(cmd *cobra.Command, names []string, so skillsOpts) error {
	root, err := skillRoot(so.global)
	if err != nil {
		return err
	}
	agents, err := resolveAgents(so.agents)
	if err != nil {
		return err
	}
	for _, name := range names {
		s, err := skills.Get(name)
		if err != nil {
			return err
		}
		for _, a := range agents {
			dir := filepath.Join(root, a.Dir)
			if skills.Check(dir, s) == skills.Missing {
				continue
			}
			if err := skills.Uninstall(dir, s, so.force); err != nil {
				return err
			}
			tui.Print(cmd.OutOrStdout(), tui.OK.Render("− ")+"removed "+relTo(root, filepath.Join(dir, s.Name)))
		}
	}
	return nil
}

func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := strings.LastIndex(s[:n], " ")
	if cut < n/2 {
		cut = n
	}
	return s[:cut] + "…"
}

func wrap(s string, width int, indent string) string {
	var lines []string
	var line string
	for _, w := range strings.Fields(s) {
		if len(line)+len(w)+1 > width && line != "" {
			lines = append(lines, line)
			line = w
			continue
		}
		line = strings.TrimSpace(line + " " + w)
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"+indent)
}
