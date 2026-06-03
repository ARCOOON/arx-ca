package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	unitFilePath = "/etc/systemd/system/arx-server.service"
	unitName     = "arx-server"
)

var unitTemplate = template.Must(template.New("unit").Parse(`[Unit]
Description=ARX Certificate Authority Server
Documentation=https://github.com/ARCOOON/arx-ca
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{.RunAsUser}}
Group={{.RunAsUser}}
WorkingDirectory={{.InstallDir}}
ExecStartPre=+/usr/bin/chown -R {{.RunAsUser}}:{{.RunAsUser}} {{.InstallDir}}
ExecStartPre=+/usr/bin/chmod 600 {{.ConfigPath}}
ExecStartPre=+/usr/bin/chmod 700 {{.ExecPath}}
ExecStart={{.ExecPath}} server start --config {{.ConfigPath}}
Restart=on-failure
RestartSec=5

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
`))

// UnitParams holds values substituted into the systemd unit template.
type UnitParams struct {
	RunAsUser  string
	InstallDir string
	ExecPath   string
	ConfigPath string
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
