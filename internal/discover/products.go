package discover

import "github.com/posit-dev/pev/internal/system"

// Products records which Posit professional products appear installed on the host.
type Products struct {
	Workbench      bool `json:"workbench"`
	Connect        bool `json:"connect"`
	PackageManager bool `json:"package_manager"`
}

// DetectProducts probes by binary presence. License-manager binaries are
// authoritative for Connect/PPM; rstudio-server on $PATH for Workbench.
func DetectProducts() Products {
	return Products{
		Workbench:      system.CommandExists("rstudio-server") || system.FileExists("/usr/lib/rstudio-server/bin/rserver"),
		Connect:        system.FileExists("/opt/rstudio-connect/bin/rstudio-connect"),
		PackageManager: system.FileExists("/opt/rstudio-pm/bin/rstudio-pm"),
	}
}

// SelectedFromFlag merges user flag input with discovery: if the flag list is
// empty, return whatever was detected; if the user named explicit products,
// honor that.
func SelectedFromFlag(flagList []string, p Products) []string {
	if len(flagList) > 0 {
		return flagList
	}
	out := []string{}
	if p.Workbench {
		out = append(out, "workbench")
	}
	if p.Connect {
		out = append(out, "connect")
	}
	if p.PackageManager {
		out = append(out, "packagemanager")
	}
	return out
}
