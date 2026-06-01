package acmeprotocol

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	acmeAPI "github.com/smallstep/certificates/acme/api"
)

// NewRouter returns an ACME handler that exposes flat /acme/* paths while using
// the step-ca RFC 8555 implementation for JWS, nonce, and challenge validation.
func NewRouter(linker *FlatLinker, provisionerName string) chi.Router {
	inner := chi.NewRouter()
	registerChallengeValidationRoute(inner, provisionerName, linker)
	acmeAPI.Route(inner)

	adapter := &pathAdapter{
		provisionerName: provisionerName,
		inner:           inner,
	}

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return linker.Middleware(next)
	})
	router.Mount("/", adapter)
	return router
}

// pathAdapter rewrites flat ACME paths (/directory) to step-ca paths (/acme/directory).
type pathAdapter struct {
	provisionerName string
	inner           http.Handler
}

func (a *pathAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "" || path == "/" {
		path = "/directory"
	}
	if !strings.HasPrefix(path, "/"+a.provisionerName+"/") {
		if path == "/directory" || path == "/new-nonce" || path == "/new-account" ||
			path == "/new-order" || path == "/revoke-cert" || path == "/key-change" {
			path = "/" + a.provisionerName + path
		} else if strings.HasPrefix(path, "/account/") || strings.HasPrefix(path, "/order/") ||
			strings.HasPrefix(path, "/authz/") || strings.HasPrefix(path, "/challenge/") ||
			strings.HasPrefix(path, "/certificate/") {
			path = "/" + a.provisionerName + path
		}
	}

	r = r.Clone(r.Context())
	r.URL.Path = path

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provisionerID", a.provisionerName)
	if parts := strings.Split(strings.Trim(path, "/"), "/"); len(parts) >= 2 {
		switch parts[1] {
		case "account":
			if len(parts) >= 3 {
				rctx.URLParams.Add("accID", parts[2])
			}
		case "order":
			if len(parts) >= 3 {
				rctx.URLParams.Add("ordID", parts[2])
			}
		case "authz":
			if len(parts) >= 3 {
				rctx.URLParams.Add("authzID", parts[2])
			}
		case "challenge":
			if len(parts) >= 4 {
				rctx.URLParams.Add("authzID", parts[2])
				rctx.URLParams.Add("chID", parts[3])
			}
		case "certificate":
			if len(parts) >= 3 {
				rctx.URLParams.Add("certID", parts[2])
			}
		}
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	a.inner.ServeHTTP(w, r.WithContext(ctx))
}
