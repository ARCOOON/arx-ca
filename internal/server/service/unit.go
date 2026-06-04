package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const unitName = "arx"

type unitTarget struct {
	filePath string
	userMode bool
}

func unitTargetForScope(scope InstallScope) (unitTarget, error) {
	if scope.IsSystem() {
		return unitTarget{filePath: "/etc/systemd/system/arx.service", userMode: false}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return unitTarget{}, fmt.Errorf("resolve home directory: %w", err)
	}
	path := filepath.Join(home, ".config", "systemd", "user", "arx.service")
	return unitTarget{filePath: path, userMode: true}, nil
}

var systemUnitTemplate = template.Must(template.New("systemUnit").Parse(`[Unit]
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

var userUnitTemplate = template.Must(template.New("userUnit").Parse(`[Unit]
Description=ARX Certificate Authority Server (user)
Documentation=https://github.com/ARCOOON/arx-ca
After=network-online.target

[Service]
Type=simple
WorkingDirectory={{.InstallDir}}
ExecStart={{.ExecPath}} server start --config {{.ConfigPath}}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`))

// UnitParams holds values substituted into the systemd unit template.
type UnitParams struct {
	RunAsUser  string
	InstallDir string
	ExecPath   string
	ConfigPath string
}

func renderUnitFile(params UnitParams, userMode bool) ([]byte, error) {
	var buf bytes.Buffer
	tmpl := systemUnitTemplate
	if userMode {
		tmpl = userUnitTemplate
	}
	if err := tmpl.Execute(&buf, params); err != nil {
		return nil, fmt.Errorf("render systemd unit: %w", err)
	}
	return buf.Bytes(), nil
}

func writeUnitFile(target unitTarget, params UnitParams) error {
	content, err := renderUnitFile(params, target.userMode)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure systemd unit directory: %w", err)
	}
	if err := os.WriteFile(target.filePath, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target.filePath, err)
	}
	return nil
}

func legacySystemUnitPath() string {
	return "/etc/systemd/system/arx-server.service"
}
