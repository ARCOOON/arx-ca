package events

// Payload field keys shared across event subscribers.
const (
	KeyAction      = "action"
	KeySerial      = "serial"
	KeyAlias       = "alias"
	KeyCustomID    = "custom_id"
	KeyCommonName  = "common_name"
	KeyProvisioner = "provisioner"
	KeyFingerprint = "fingerprint"
	KeyIPAddress   = "ip_address"
	KeyHTTPMethod  = "http_method"
	KeyEndpoint    = "endpoint"
	KeyStatusCode  = "status_code"
	KeyRequestID   = "request_id"
	KeyActorType   = "actor_type"
	KeyActorID     = "actor_id"
	KeyActorRoles  = "actor_roles"
	KeyMetadata    = "metadata"
	KeyTimestamp   = "timestamp"
	KeyListenAddr  = "listen_addr"
	KeyUpdated     = "updated"
	KeySource      = "source"
	KeyVersion     = "version"
	KeyChannel     = "channel"
)

// PayloadCertIssued builds the required schema for certificate issuance events.
func PayloadCertIssued(serial, alias, customID, provisioner, fingerprint string) map[string]any {
	return map[string]any{
		KeySerial:      serial,
		KeyAlias:       alias,
		KeyCustomID:    customID,
		KeyProvisioner: provisioner,
		KeyFingerprint: fingerprint,
	}
}

// PayloadCertRevoked builds the required schema for certificate revocation events.
func PayloadCertRevoked(serial, alias, customID string, reasonCode int, reason string) map[string]any {
	return map[string]any{
		KeySerial:     serial,
		KeyAlias:      alias,
		KeyCustomID:   customID,
		"reason":      reason,
		"reason_code": reasonCode,
	}
}

// PayloadAuditRecorded builds the schema for persisted HTTP or system audit entries.
func PayloadAuditRecorded(
	action, requestID, ipAddress, httpMethod, endpoint string,
	statusCode int,
	actorType, actorID string,
	actorRoles []string,
	provisioner, fingerprint string,
	metadata map[string]any,
) map[string]any {
	payload := map[string]any{
		KeyAction:      action,
		KeyRequestID:   requestID,
		KeyIPAddress:   ipAddress,
		KeyHTTPMethod:  httpMethod,
		KeyEndpoint:    endpoint,
		KeyStatusCode:  statusCode,
		KeyActorType:   actorType,
		KeyActorID:     actorID,
		KeyProvisioner: provisioner,
		KeyFingerprint: fingerprint,
	}
	if len(actorRoles) > 0 {
		payload[KeyActorRoles] = append([]string(nil), actorRoles...)
	}
	if len(metadata) > 0 {
		payload[KeyMetadata] = metadata
	}
	return payload
}

// PayloadSystemEvent builds metadata for system-originated events.
func PayloadSystemEvent(listenAddr string, extra map[string]any) map[string]any {
	payload := map[string]any{
		KeyListenAddr: listenAddr,
	}
	for k, v := range extra {
		payload[k] = v
	}
	return payload
}

// PayloadConfigUpdated builds metadata for configuration hot-reload events.
func PayloadConfigUpdated(path, source, actor string, sections []string) map[string]any {
	return map[string]any{
		"path":     path,
		KeySource:  source,
		"actor":    actor,
		KeyUpdated: sections,
	}
}
