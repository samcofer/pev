package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/discover"
	"github.com/posit-dev/pev/internal/logging"
	"github.com/posit-dev/pev/internal/prompt"
	"github.com/posit-dev/pev/internal/report"
)

func newAssessCmd() *cobra.Command {
	var (
		products       []string
		profile        string
		extraFiles     []string
		includeUser    bool
		nonInteractive bool
		yes            bool
		licenseFile    string
		hostnames      []string
		idp            string
		outputs        []string
		severityMin    string
		tagsAny        []string
		skipTags       []string
		skipChecks     []string
		noCmdLog       bool
	)
	c := &cobra.Command{
		Use:   "assess",
		Short: "Run the readiness checks and write Markdown + JSON reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			outDir, _ := cmd.Flags().GetString("out-dir")
			if outDir == "" {
				outDir = "."
			}
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("create out-dir: %w", err)
			}

			// Logger
			logFile, err := logging.Init(outDir, mustGetString(cmd, "loglevel"))
			if err != nil {
				return err
			}
			defer logFile.Close()

			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			ctx, cancel := context.WithCancel(parent)
			defer cancel()

			started := time.Now()

			// Discovery first — every prompt is seeded from this.
			facts := discover.Gather(ctx)
			log.WithField("facts", facts).Debug("discovery complete")

			// Resolve selected products.
			selected := discover.SelectedFromFlag(products, facts.Products)
			if profile != "" && len(selected) == 0 {
				selected = profileToProducts(profile)
			}
			if len(selected) == 0 {
				selected = []string{"workbench", "connect", "packagemanager"}
			}

			// Pick the prompt mode from CLI flags. --non-interactive wins
			// over --yes; both fall through to interactive when neither
			// is set. The interactive driver auto-downgrades to yes when
			// stdin/stdout are not a TTY (CI, piped runs).
			mode := prompt.ModeInteractive
			switch {
			case nonInteractive:
				mode = prompt.ModeNonInteractive
			case yes:
				mode = prompt.ModeYes
			}
			driver := prompt.New(mode)
			inputs := buildInputs(licenseFile, hostnames, idp, facts, selected, driver)

			// Load catalog.
			extraDirs := []string{}
			if includeUser {
				if home, err := os.UserHomeDir(); err == nil {
					extraDirs = append(extraDirs, filepath.Join(home, ".config", "pev", "checks.d"))
				}
			}
			all, err := checks.Load(checksFS, checksFSRoot, extraFiles, extraDirs)
			if err != nil {
				return fmt.Errorf("load catalog: %w", err)
			}
			if errs := checks.Lint(all); len(errs) > 0 {
				for _, e := range errs {
					log.WithError(e).Error("catalog lint")
				}
				return fmt.Errorf("catalog has %d lint error(s); run `pev lint-checks` for details", len(errs))
			}

			// Filter for selected products + tags + severity.
			f := checks.Filter{
				Products:    selected,
				Tags:        tagsAny,
				SkipTags:    skipTags,
				SkipIDs:     skipChecks,
				SeverityMin: checks.Severity(severityMin),
			}
			filtered := f.Apply(all)

			// cmdlog
			cl, err := logging.NewCmdLog(outDir, facts.Hostname, !noCmdLog)
			if err != nil {
				return err
			}
			defer cl.Close()

			// Engine.
			eng := checks.Engine{Facts: facts, Inputs: inputs, CmdLog: cl}
			results := eng.Run(ctx, filtered)

			finished := time.Now()
			rep := checks.Report{
				PevVersion:    buildVersion,
				SchemaVersion: checks.SchemaVersion,
				Host:          facts,
				Run: checks.Run{
					Products: selected,
					Profile:  profile,
					Inputs:   inputs,
				},
				StartedAt:  started,
				FinishedAt: finished,
				Results:    results,
				Summary:    report.Summarize(results),
			}

			ts := started.UTC().Format("20060102T150405")
			base := fmt.Sprintf("pev-report-%s-%s", facts.Hostname, ts)

			wantMd, wantJSON := wantOutputs(outputs)
			if wantMd {
				p, err := report.WriteMarkdown(outDir, base, rep)
				if err != nil {
					return err
				}
				fmt.Println(p)
			}
			if wantJSON {
				p, err := report.WriteJSON(outDir, base, rep)
				if err != nil {
					return err
				}
				fmt.Println(p)
			}

			// Always print the Markdown to screen so an SE running on a customer box
			// gets immediate feedback even if they forgot to look in --out-dir.
			fmt.Println()
			fmt.Println(report.RenderMarkdown(rep))

			if rep.Summary.Blocking > 0 {
				return fmt.Errorf("%d blocking failure(s) — see report", rep.Summary.Blocking)
			}
			return nil
		},
	}
	c.Flags().StringSliceVar(&products, "products", nil, "products to assess (workbench,connect,packagemanager); auto-detected if empty")
	c.Flags().StringVar(&profile, "profile", "", "convenience preset: single-server | workbench | connect | ppm")
	c.Flags().StringArrayVar(&extraFiles, "checks-file", nil, "extra YAML check pack (repeatable)")
	c.Flags().BoolVar(&includeUser, "include-user-checks", true, "load packs from ~/.config/pev/checks.d/*.yaml")
	c.Flags().BoolVar(&nonInteractive, "non-interactive", false, "do not prompt; missing inputs become SKIP")
	c.Flags().BoolVar(&yes, "yes", false, "auto-accept discovered defaults")
	c.Flags().StringVar(&licenseFile, "license-file", "", "license file path (overrides discovery)")
	c.Flags().StringSliceVar(&hostnames, "hostnames", nil, "comma-separated key=value pairs: workbench=...,connect=...,ppm=...")
	c.Flags().StringVar(&idp, "idp", "", "identity provider: ldap|saml|oidc|none")
	c.Flags().StringSliceVar(&outputs, "output", []string{"md", "json"}, "comma-separated outputs to write: md,json")
	c.Flags().StringVar(&severityMin, "severity-min", "", "drop checks below this severity (info|warning|blocking)")
	c.Flags().StringSliceVar(&tagsAny, "tags", nil, "only run checks tagged with ALL of these")
	c.Flags().StringSliceVar(&skipTags, "skip-tags", nil, "skip checks tagged with any of these")
	c.Flags().StringSliceVar(&skipChecks, "skip-checks", nil, "skip checks by ID")
	c.Flags().BoolVar(&noCmdLog, "no-cmdlog", false, "do not write the replayable shell command log")
	return c
}

