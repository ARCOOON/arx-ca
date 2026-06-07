package models

// UpdaterChangelogResponse carries release notes for the running binary version.
type UpdaterChangelogResponse struct {
	Version  string `json:"version"`
	Markdown string `json:"markdown"`
}
