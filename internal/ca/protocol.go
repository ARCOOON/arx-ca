package ca

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// protocolContextHandler injects the PKI base context into protocol requests (ACME, SCEP, NDES).
type protocolContextHandler struct {
	engine *PKIEngine
	router http.Handler
}

func (h *protocolContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.engine != nil {
		r = r.WithContext(h.engine.buildBaseContext())
	}
	h.router.ServeHTTP(w, r)
}

func newChiProtocolHandler(engine *PKIEngine, router chi.Router) *protocolContextHandler {
	return &protocolContextHandler{engine: engine, router: router}
}
