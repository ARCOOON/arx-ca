package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that applies Cross-Origin Resource Sharing headers.
func CORS(allowedOrigins, allowedMethods []string, next http.Handler) http.Handler {
	origins := append([]string(nil), allowedOrigins...)
	methods := append([]string(nil), allowedMethods...)
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	allowMethods := strings.Join(methods, ", ")
	allowedHeaders := "Authorization, Content-Type, Accept, X-API-Key"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && corsOriginAllowed(origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", corsAllowOriginValue(origins, origin))
			w.Header().Add("Vary", "Origin")
		} else if corsAllowsAnyOrigin(origins) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func corsAllowsAnyOrigin(origins []string) bool {
	for _, o := range origins {
		if strings.TrimSpace(o) == "*" {
			return true
		}
	}
	return false
}

func corsOriginAllowed(origins []string, requestOrigin string) bool {
	if corsAllowsAnyOrigin(origins) {
		return true
	}
	for _, o := range origins {
		if strings.EqualFold(strings.TrimSpace(o), requestOrigin) {
			return true
		}
	}
	return false
}

func corsAllowOriginValue(origins []string, requestOrigin string) string {
	if corsAllowsAnyOrigin(origins) {
		return "*"
	}
	return requestOrigin
}
