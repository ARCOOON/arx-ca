package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/ARCOOON/arx-ca/internal/api"
	"github.com/ARCOOON/arx-ca/internal/ca"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/models"
)

// ACMEHandler serves ACME administration endpoints.
type ACMEHandler struct {
	engine     *ca.PKIEngine
	listenHost string
}

// NewACMEHandler constructs an ACME handler bound to the PKI engine.
func NewACMEHandler(engine *ca.PKIEngine, listenHost string) *ACMEHandler {
	return &ACMEHandler{
		engine:     engine,
		listenHost: listenHost,
	}
}

// CreateEABKey handles POST /api/v1/acme/eab-keys.
func (h *ACMEHandler) CreateEABKey() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req models.CreateACMEEABKeyRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			api.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		recordAuditAction(r, db.ActionEABGenerate)
		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(strings.TrimSpace(req.Provisioner))
			if req.Reference != "" {
				ac.PutMetadata("reference", req.Reference)
			}
		}

		resp, err := h.engine.CreateACMEEABKey(r.Context(), req)
		if err != nil {
			if isACMEEABClientError(err) {
				api.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			status, message := ca.MapCAError(err)
			if status >= http.StatusInternalServerError {
				log.Printf("acme: create eab key: %v", err)
			}
			api.WriteError(w, status, message)
			return
		}

		if ac := auditFromRequest(r); ac != nil {
			ac.SetProvisioner(resp.Provisioner)
			ac.PutMetadata("eab_key_id", resp.KeyID)
		}

		api.WriteSuccess(w, http.StatusCreated, resp)
	})
}

// Status handles GET /api/v1/acme/status.
func (h *ACMEHandler) Status() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp := models.ACMEStatusResponse{
			Enabled:             h.engine.ACMEEnabled(),
			Provisioner:         "acme",
			Challenges:          h.engine.ACMEConfiguredChallenges(),
			RequireEAB:          h.engine.ACMEEABRequired(),
			DeviceAttestEnabled: h.engine.ACMEDeviceAttestationEnabled(),
			AttestationFormats:  h.engine.ACMEAttestationFormats(),
		}
		if resp.Enabled {
			resp.DirectoryURL = h.engine.ACMEDirectoryURL(h.listenHost)
			resp.DNSName = h.engine.ACMEDNS()
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// SCEPStatus handles GET /api/v1/scep/status.
func (h *ACMEHandler) SCEPStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp := models.SCEPStatusResponse{
			Enabled:     h.engine.SCEPEnabled(),
			Provisioner: h.engine.SCEPProvisionerName(),
		}
		if resp.Enabled {
			resp.BaseURL = h.engine.SCEPBaseURL(h.listenHost)
			if h.engine.SCEPChallengeConfigured() {
				resp.ChallengeHint = "configured"
			}
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

// NDESStatus handles GET /api/v1/ndes/status.
func (h *ACMEHandler) NDESStatus() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		resp := models.NDESStatusResponse{
			Enabled:        h.engine.NDESEnabled(),
			ADCSCompatible: h.engine.NDESEnabled(),
		}
		if resp.Enabled {
			resp.SCEPEndpoint = ca.NDESMSCEPURL(h.listenHost)
			resp.AdminEndpoint = h.engine.NDESAdminURL(h.listenHost)
			if reg := h.engine.NDESRegistryRef(); reg != nil {
				resp.Connectors = reg.Names()
			}
		}

		api.WriteSuccess(w, http.StatusOK, resp)
	})
}

func isACMEEABClientError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "not an ACME provisioner") ||
		strings.Contains(msg, "reference must not") ||
		strings.Contains(msg, "ACME database")
}
