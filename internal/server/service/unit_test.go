package service

import (
	"strings"
	"testing"
)

func TestRenderUnitFile(t *testing.T) {
	content, err := renderUnitFile(UnitParams{
		ExecPath:   "/opt/arx/bin/arx-ca-server",
		ConfigPath: "/opt/arx/config/server.yaml",
		WorkingDir: "/opt/arx/bin",
	})
	if err != nil {
		t.Fatalf("renderUnitFile: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"ExecStart=/opt/arx/bin/arx-ca-server --config /opt/arx/config/server.yaml",
		"WorkingDirectory=/opt/arx/bin",
		"User=arx-ca",
		"AmbientCapabilities=CAP_NET_BIND_SERVICE",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("unit file missing %q:\n%s", want, text)
		}
	}
}
