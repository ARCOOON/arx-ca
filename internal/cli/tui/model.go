package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	cliapi "github.com/ARCOOON/arx-ca/internal/cli/api"
	"github.com/ARCOOON/arx-ca/internal/models"
)

type view int

const (
	viewDashboard view = iota
	viewCertificates
	viewProvisioners
)

type provisionerRow struct {
	Name    string
	Type    string
	Enabled bool
	Detail  string
}

type model struct {
	client *cliapi.Client
	width  int
	height int

	activeView view

	health      *models.HealthReport
	healthErr   error
	loadingDash bool

	certs         []models.CertificateSummary
	certCursor    int
	certsErr      error
	loadingCerts  bool
	confirmRevoke bool
	revokeBusy    bool
	flash         string
	flashIsErr    bool

	provisioners    []provisionerRow
	provCursor      int
	provisionersErr error
	loadingProv     bool
}

type healthLoadedMsg struct {
	report *models.HealthReport
	err    error
}

type certsLoadedMsg struct {
	list *models.ListCertificatesResponse
	err  error
}

type provLoadedMsg struct {
	rows []provisionerRow
	err  error
}

type revokeDoneMsg struct {
	serial string
	err    error
}

// Run starts the Bubble Tea TUI.
func Run(client *cliapi.Client) error {
	m := model{
		client:       client,
		activeView:   viewDashboard,
		loadingDash:  true,
		loadingCerts: true,
		loadingProv:  true,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchHealth(),
		m.fetchCerts(),
		m.fetchProvisioners(),
	)
}

func (m model) fetchHealth() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		report, err := m.client.Health(ctx)
		return healthLoadedMsg{report: report, err: err}
	}
}

func (m model) fetchCerts() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		list, err := m.client.ListCertificates(ctx)
		return certsLoadedMsg{list: list, err: err}
	}
}

func (m model) fetchProvisioners() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var rows []provisionerRow
		var firstErr error

		if acme, err := m.client.ACMEStatus(ctx); err != nil {
			firstErr = err
		} else {
			detail := "disabled"
			if acme.Enabled {
				detail = acme.DirectoryURL
				if detail == "" {
					detail = strings.Join(acme.Challenges, ", ")
				}
			}
			name := acme.Provisioner
			if name == "" {
				name = "acme"
			}
			rows = append(rows, provisionerRow{
				Name: name, Type: "ACME", Enabled: acme.Enabled, Detail: detail,
			})
		}

		if scep, err := m.client.SCEPStatus(ctx); err != nil && firstErr == nil {
			firstErr = err
		} else if scep != nil {
			detail := scep.BaseURL
			if detail == "" {
				detail = scep.ChallengeHint
			}
			name := scep.Provisioner
			if name == "" {
				name = "scep"
			}
			rows = append(rows, provisionerRow{
				Name: name, Type: "SCEP", Enabled: scep.Enabled, Detail: detail,
			})
		}

		if k8s, err := m.client.K8sStatus(ctx); err != nil && firstErr == nil {
			firstErr = err
		} else if k8s != nil {
			detail := k8s.ReviewMode
			if k8s.UsesAPI {
				detail += " · TokenReview API"
			}
			name := k8s.Provisioner
			if name == "" {
				name = "kubernetes"
			}
			rows = append(rows, provisionerRow{
				Name: name, Type: "Kubernetes SA", Enabled: k8s.Enabled, Detail: detail,
			})
		}

		if ndes, err := m.client.NDESStatus(ctx); err != nil && firstErr == nil {
			firstErr = err
		} else if ndes != nil {
			detail := ndes.SCEPEndpoint
			if len(ndes.Connectors) > 0 {
				detail = strings.Join(ndes.Connectors, ", ")
			}
			rows = append(rows, provisionerRow{
				Name: "ndes", Type: "NDES", Enabled: ndes.Enabled, Detail: detail,
			})
		}

		return provLoadedMsg{rows: rows, err: firstErr}
	}
}

