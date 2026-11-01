package buildinfo

// These values are overridden at build time with -ldflags "-X".
var (
	Version   = "0.1.0-dev"
	Commit    = "local"
	BuildDate = "unknown"
	Repo      = ""
)

// IsRelease reports whether this binary was built by the release workflow.
func IsRelease() bool {
	return Repo != "" && Version != "" && Version != "0.1.0-dev" && Version != "dev"
}

// UpdateRepo returns the GitHub repository slug used for application updates.
func UpdateRepo() string {
	return Repo
}
