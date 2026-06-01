package acmeprotocol

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	stepacme "github.com/smallstep/certificates/acme"
)

func link(url, rel string) string {
	return fmt.Sprintf("<%s>;rel=%q", url, rel)
}

func registerChallengeValidationRoute(r chi.Router, provisionerName string, linker *FlatLinker) {
	commonMiddleware := func(next nextHTTP) nextHTTP {
		return func(w http.ResponseWriter, req *http.Request) {
			linker.Middleware(http.HandlerFunc(checkPrerequisites(next))).ServeHTTP(w, req)
		}
	}
	validatingMiddleware := func(next nextHTTP) nextHTTP {
		return commonMiddleware(addNonce(addDirLink(verifyContentType(parseJWS(validateJWS(next))))))
	}
	extractPayloadByKid := func(next nextHTTP) nextHTTP {
		return validatingMiddleware(lookupJWK(verifyAndExtractJWSPayload(next)))
	}

	path := stepacme.GetUnescapedPathSuffix(stepacme.ChallengeLinkType, provisionerName, "{authzID}", "{chID}")
	r.MethodFunc(http.MethodPost, path, extractPayloadByKid(GetChallenge))
}
