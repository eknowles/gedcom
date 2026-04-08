package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
)

type cliCommand struct {
	name        string
	summary     string
	runLegacy   func()
	args        cobra.PositionalArgs
	example     string
	bashAliases []string
}

func newRootCmd(binaryName string) *cobra.Command {
	bin := filepath.Base(binaryName)

	cmd := &cobra.Command{
		Use:           "gedcom",
		Short:         "Tools for comparing, publishing, querying, and validating GEDCOM files",
		Long:          rootLongHelp(bin),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	for _, spec := range commandSpecs() {
		cmd.AddCommand(newLegacyCommand(spec))
	}

	carapace.Gen(cmd).Standalone()
	return cmd
}

func rootLongHelp(binaryName string) string {
	lines := []string{
		"GEDCOM command line tools:",
		"",
		"Commands:",
	}

	for _, spec := range commandSpecs() {
		lines = append(lines,
			fmt.Sprintf("  %s %-10s %s", binaryName, spec.name, spec.summary),
		)
	}

	lines = append(lines,
		"",
		fmt.Sprintf("Use '%s <command> -help' for command-specific options.", binaryName),
		fmt.Sprintf("Use '%s _carapace nushell' to generate Nushell completions.", binaryName),
	)

	return strings.Join(lines, "\n")
}

func commandSpecs() []cliCommand {
	specs := []cliCommand{
		{
			name:      "diff",
			summary:   "Compare gedcom files",
			runLegacy: runDiffCommand,
			example:   "gedcom diff -left-gedcom left.ged -right-gedcom right.ged -output out.html",
		},
		{
			name:      "publish",
			summary:   "Publish as HTML",
			runLegacy: runPublishCommand,
			example:   "gedcom publish -gedcom tree.ged -output-dir ./site",
		},
		{
			name:      "query",
			summary:   "Query with gedcomq",
			runLegacy: runQueryCommand,
			example:   "gedcom query -gedcom tree.ged '.Individuals | length'",
		},
		{
			name:      "tune",
			summary:   "Calculate ideal weights and similarities",
			runLegacy: runTuneCommand,
			example:   "gedcom tune -gedcom1 file1.ged -gedcom2 file2.ged",
		},
		{
			name:      "version",
			summary:   "Show version and exit",
			runLegacy: runVersionCommand,
			example:   "gedcom version",
		},
		{
			name:      "warnings",
			summary:   "Show warnings for a gedcom file",
			runLegacy: runWarningsCommand,
			example:   "gedcom warnings tree.ged",
			args:      cobra.MaximumNArgs(1),
		},
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].name < specs[j].name
	})

	return specs
}

func newLegacyCommand(spec cliCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:                spec.name,
		Short:              spec.summary,
		Example:            spec.example,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		SilenceErrors:      true,
		Run: func(_ *cobra.Command, args []string) {
			runLegacySubcommand(spec.name, args, spec.runLegacy)
		},
	}

	if spec.args != nil {
		cmd.Args = spec.args
	}

	if len(spec.bashAliases) > 0 {
		cmd.Aliases = spec.bashAliases
	}

	if spec.name == "warnings" {
		carapace.Gen(cmd).PositionalCompletion(carapace.ActionFiles())
	}

	return cmd
}

func runLegacySubcommand(name string, args []string, run func()) {
	flag.CommandLine = flag.NewFlagSet(name, flag.ContinueOnError)
	flag.CommandLine.SetOutput(os.Stderr)
	os.Args = append([]string{os.Args[0], name}, args...)
	run()
}