func (m model) revokeSelected() tea.Cmd {
	if m.certCursor < 0 || m.certCursor >= len(m.certs) {
		return nil
	}
	serial := m.certs[m.certCursor].Serial
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := m.client.RevokeCertificate(ctx, serial, "")
		return revokeDoneMsg{serial: serial, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.activeView = viewDashboard
			m.confirmRevoke = false
			return m, nil
		case "2":
			m.activeView = viewCertificates
			m.confirmRevoke = false
			return m, nil
		case "3":
			m.activeView = viewProvisioners
			m.confirmRevoke = false
			return m, nil
		case "tab":
			m.activeView = (m.activeView + 1) % 3
			m.confirmRevoke = false
			return m, nil
		case "r":
			if m.activeView == viewCertificates && !m.revokeBusy && len(m.certs) > 0 {
				if m.confirmRevoke {
					return m, nil
				}
				c := m.certs[m.certCursor]
				if c.Revoked {
					m.flash = "certificate is already revoked"
					m.flashIsErr = true
					return m, nil
				}
				m.confirmRevoke = true
				m.flash = ""
				return m, nil
			}
		case "y":
			if m.activeView == viewCertificates && m.confirmRevoke && !m.revokeBusy {
				m.revokeBusy = true
				m.confirmRevoke = false
				return m, m.revokeSelected()
			}
		case "n", "esc":
			m.confirmRevoke = false
			m.flash = ""
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			return m, nil
		}

	case healthLoadedMsg:
		m.loadingDash = false
		m.health = msg.report
		m.healthErr = msg.err
		return m, nil

	case certsLoadedMsg:
		m.loadingCerts = false
		m.certsErr = msg.err
		if msg.list != nil {
			m.certs = msg.list.Certificates
			if m.certCursor >= len(m.certs) {
				m.certCursor = max(0, len(m.certs)-1)
			}
		}
		return m, nil

	case provLoadedMsg:
		m.loadingProv = false
		m.provisioners = msg.rows
		m.provisionersErr = msg.err
		if m.provCursor >= len(m.provisioners) {
			m.provCursor = max(0, len(m.provisioners)-1)
		}
		return m, nil

	case revokeDoneMsg:
		m.revokeBusy = false
		if msg.err != nil {
			m.flash = msg.err.Error()
			m.flashIsErr = true
			return m, nil
		}
		m.flash = fmt.Sprintf("revoked %s", msg.serial)
		m.flashIsErr = false
		m.loadingCerts = true
		return m, m.fetchCerts()
	}

	return m, nil
}

func (m *model) moveCursor(delta int) {
	switch m.activeView {
	case viewCertificates:
		if len(m.certs) == 0 {
			return
		}
		m.certCursor = clamp(m.certCursor+delta, 0, len(m.certs)-1)
	case viewProvisioners:
		if len(m.provisioners) == 0 {
			return
		}
		m.provCursor = clamp(m.provCursor+delta, 0, len(m.provisioners)-1)
	}
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("arx · Super Admin"))
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	switch m.activeView {
	case viewDashboard:
		b.WriteString(m.viewDashboard())
	case viewCertificates:
		b.WriteString(m.viewCertificates())
	case viewProvisioners:
		b.WriteString(m.viewProvisioners())
	}

	if m.flash != "" {
		b.WriteString("\n")
		if m.flashIsErr {
			b.WriteString(styleFlashErr.Render(m.flash))
		} else {
			b.WriteString(styleFlashOK.Render(m.flash))
		}
	}

	b.WriteString("\n")
	b.WriteString(styleHelp.Render(m.helpLine()))
	return styleApp.Render(b.String())
}

