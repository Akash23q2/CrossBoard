package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dashboardTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00ffe0"))
	dashboardMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9292a8"))
	dashboardKeyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff9c66"))
	dashboardPanelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#44445a")).Padding(0, 1)
)

type dashboardModel struct {
	logs          []appLogEntry
	width         int
	height        int
	status        string
	showShortcuts bool
	showTunnelQR  bool
	serverDone    <-chan error
}

type dashboardLogsMsg []appLogEntry
type dashboardActionMsg string
type dashboardServerErrorMsg error

func runTerminalDashboard(serverDone <-chan error) error {
	program := tea.NewProgram(dashboardModel{
		logs:       recentAppLogs(),
		status:     "CrossBoard is running",
		serverDone: serverDone,
	}, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(refreshDashboardLogs(), waitForDashboardServer(m.serverDone))
}

func (m dashboardModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case dashboardLogsMsg:
		m.logs = message
		return m, refreshDashboardLogs()
	case dashboardActionMsg:
		m.status = string(message)
	case dashboardServerErrorMsg:
		m.status = fmt.Sprintf("HTTP server stopped: %v", error(message))
		return m, tea.Quit
	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.showShortcuts = false
			m.showTunnelQR = false
			m.status = "CrossBoard is running"
		case "k":
			return m, doDashboardAction("scrcpy")
		case "l":
			return m, doDashboardAction("logs")
		case "r":
			m.status = "Dashboard refreshed"
			return m, func() tea.Msg { return tea.ClearScreen() }
		case "t":
			m.showTunnelQR = !m.showTunnelQR
		case "s":
			return m, doDashboardAction("settings")
		case "?":
			m.showShortcuts = !m.showShortcuts
		case "h":
			openDesktopPage("/help")
			m.status = "Help opened in your browser"
		}
	}
	return m, nil
}

func (m dashboardModel) View() string {
	if m.width == 0 {
		return "Starting CrossBoard..."
	}
	var out strings.Builder
	tunnel, _ := getServerURLs()
	out.WriteString(dashboardTitleStyle.Render("╭─ CROSSBOARD ─────────────────────────────────────────────────╮"))
	out.WriteString("\n")
	out.WriteString("  " + dashboardMutedStyle.Render(m.status) + "\n")
	if tunnel != "" {
		out.WriteString("  " + dashboardMutedStyle.Render("Cloudflare tunnel: "+tunnel) + "\n")
	}
	out.WriteString("\n")
	qrText := currentLANQRText()
	if m.showTunnelQR {
		qrText = currentTunnelQRText()
	}
	out.WriteString(dashboardPanelStyle.Render(qrText))
	out.WriteString("\n")
	out.WriteString(dashboardMutedStyle.Render("  ? see global shortcuts · t toggle LAN/tunnel QR") + "\n")
	if m.showShortcuts {
		out.WriteString("\n" + dashboardTitleStyle.Render("GLOBAL SHORTCUTS") + "\n")
		out.WriteString(renderGlobalShortcuts())
	}
	out.WriteString("\n" + dashboardTitleStyle.Render("RECENT LOGS") + "\n")

	logLimit := m.height - 19
	if m.showShortcuts {
		logLimit -= 8
	}
	if logLimit < 1 {
		logLimit = 1
	}
	logs := m.logs
	if len(logs) > logLimit {
		logs = logs[len(logs)-logLimit:]
	}
	for _, entry := range logs {
		line := fmt.Sprintf("%s  %s", entry.Time.Format("15:04:05"), entry.Message)
		out.WriteString(dashboardMutedStyle.Render(lipgloss.NewStyle().Width(max(m.width-4, 20)).Render(line)))
		out.WriteByte('\n')
	}
	if len(m.logs) == 0 {
		out.WriteString(dashboardMutedStyle.Render("  Waiting for log entries…\n"))
	}
	out.WriteString("\n" + dashboardMutedStyle.Render("k scrcpy · l logs · r refresh screen · s settings · ? global shortcuts · h help · q quit"))
	return out.String()
}

func renderGlobalShortcuts() string {
	lines := [][2]string{
		{"Alt+C", "send Ctrl+Shift+C to copy selected PC text, then send it to the registered app device"},
		{"Alt+V", "fetch app clipboard, copy it to the PC, then paste"},
		{"Alt+K", "connect over ADB TCP :2004 and start scrcpy"},
	}
	var out strings.Builder
	for _, line := range lines {
		out.WriteString("  " + dashboardKeyStyle.Render(line[0]) + "  " + line[1] + "\n")
	}
	return out.String()
}

func refreshDashboardLogs() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return dashboardLogsMsg(recentAppLogs())
	})
}

func waitForDashboardServer(serverDone <-chan error) tea.Cmd {
	return func() tea.Msg {
		if err := <-serverDone; err != nil {
			return dashboardServerErrorMsg(err)
		}
		return dashboardServerErrorMsg(fmt.Errorf("server stopped"))
	}
}

func doDashboardAction(action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "scrcpy":
			err = castActiveAppDevice()
		case "logs":
			openDesktopPage("/logs")
		case "settings":
			openDesktopPage("/settings")
		}
		if err != nil {
			appLogf("terminal action %s failed: %v", action, err)
			return dashboardActionMsg(action + " failed: " + err.Error())
		}
		return dashboardActionMsg(action + " complete")
	}
}

func currentLANQRText() string {
	serverURLs.RLock()
	lan := serverURLs.lan
	serverURLs.RUnlock()
	if lan != "" {
		return "LAN · " + lan + "\n\n" + renderQr(lan)
	}
	return "LAN QR is unavailable: no local network address was found."
}

func currentTunnelQRText() string {
	serverURLs.RLock()
	tunnel := serverURLs.tunnel
	serverURLs.RUnlock()
	if tunnel == "" {
		return "Cloudflare tunnel is unavailable.\nInstall cloudflared to enable remote access."
	}
	return "CLOUDFLARE · " + tunnel + "\n\n" + renderQr(tunnel)
}
