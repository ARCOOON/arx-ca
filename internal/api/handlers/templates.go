package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// TemplateHandler serves certificate template management endpoints.
type TemplateHandler struct {
	engine *ca.PKIEngine
}

// NewTemplateHandler constructs a TemplateHandler.
func NewTemplateHandler(engine *ca.PKIEngine) *TemplateHandler {
	return &TemplateHandler{engine: engine}
}

// Create handles POST /api/v1/templates.
func (h *TemplateHandler) Create() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.CreateCertificateTemplateRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		recordAuditAction(r, "TEMPLATE_CREATE")
		if ac := auditFromRequest(r); ac != nil {
			ac.PutMetadata("template_name", strings.TrimSpace(req.Name))
		}

		tpl, err := h.engine.CreateCertificateTemplate(req)
		if err != nil {
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "invalid template") ||
				strings.Contains(err.Error(), "already exists") {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("templates: create: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to create template")
			return
		}

		api.WriteSuccess(w, http.StatusCreated, tpl)
	})
}

// List handles GET /api/v1/templates.
func (h *TemplateHandler) List() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp, err := h.engine.ListCertificateTemplates()
		if err != nil {
			log.Printf("templates: list: %v", err)
			api.WriteError(w, http.StatusInternalServerError, "failed to list templates")
			return
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}
