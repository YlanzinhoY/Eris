package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/enzom/hv-game-cli/internal/catalog"
	"github.com/enzom/hv-game-cli/internal/downloader"
	"github.com/enzom/hv-game-cli/internal/launcher"
	"github.com/enzom/hv-game-cli/internal/scanner"
)

var (
	colorPurple = lipgloss.Color("#9B7BFF")
	colorCyan   = lipgloss.Color("#45D6D0")
	colorGreen  = lipgloss.Color("#78E08F")
	colorMuted  = lipgloss.Color("#7C8293")
	colorText   = lipgloss.Color("#F4F4F5")
	colorPanel  = lipgloss.Color("#2B2D3A")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPurple).
			Padding(0, 2)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPanel).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	successStyle = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B81")).Bold(true)
)

type confirmationKind uint8

const (
	confirmationNone confirmationKind = iota
	confirmationDownload
	confirmationOpenLink
)

type model struct {
	games              []catalog.Game
	scanner            *scanner.Scanner
	cursor             int
	width              int
	height             int
	busy               bool
	scanning           bool
	downloading        bool
	confirmation       confirmationKind
	downloadRequested  bool
	status             string
	results            []scanner.Result
	err                error
	downloadedBytes    int64
	downloadTotalBytes int64
	downloadPath       string
	downloadEvents     <-chan tea.Msg
	downloadCancel     context.CancelFunc
}

type scanFinishedMsg struct {
	results []scanner.Result
	err     error
}

type linkOpenedMsg struct {
	err error
}

type downloadStartedMsg struct {
	events <-chan tea.Msg
}

type downloadProgressMsg struct {
	progress downloader.Progress
}

type downloadFinishedMsg struct {
	result downloader.Result
	err    error
}

func Run(games []catalog.Game, gameScanner *scanner.Scanner) error {
	program := tea.NewProgram(
		model{
			games:   games,
			scanner: gameScanner,
			status:  "Selecione um jogo para começar.",
		},
		tea.WithAltScreen(),
	)
	_, err := program.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height

	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+c", "q":
			if m.downloading && m.downloadCancel != nil {
				m.status = "Cancelando download..."
				m.downloadCancel()
				return m, nil
			}
			return m, tea.Quit
		case "up", "k":
			if !m.busy && m.confirmation == confirmationNone && m.cursor > 0 {
				m.cursor--
				m.resetSelectionState()
				return m, nil
			}
		case "down", "j":
			if !m.busy && m.confirmation == confirmationNone && m.cursor < len(m.games)-1 {
				m.cursor++
				m.resetSelectionState()
				return m, nil
			}
		case "s":
			if !m.busy && m.confirmation == confirmationNone {
				m.downloadRequested = false
				return m.startScan()
			}
		case "enter":
			if !m.busy && m.confirmation == confirmationNone {
				m.downloadRequested = true
				return m.startScan()
			}
		case "y":
			switch m.confirmation {
			case confirmationDownload:
				m.confirmation = confirmationNone
				return m.startDownload()
			case confirmationOpenLink:
				m.confirmation = confirmationNone
				m.busy = true
				m.status = "Abrindo o link no navegador..."
				return m, openLinkCmd(m.selectedGame().DownloadLink)
			}
		case "n", "esc":
			if m.confirmation != confirmationNone {
				m.confirmation = confirmationNone
				m.downloadRequested = false
				m.status = "Operação cancelada."
			}
		}

	case scanFinishedMsg:
		m.busy = false
		m.scanning = false
		m.results = message.results
		m.err = message.err
		if message.err != nil {
			m.status = "O scan não pôde ser concluído."
			m.downloadRequested = false
			break
		}
		if len(message.results) == 0 {
			m.status = "Jogo não encontrado na Steam nem nos discos locais."
			m.confirmation = confirmationOpenLink
		} else {
			m.status = fmt.Sprintf("Instalação encontrada via %s.", message.results[0].Source)
			if m.downloadRequested {
				m.confirmation = confirmationDownload
			}
		}

	case downloadStartedMsg:
		m.downloadEvents = message.events
		return m, waitDownloadEvent(message.events)

	case downloadProgressMsg:
		m.downloadedBytes = message.progress.Downloaded
		m.downloadTotalBytes = message.progress.Total
		m.status = downloadProgressLabel(message.progress)
		return m, waitDownloadEvent(m.downloadEvents)

	case downloadFinishedMsg:
		m.busy = false
		m.downloading = false
		m.downloadEvents = nil
		m.downloadRequested = false
		if m.downloadCancel != nil {
			m.downloadCancel()
			m.downloadCancel = nil
		}
		if message.err != nil {
			if errors.Is(message.err, context.Canceled) {
				m.err = nil
				m.status = "Download cancelado. O arquivo temporário foi removido."
			} else {
				m.err = fmt.Errorf("download: %w", message.err)
				m.status = "O download falhou."
			}
		} else {
			m.err = nil
			m.downloadPath = message.result.Path
			m.downloadedBytes = message.result.Bytes
			m.downloadTotalBytes = message.result.Bytes
			m.status = "Download concluído. Arquivo salvo na pasta do jogo."
		}

	case linkOpenedMsg:
		m.busy = false
		m.downloadRequested = false
		m.err = message.err
		if message.err != nil {
			m.status = "Não foi possível abrir o navegador."
		} else {
			m.status = "Link aberto no navegador padrão."
		}

	}

	return m, nil
}

