package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/posit-dev/pev/internal/checks"
)

func newListChecksCmd() *cobra.Command {
	var (
		products []string
		tags     []string
		sev      string
	)
	c := &cobra.Command{
		Use:   "list-checks",
		Short: "List every check in the catalog with severity and tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			extraDirs := []string{}
			if home, err := os.UserHomeDir(); err == nil {
				extraDirs = append(extraDirs, filepath.Join(home, ".config", "pev", "checks.d"))
			}
			all, err := checks.Load(checksFS, checksFSRoot, nil, extraDirs)
			if err != nil {
				return err
			}
			f := checks.Filter{Products: products, Tags: tags, SeverityMin: checks.Severity(sev)}
			out := f.Apply(all)
			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tSEVERITY\tPRIMITIVE\tROOT\tTAGS\tTITLE")
			for _, c := range out {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%s\t%s\n",
					c.ID, c.Severity, c.Primitive, c.RequiresRoot, strings.Join(c.Tags, ","), c.Title)
			}
			return tw.Flush()
		},
	}
	c.Flags().StringSliceVar(&products, "products", nil, "filter by product")
	c.Flags().StringSliceVar(&tags, "tags", nil, "checks must have ALL of these tags")
	c.Flags().StringVar(&sev, "severity", "", "minimum severity (info|warning|blocking)")
	return c
}
