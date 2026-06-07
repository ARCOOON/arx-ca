#!/bin/sh
# Poll GET /api/v1/ca/info after obtaining a short-lived admin JWT.
# Credentials are read from ARX_HEALTHCHECK_EMAIL and ARX_HEALTHCHECK_PASSWORD.
set -eu

API_BASE="${ARX_HEALTHCHECK_URL:-http://127.0.0.1:8080}"

EMAIL="${ARX_HEALTHCHECK_EMAIL:-}"
PASSWORD="${ARX_HEALTHCHECK_PASSWORD:-}"

if [ -z "${EMAIL}" ] || [ -z "${PASSWORD}" ]; then
	echo "docker-healthcheck: ARX_HEALTHCHECK_EMAIL and ARX_HEALTHCHECK_PASSWORD must be set" >&2
	exit 1
fi

LOGIN_PAYLOAD=$(printf '{"email":"%s","password":"%s"}' "${EMAIL}" "${PASSWORD}")

RESPONSE=$(wget -qO- \
	--header='Content-Type: application/json' \
	--post-data="${LOGIN_PAYLOAD}" \
	"${API_BASE}/api/v1/auth/login")

TOKEN=$(printf '%s' "${RESPONSE}" | jq -er '.data.token')
if [ -z "${TOKEN}" ] || [ "${TOKEN}" = "null" ]; then
	echo "docker-healthcheck: login did not return a token" >&2
	exit 1
fi

wget -q --header="Authorization: Bearer ${TOKEN}" --spider "${API_BASE}/api/v1/ca/info"
exit 0