func mustGetString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func profileToProducts(profile string) []string {
	switch profile {
	case "single-server":
		return []string{"workbench", "connect", "packagemanager"}
	case "workbench":
		return []string{"workbench"}
	case "connect":
		return []string{"connect"}
	case "ppm":
		return []string{"packagemanager"}
	}
	return nil
}

// buildInputs collects every input the catalog references: per-product
// hostname/cert/key paths, IdP metadata URL, SMTP host, license file path.
// Order of precedence: explicit CLI flag > prompt answer > discovered default.
// The prompt driver decides at runtime whether to actually show a TUI prompt
// (interactive), silently take the discovered default (yes / non-interactive),
// or downgrade to yes when stdin is not a TTY (e.g. CI, piped runs).
func buildInputs(
	licenseFile string,
	hostnamePairs []string,
	idp string,
	facts discover.HostFacts,
	selected []string,
	d prompt.Driver,
) map[string]string {
	in := map[string]string{}
	if licenseFile != "" {
		in["license_file"] = licenseFile
	}

	defaultHost := facts.FQDN
	if defaultHost == "" {
		defaultHost = facts.Hostname
	}

	// Per-product hostname: flag wins; otherwise use the host's FQDN as the
	// prompt default. Customers running a multi-host install will override.
	hostnameOverrides := map[string]string{}
	for _, kv := range hostnamePairs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || v == "" {
			continue
		}
		hostnameOverrides[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	type productPrompt struct{ key, label, defaultCert, defaultKey string }
	products := []productPrompt{
		{"workbench", "Workbench", "/etc/rstudio/workbench.crt", "/etc/rstudio/workbench.key"},
		{"connect", "Connect", "/etc/rstudio-connect/connect.crt", "/etc/rstudio-connect/connect.key"},
		{"ppm", "Package Manager", "/etc/rstudio-pm/pm.crt", "/etc/rstudio-pm/pm.key"},
	}
	wanted := map[string]bool{}
	for _, s := range selected {
		// catalog uses "packagemanager" but flag/input keys use "ppm".
		switch s {
		case "packagemanager":
			wanted["ppm"] = true
		default:
			wanted[s] = true
		}
	}

	for _, p := range products {
		if !wanted[p.key] {
			continue
		}
		// Hostname.
		def := defaultHost
		if v, ok := hostnameOverrides[p.key]; ok {
			def = v
		}
		got, _ := d.Input(p.label+" public hostname:", def)
		in[p.key+"_hostname"] = got

		// Cert + key. Skip-prompted by typing "skip" in interactive mode
		// (handled inside prompt.ErrSkipped), which leaves the input empty
		// and SKIPs the dependent x509 checks.
		certPath, _ := d.Input(p.label+" SSL cert path (or 'skip'):", p.defaultCert)
		in[p.key+"_cert"] = certPath
		keyPath, _ := d.Input(p.label+" SSL key path (or 'skip'):", p.defaultKey)
		in[p.key+"_key"] = keyPath
	}

	// IdP type + metadata URL.
	if idp == "" {
		idp, _ = d.Select("Identity provider for Workbench/Connect:", []string{"none", "saml", "oidc", "ldap"}, "none")
	}
	in["idp"] = idp
	if wanted["workbench"] && idp != "none" && idp != "" {
		def := ""
		switch idp {
		case "saml":
			def = "https://idp.example.com/saml/metadata"
		case "oidc":
			def = "https://idp.example.com/.well-known/openid-configuration"
		}
		got, _ := d.Input("IdP metadata or discovery URL (or 'skip'):", def)
		in["idp_metadata_url"] = got
	}

	// SMTP host for Connect.
	if wanted["connect"] {
		got, _ := d.Input("Outbound SMTP host for Connect (or 'skip'):", "smtp.example.com")
		in["connect_smtp_host"] = got
	}

	return in
}

func wantOutputs(out []string) (md, j bool) {
	if len(out) == 0 {
		return true, true
	}
	for _, x := range out {
		switch strings.TrimSpace(x) {
		case "md", "markdown":
			md = true
		case "json":
			j = true
		}
	}
	return md, j
}
