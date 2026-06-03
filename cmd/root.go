// Package cmd holds pev's cobra subcommands. Conventions copied from
// sol-eng/wbi: persistent --loglevel flag bound via viper, all commands attach
// under newRootCmd, every subcommand returns a *cobra.Command.
package cmd

import (
	"embed"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// Importing primitives wires their init() registration.
	_ "github.com/posit-dev/pev/internal/primitives"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
	checksFS     embed.FS
	checksFSRoot string
)

// Execute is the entrypoint called from main.go. The embed.FS holds the
// built-in YAML catalog; root is the directory inside the FS to walk.
func Execute(version, commit, date string, fs embed.FS, root string) error {
	buildVersion = version
	buildCommit = commit
	buildDate = date
	checksFS = fs
	checksFSRoot = root
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pev",
		Short: "Posit Environment Validator — assess Linux readiness for Workbench, Connect, and Package Manager",
		Long: `pev assesses a Linux host's readiness to install and operate Posit
professional products. It is read-only by default and produces a Markdown
report (for humans) and a JSON sidecar (for diffing between runs).`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	root.PersistentFlags().String("loglevel", "info", "log level (trace|debug|info|warn|error)")
	root.PersistentFlags().String("out-dir", ".", "directory to write report artifacts into")
	_ = viper.BindPFlag("loglevel", root.PersistentFlags().Lookup("loglevel"))
	_ = viper.BindPFlag("out_dir", root.PersistentFlags().Lookup("out-dir"))

	root.AddCommand(newAssessCmd())
	root.AddCommand(newDiscoverCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newListChecksCmd())
	root.AddCommand(newLintChecksCmd())
	root.AddCommand(newFixCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newCompletionCmd(root))
	return root
}
