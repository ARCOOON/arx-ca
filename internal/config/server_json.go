package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// UnmarshalJSON accepts Go duration strings (e.g. "15s") for timeout fields so
// PUT /api/v1/settings/config matches the string values returned by GET.
func (s *ServerSettings) UnmarshalJSON(data []byte) error {
	type serverSettingsJSON struct {
		Host         string          `json:"host"`
		Port         int             `json:"port"`
		LogLevel     string          `json:"log_level"`
		ReadTimeout  json.RawMessage `json:"read_timeout"`
		WriteTimeout json.RawMessage `json:"write_timeout"`
		TLS          ServerTLSConfig `json:"tls"`
	}

	var raw serverSettingsJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.Host = raw.Host
	s.Port = raw.Port
	s.LogLevel = raw.LogLevel
	s.TLS = raw.TLS

	readTimeout, err := parseDurationField(raw.ReadTimeout)
	if err != nil {
		return fmt.Errorf("read_timeout: %w", err)
	}
	if readTimeout > 0 {
		s.ReadTimeout = readTimeout
	}

	writeTimeout, err := parseDurationField(raw.WriteTimeout)
	if err != nil {
		return fmt.Errorf("write_timeout: %w", err)
	}
	if writeTimeout > 0 {
		s.WriteTimeout = writeTimeout
	}

	return nil
}

func parseDurationField(raw json.RawMessage) (time.Duration, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		asString = strings.TrimSpace(asString)
		if asString == "" {
			return 0, nil
		}
		return time.ParseDuration(asString)
	}

	var asNumber int64
	if err := json.Unmarshal(raw, &asNumber); err == nil {
		return time.Duration(asNumber), nil
	}

	return 0, fmt.Errorf("invalid duration value %s", string(raw))
}