func (m model) View() string {
	if len(m.games) == 0 {
		return errorStyle.Render("Nenhum jogo disponível.")
	}

	header := titleStyle.Render("ÉRIS") + "  " + mutedStyle.Render("catálogo de releases")
	listPanel := m.renderList()
	detailPanel := m.renderDetails()

	var body string
	if m.width >= 92 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, listPanel, "  ", detailPanel)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, listPanel, detailPanel)
	}

	status := mutedStyle.Render(m.status)
	if m.err != nil {
		status = errorStyle.Render(m.err.Error())
	}
	switch m.confirmation {
	case confirmationDownload:
		status = selectedStyle.Render("Baixar o arquivo para a pasta encontrada?  [y] sim  [n] não")
	case confirmationOpenLink:
		status = selectedStyle.Render("Instalação não encontrada. Abrir o link no navegador?  [y] sim  [n] não")
	}
	helpText := "↑/↓ navegar  •  s escanear  •  enter baixar  •  q sair"
	if m.downloading {
		helpText = "download em andamento  •  q cancelar"
	}
	help := mutedStyle.Render(helpText)

	return lipgloss.NewStyle().Padding(1, 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", status, help),
	)
}

func (m model) renderList() string {
	width := 43
	if m.width > 0 && m.width < 92 {
		width = max(32, m.width-8)
	}

	rows := make([]string, 0, len(m.games)+1)
	rows = append(rows, selectedStyle.Render("JOGOS DISPONÍVEIS"))
	for index, game := range m.games {
		style := lipgloss.NewStyle().Foreground(colorText)
		selected := index == m.cursor
		if selected {
			style = selectedStyle
		}
		rows = append(rows, style.Render(formatGameListRow(game, selected, width)))
	}
	return panelStyle.Width(width).Render(strings.Join(rows, "\n"))
}

func formatGameListRow(game catalog.Game, selected bool, width int) string {
	const versionGap = "  "

	prefix := "  "
	if selected {
		prefix = "› "
	}

	availableWidth := max(0, width-lipgloss.Width(prefix))
	version := truncateToWidth(game.Version, max(0, availableWidth-lipgloss.Width(versionGap)-1))
	nameWidth := max(0, availableWidth-lipgloss.Width(versionGap)-lipgloss.Width(version))
	name := truncateToWidth(game.Name, nameWidth)
	padding := strings.Repeat(" ", max(0, nameWidth-lipgloss.Width(name)))
	return prefix + name + padding + versionGap + version
}

func truncateToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	return ansi.Truncate(value, width, "…")
}

