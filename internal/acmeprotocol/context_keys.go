package acmeprotocol

import acmeAPI "github.com/smallstep/certificates/acme/api"

// Context keys match github.com/smallstep/certificates/acme/api middleware values.
const (
	contextKeyAccount = acmeAPI.ContextKey("acc")
	contextKeyJWK     = acmeAPI.ContextKey("jwk")
	contextKeyJWS     = acmeAPI.ContextKey("jws")
	contextKeyPayload = acmeAPI.ContextKey("payload")
)

type payloadInfo struct {
	value       []byte
	isPostAsGet bool
	isEmptyJSON bool
}
