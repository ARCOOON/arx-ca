package handlers

import (
	"log"
	"net/http"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/models"
	"github.com/ARCOOON/arx-ca/internal/updater"
	"github.com/ARCOOON/arx-ca/internal/version"
)

const releaseNotesFallback = "Release notes could not be fetched from GitHub."

// UpdaterHandler serves release metadata for the running binary.
type UpdaterHandler struct{}

// NewUpdaterHandler constructs an UpdaterHandler.
func NewUpdaterHandler() *UpdaterHandler {
	return &UpdaterHandler{}
}

// CurrentChangelog handles GET /api/v1/updater/current-changelog.
func (h *UpdaterHandler) CurrentChangelog() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		current := version.Current()
		markdown, err := updater.FetchReleaseNotes(current)
		if err != nil {
			log.Printf("updater: fetch release notes for %s: %v", current, err)
			markdown = releaseNotesFallback
		}

		api.WriteSuccess(w, http.StatusOK, models.UpdaterChangelogResponse{
			Version:  current,
			Markdown: markdown,
		})
	})
}