func (m model) renderDetails() string {
	game := m.selectedGame()
	installStatus := mutedStyle.Render("Ainda não escaneado")
	if m.scanning {
		installStatus = selectedStyle.Render("Steam primeiro; depois discos locais...")
	} else if len(m.results) > 0 {
		installStatus = successStyle.Render(m.results[0].Path)
	} else if strings.Contains(m.status, "não encontrado") {
		installStatus = errorStyle.Render("Não encontrado")
	}

	sections := []string{selectedStyle.Render(strings.ToUpper(game.Name))}
	sections = append(sections,
		"",
		mutedStyle.Render("VERSÃO"),
		game.Version,
		"",
		mutedStyle.Render("EXECUTÁVEL"),
		fallback(game.Executable, "Não informado"),
		"",
		mutedStyle.Render("INSTALAÇÃO"),
		installStatus,
		"",
		mutedStyle.Render("DOWNLOAD"),
		m.renderDownloadStatus(game),
	)
	width := 58
	if m.width > 0 && m.width < 92 {
		width = max(32, m.width-8)
	}
	return panelStyle.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

func (m model) renderDownloadStatus(game catalog.Game) string {
	if m.downloading {
		return renderProgressBar(m.downloadedBytes, m.downloadTotalBytes, 34)
	}
	if m.downloadPath != "" {
		return successStyle.Render(truncate(m.downloadPath, 52))
	}
	return truncate(game.DownloadLink, 52)
}

func (m model) selectedGame() catalog.Game {
	return m.games[m.cursor]
}

func (m model) startScan() (tea.Model, tea.Cmd) {
	m.busy = true
	m.scanning = true
	m.results = nil
	m.err = nil
	m.downloadPath = ""
	m.status = "Procurando na Steam; depois nos discos locais..."
	game := m.selectedGame()
	return m, scanCmd(m.scanner, game.Name, game.Executable)
}

func (m model) startDownload() (tea.Model, tea.Cmd) {
	if len(m.results) == 0 {
		m.confirmation = confirmationOpenLink
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.downloadCancel = cancel
	m.busy = true
	m.downloading = true
	m.downloadedBytes = 0
	m.downloadTotalBytes = 0
	m.downloadPath = ""
	m.err = nil
	destination := m.results[0].Path
	m.status = "Conectando ao servidor de download..."
	return m, beginDownloadCmd(ctx, m.selectedGame().DownloadLink, destination)
}

func (m *model) resetSelectionState() {
	m.results = nil
	m.err = nil
	m.downloadPath = ""
	m.downloadedBytes = 0
	m.downloadTotalBytes = 0
	m.status = "Selecione uma ação."
}

func scanCmd(gameScanner *scanner.Scanner, gameName string, executableName string) tea.Cmd {
	return func() tea.Msg {
		results, err := gameScanner.Scan(context.Background(), gameName, executableName)
		return scanFinishedMsg{results: results, err: err}
	}
}

func beginDownloadCmd(ctx context.Context, rawURL string, destination string) tea.Cmd {
	return func() tea.Msg {
		events := make(chan tea.Msg, 2)
		go func() {
			result, err := downloader.Download(ctx, rawURL, destination, func(progress downloader.Progress) {
				events <- downloadProgressMsg{progress: progress}
			})
			events <- downloadFinishedMsg{result: result, err: err}
			close(events)
		}()
		return downloadStartedMsg{events: events}
	}
}

func waitDownloadEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		message, ok := <-events
		if !ok {
			return downloadFinishedMsg{err: errors.New("canal de download encerrado inesperadamente")}
		}
		return message
	}
}

func openLinkCmd(rawURL string) tea.Cmd {
	return func() tea.Msg {
		return linkOpenedMsg{err: launcher.Open(rawURL)}
	}
}

func renderProgressBar(downloaded int64, total int64, width int) string {
	if total <= 0 {
		return selectedStyle.Render(fmt.Sprintf("Recebidos %s", formatBytes(downloaded)))
	}
	ratio := min(1, max(0, float64(downloaded)/float64(total)))
	filled := int(ratio * float64(width))
	bar := selectedStyle.Render(strings.Repeat("█", filled)) + mutedStyle.Render(strings.Repeat("░", width-filled))
	label := fmt.Sprintf("%s / %s  %3.0f%%", formatBytes(downloaded), formatBytes(total), ratio*100)
	return lipgloss.JoinVertical(lipgloss.Left, bar, label)
}

func downloadProgressLabel(progress downloader.Progress) string {
	if progress.Total <= 0 {
		return fmt.Sprintf("Baixando: %s recebidos", formatBytes(progress.Downloaded))
	}
	percentage := float64(progress.Downloaded) / float64(progress.Total) * 100
	return fmt.Sprintf("Baixando: %.1f%% — %s de %s", percentage, formatBytes(progress.Downloaded), formatBytes(progress.Total))
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := int64(unit)
	exponent := 0
	for quotient := value / unit; quotient >= unit && exponent < 4; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}

func fallback(value string, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return mutedStyle.Render(defaultValue)
	}
	return value
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit-1] + "…"
}