func (m model) renderTabs() string {
	tabs := []struct {
		key  string
		name string
		v    view
	}{
		{"1", "Dashboard", viewDashboard},
		{"2", "Certificates", viewCertificates},
		{"3", "Provisioners", viewProvisioners},
	}
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		label := fmt.Sprintf("%s %s", t.key, t.name)
		if m.activeView == t.v {
			parts[i] = styleTabActive.Render(label)
		} else {
			parts[i] = styleTab.Render(label)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m model) viewDashboard() string {
	if m.loadingDash {
		return stylePanel.Render("Loading server health…")
	}
	if m.healthErr != nil {
		return stylePanel.Render(styleStatusErr.Render("Error: " + m.healthErr.Error()))
	}
	if m.health == nil {
		return stylePanel.Render("No health data.")
	}

	h := m.health
	lines := []string{
		kv("API Status", statusStyle(h.API.Status)),
		kv("API Version", h.API.Version),
		kv("CA Status", statusStyle(h.CABackend.Status)),
		kv("CA Engine", h.CABackend.Engine),
		kv("CA Initialized", fmt.Sprintf("%v", h.CABackend.Initialized)),
		kv("CA Message", h.CABackend.Message),
		kv("Uptime", h.Uptime.Human),
		kv("Uptime (sec)", fmt.Sprintf("%d", h.Uptime.Seconds)),
		kv("Goroutines", fmt.Sprintf("%d", h.Memory.Goroutines)),
	}
	return stylePanel.Render(strings.Join(lines, "\n"))
}

func (m model) viewCertificates() string {
	if m.loadingCerts {
		return stylePanel.Render("Loading certificates…")
	}
	if m.certsErr != nil {
		return stylePanel.Render(styleStatusErr.Render("Error: " + m.certsErr.Error()))
	}
	if len(m.certs) == 0 {
		return stylePanel.Render("No issued certificates found.")
	}

	var rows []string
	for i, c := range m.certs {
		status := styleStatusOK.Render("active")
		if c.Revoked {
			status = styleStatusErr.Render("revoked")
		}
		subject := c.Subject
		if len(subject) > 40 {
			subject = subject[:37] + "…"
		}
		line := fmt.Sprintf("%-36s %-10s %s", truncate(c.Serial, 18), status, subject)
		if i == m.certCursor {
			rows = append(rows, styleRowSelected.Render("▸ "+line))
		} else {
			rows = append(rows, styleRow.Render("  "+line))
		}
	}

	header := styleLabel.Render("Serial") + "  " +
		styleLabel.Copy().Width(10).Render("Status") + "  " +
		styleLabel.Render("Subject")
	body := strings.Join(rows, "\n")
	panel := header + "\n" + strings.Repeat("─", 60) + "\n" + body

	if m.confirmRevoke {
		serial := m.certs[m.certCursor].Serial
		panel += "\n\n" + styleStatusWarn.Render(fmt.Sprintf("Revoke %s? Press y to confirm, n to cancel.", serial))
	}
	return stylePanel.Render(panel)
}

func (m model) viewProvisioners() string {
	if m.loadingProv {
		return stylePanel.Render("Loading provisioners…")
	}
	if m.provisionersErr != nil {
		return stylePanel.Render(styleStatusErr.Render("Error: " + m.provisionersErr.Error()))
	}
	if len(m.provisioners) == 0 {
		return stylePanel.Render("No provisioners configured.")
	}

	var rows []string
	for i, p := range m.provisioners {
		state := styleStatusErr.Render("off")
		if p.Enabled {
			state = styleStatusOK.Render("on")
		}
		line := fmt.Sprintf("%-14s %-18s %-4s %s", p.Type, p.Name, state, truncate(p.Detail, 32))
		if i == m.provCursor {
			rows = append(rows, styleRowSelected.Render("▸ "+line))
		} else {
			rows = append(rows, styleRow.Render("  "+line))
		}
	}

	header := styleLabel.Render("Type") + "  " +
		styleLabel.Render("Name") + "  " +
		styleLabel.Render("State") + "  " +
		styleLabel.Render("Details")
	body := strings.Join(rows, "\n")
	return stylePanel.Render(header + "\n" + strings.Repeat("─", 60) + "\n" + body)
}

func (m model) helpLine() string {
	switch m.activeView {
	case viewDashboard:
		return "1-3 or Tab · switch view · q quit"
	case viewCertificates:
		if m.confirmRevoke {
			return "y confirm revoke · n cancel · q quit"
		}
		return "↑/↓ select · r revoke · 1-3 switch view · q quit"
	default:
		return "↑/↓ navigate · 1-3 switch view · q quit"
	}
}

func kv(label, value string) string {
	return styleLabel.Render(label) + styleValue.Render(value)
}

func statusStyle(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy", "ok", "ready", "running":
		return styleStatusOK.Render(status)
	case "degraded", "warning":
		return styleStatusWarn.Render(status)
	default:
		if status == "" {
			return styleStatusWarn.Render("unknown")
		}
		return styleStatusErr.Render(status)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
