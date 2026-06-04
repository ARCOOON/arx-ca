// Package version holds the build version injected at link time.
package version

// Version is the semantic version of the running binary (e.g. v1.2.3).
// When unset at build time, it defaults to a development placeholder.
var Version = "v0.0.0-dev"

// Default is returned when Version is empty after link.
const Default = "v0.0.0-dev"

// Current returns the effective version string for this binary.
func Current() string {
	if Version == "" {
		return Default
	}
	return Version
}
