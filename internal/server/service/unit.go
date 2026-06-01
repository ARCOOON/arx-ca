package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	unitFilePath = "/etc/systemd/system/arx-ca-server.service"
	unitName     = "arx-ca-server"
	systemUser   = "arx-ca"
)

var unitTemplate = template.Must(template.New("unit").Parse(`[Unit]
Description=ARX Certificate Authority Server
After=network.target postgresql.service

[Service]
Type=simple
User=arx-ca
Group=arx-ca
ExecStart={{.ExecPath}} server start --config {{.ConfigPath}}
WorkingDirectory={{.WorkingDir}}
Restart=always
RestartSec=5
# Security (Relaxed for dynamic paths)
NoNewPrivileges=yes
PrivateTmp=yes
AmbientCapabilities=CAP_NET_BIND_SERVICE
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`))

// UnitParams holds values substituted into the systemd unit template.
type UnitParams struct {
	ExecPath   string
	ConfigPath string
	WorkingDir string
}

func renderUnitFile(params UnitParams) ([]byte, error) {
	var buf bytes.Buffer
	if err := unitTemplate.Execute(&buf, params); err != nil {
		return nil, fmt.Errorf("render systemd unit: %w", err)
	}
	return buf.Bytes(), nil
}

func writeUnitFile(params UnitParams) error {
	content, err := renderUnitFile(params)
	if err != nil {
		return err
	}
	dir := filepath.Dir(unitFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure systemd unit directory: %w", err)
	}
	if err := os.WriteFile(unitFilePath, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", unitFilePath, err)
	}
	return nil
}
