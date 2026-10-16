package main

import (
	"bufio"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const appName = "MICMETER"

var (
	accent = lipgloss.Color("86")
	muted  = lipgloss.Color("241")
	good   = lipgloss.Color("42")
	warn   = lipgloss.Color("214")
	bad    = lipgloss.Color("196")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(1, 2)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(muted)
	labelStyle    = lipgloss.NewStyle().Foreground(muted)
)

var devicePattern = regexp.MustCompile(`^\[dshow[^]]*\]\s+"(.+)" \(audio\)`)
var rmsPattern = regexp.MustCompile(`lavfi\.astats\.Overall\.RMS_level=([-+]?\d+(?:\.\d+)?|[-+]?inf)`)

type deviceListMsg struct {
	devices []string
	err     error
}
type captureStartedMsg struct {
	cmd    *exec.Cmd
	levels chan float64
	err    error
}
type levelMsg float64
type captureEndedMsg struct{ err error }
type tickMsg time.Time

type model struct {
	ffmpeg    string
	devices   []string
	selected  int
	capturing bool
	level     float64
	peak      float64
	cmd       *exec.Cmd
	levels    chan float64
	status    string
	err       string
	width     int
	height    int
}

func initialModel(ffmpeg string) model {
	return model{ffmpeg: ffmpeg, level: -120, peak: -120, status: "Scanning for audio devices…"}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(listDevicesCmd(m.ffmpeg), tickCmd())
}

func listDevicesCmd(ffmpeg string) tea.Cmd {
	return func() tea.Msg {
		if runtime.GOOS != "windows" {
			return deviceListMsg{err: fmt.Errorf("this build currently uses FFmpeg's DirectShow backend and is intended for Windows")}
		}
		cmd := exec.Command(ffmpeg, "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return deviceListMsg{err: err}
		}
		if err := cmd.Start(); err != nil {
			return deviceListMsg{err: fmt.Errorf("could not start FFmpeg: %w", err)}
		}
		var devices []string
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := devicePattern.FindStringSubmatch(line); len(match) == 2 {
				name := strings.TrimSpace(match[1])
				found := false
				for _, existing := range devices {
					if existing == name {
						found = true
						break
					}
				}
				if !found {
					devices = append(devices, name)
				}
			}
		}
		_ = cmd.Wait() // FFmpeg exits non-zero here by design because the input is "dummy".
		if err := scanner.Err(); err != nil {
			return deviceListMsg{err: err}
		}
		if len(devices) == 0 {
			return deviceListMsg{err: fmt.Errorf("no audio devices found; check that FFmpeg supports DirectShow and your microphone is available")}
		}
		return deviceListMsg{devices: devices}
	}
}

func startCaptureCmd(ffmpeg, device string) tea.Cmd {
	return func() tea.Msg {
		args := []string{
			"-hide_banner", "-nostats", "-loglevel", "info",
			"-f", "dshow", "-i", "audio=" + device,
			"-vn", "-af", "astats=metadata=1:reset=1,ametadata=print:key=lavfi.astats.Overall.RMS_level",
			"-f", "null", "-",
		}
		cmd := exec.Command(ffmpeg, args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return captureStartedMsg{err: err}
		}
		if err := cmd.Start(); err != nil {
			return captureStartedMsg{err: fmt.Errorf("could not start capture: %w", err)}
		}
		levels := make(chan float64, 16)
		go func() {
			defer close(levels)
			scanner := bufio.NewScanner(stderr)
			scanner.Buffer(make([]byte, 4096), 1024*1024)
			for scanner.Scan() {
				matches := rmsPattern.FindStringSubmatch(scanner.Text())
				if len(matches) != 2 {
					continue
				}
				value, parseErr := strconv.ParseFloat(matches[1], 64)
				if parseErr != nil || math.IsNaN(value) {
					value = -120
				}
				select {
				case levels <- value:
				default:
				}
			}
			_ = cmd.Wait()
		}()
		return captureStartedMsg{cmd: cmd, levels: levels}
	}
}

