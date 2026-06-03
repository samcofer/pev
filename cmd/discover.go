package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/posit-dev/pev/internal/discover"
)

func newDiscoverCmd() *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "discover",
		Short: "Run discovery probes only and print what pev would assume",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			facts := discover.Gather(ctx)
			switch format {
			case "json":
				data, err := json.MarshalIndent(facts, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			default:
				printFactsHuman(facts)
			}
			return nil
		},
	}
	c.Flags().StringVar(&format, "format", "human", "human|json")
	return c
}

func printFactsHuman(f discover.HostFacts) {
	fmt.Println("Discovered host facts:")
	fmt.Printf("  hostname            : %s\n", f.Hostname)
	if f.FQDN != "" && f.FQDN != f.Hostname {
		fmt.Printf("  fqdn                : %s\n", f.FQDN)
	}
	fmt.Printf("  os                  : %s (%s)\n", f.OSPretty, f.OS)
	fmt.Printf("  os_family           : %s\n", f.OSFamily)
	fmt.Printf("  arch                : %s\n", f.Arch)
	fmt.Printf("  cpus                : %d\n", f.CPUs)
	fmt.Printf("  mem_mb              : %d\n", f.MemMB)
	fmt.Printf("  disk_gb_free[/]     : %d\n", f.DiskGB["/"])
	fmt.Printf("  running as root     : %v\n", f.Root)
	fmt.Printf("  workbench installed : %v\n", f.Products.Workbench)
	fmt.Printf("  connect installed   : %v\n", f.Products.Connect)
	fmt.Printf("  ppm installed       : %v\n", f.Products.PackageManager)
	for _, p := range f.R {
		fmt.Printf("  R                   : %s\n", p)
	}
	for _, p := range f.Python {
		fmt.Printf("  python              : %s\n", p)
	}
	for _, p := range f.Quarto {
		fmt.Printf("  quarto              : %s\n", p)
	}
}