func readLevelCmd(levels chan float64) tea.Cmd {
	return func() tea.Msg {
		level, ok := <-levels
		if !ok {
			return captureEndedMsg{}
		}
		return levelMsg(level)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.stopCapture()
			return m, tea.Quit
		case "up", "k":
			if !m.capturing && m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if !m.capturing && m.selected < len(m.devices)-1 {
				m.selected++
			}
		case "enter", " ":
			if m.capturing {
				m.stopCapture()
				m.status = "Capture stopped"
			} else if len(m.devices) > 0 {
				m.status = "Starting microphone…"
				m.err = ""
				return m, startCaptureCmd(m.ffmpeg, m.devices[m.selected])
			}
		case "esc", "backspace":
			if m.capturing {
				m.stopCapture()
				m.status = "Capture stopped"
			}
		case "r":
			if !m.capturing {
				m.status = "Refreshing devices…"
				return m, listDevicesCmd(m.ffmpeg)
			}
		}
	case deviceListMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "Device scan failed"
		} else {
			m.devices = msg.devices
			if m.selected >= len(m.devices) {
				m.selected = 0
			}
			m.status = fmt.Sprintf("Found %d audio device(s)", len(m.devices))
			m.err = ""
		}
	case captureStartedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = "Capture failed"
			return m, nil
		}
		m.cmd, m.levels = msg.cmd, msg.levels
		m.capturing = true
		m.level, m.peak = -120, -120
		m.status = "Listening — speak into your microphone"
		return m, readLevelCmd(m.levels)
	case levelMsg:
		m.level = float64(msg)
		if m.level > m.peak {
			m.peak = m.level
		}
		return m, readLevelCmd(m.levels)
	case captureEndedMsg:
		m.capturing = false
		m.cmd, m.levels = nil, nil
		if m.status != "Capture stopped" {
			m.status = "FFmpeg capture ended"
		}
	case tickMsg:
		return m, tickCmd()
	}
	return m, nil
}

func (m *model) stopCapture() {
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
	m.capturing = false
	m.cmd, m.levels = nil, nil
}

func (m model) View() string {
	var b strings.Builder
	width := m.width
	if width < 60 {
		width = 60
	}
	b.WriteString(titleStyle.Render("🎙  "+appName) + "  " + helpStyle.Render("LIVE MICROPHONE LEVEL METER") + "\n")
	b.WriteString(helpStyle.Render("Measure input loudness in dBFS · powered by FFmpeg") + "\n\n")

	var content strings.Builder
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("AUDIO INPUT") + "\n")
	if len(m.devices) == 0 {
		content.WriteString(labelStyle.Render("No devices loaded yet.") + "\n")
	} else {
		for i, device := range m.devices {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.selected {
				prefix = "› "
				style = selectedStyle
			}
			content.WriteString(style.Render(prefix+device) + "\n")
		}
	}
	b.WriteString(panelStyle.Width(width-8).Render(content.String()) + "\n\n")

	levelLabel := "IDLE"
	if m.capturing {
		levelLabel = "● LIVE"
	}
	meterWidth := width - 20
	if meterWidth < 20 {
		meterWidth = 20
	}
	if meterWidth > 60 {
		meterWidth = 60
	}
	filled := 0
	if m.level > -60 {
		filled = int((m.level + 60) / 60 * float64(meterWidth))
	}
	if filled < 0 {
		filled = 0
	}
	if filled > meterWidth {
		filled = meterWidth
	}
	meter := strings.Repeat("█", filled) + strings.Repeat("░", meterWidth-filled)
	meterColor := good
	if m.level > -12 {
		meterColor = warn
	}
	if m.level > -3 {
		meterColor = bad
	}
	meterStyle := lipgloss.NewStyle().Foreground(meterColor).Bold(true)
	levelText := "— dBFS"
	if m.capturing {
		levelText = fmt.Sprintf("%6.1f dBFS", m.level)
	}
	peakText := "— dBFS"
	if m.capturing || m.peak > -120 {
		peakText = fmt.Sprintf("%6.1f dBFS", m.peak)
	}
	meterPanel := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Bold(true).Render("INPUT LEVEL"), strings.Repeat(" ", 2), lipgloss.NewStyle().Foreground(meterColor).Render(levelLabel)),
		meterStyle.Render(meter),
		lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render("Current  "), lipgloss.NewStyle().Bold(true).Render(levelText), strings.Repeat(" ", 4), labelStyle.Render("Peak  "), lipgloss.NewStyle().Bold(true).Render(peakText)),
	)
	b.WriteString(panelStyle.Width(width-8).Render(meterPanel) + "\n\n")

	statusStyle := lipgloss.NewStyle().Foreground(good)
	if m.err != "" {
		statusStyle = lipgloss.NewStyle().Foreground(bad)
	}
	b.WriteString(statusStyle.Render("● "+m.status) + "\n")
	if m.err != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(bad).Width(width-4).Render(m.err) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ select  ·  Enter start/stop  ·  R refresh  ·  Esc stop  ·  Q quit"))
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func main() {
	ffmpeg := "ffmpeg"
	if len(strings.TrimSpace(ffmpeg)) == 0 {
		ffmpeg = "ffmpeg.exe"
	}
	p := tea.NewProgram(initialModel(ffmpeg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("MicMeter exited with an error: %v\n", err)
	}
}
