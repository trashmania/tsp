package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time
type statusMsg string
type boostMsg struct {
	newState bool
	err      error
}
type model struct {
	activeRow        int
	profiles         []string
	currentProfile   int
	systemProfile    int
	profilesExpanded bool
	cpuBoost         bool
	status           string
	watts            float64
	temp             float64
	ghz              float64
	cpuPercent       float64
	prevIdle         uint64
	prevTotal        uint64
}

const (
	pathIntelNoTurbo = "/sys/devices/system/cpu/intel_pstate/no_turbo"
	pathCpufreqBoost = "/sys/devices/system/cpu/cpufreq/boost"
)

type boostDriver int

const (
	boostUnknown boostDriver = iota
	boostIntelPstate
	boostGeneric
)

func detectBoostDriver() boostDriver {
	if _, err := os.Stat(pathIntelNoTurbo); err == nil {
		return boostIntelPstate
	}
	if _, err := os.Stat(pathCpufreqBoost); err == nil {
		return boostGeneric
	}
	return boostUnknown
}

func getCPUBoost() (bool, error) {
	switch detectBoostDriver() {
	case boostIntelPstate:
		data, err := os.ReadFile(pathIntelNoTurbo)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(data)) == "0", nil
	case boostGeneric:
		data, err := os.ReadFile(pathCpufreqBoost)
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(string(data)) == "1", nil
	default:
		return false, fmt.Errorf("не удалось определить драйвер cpufreq")
	}
}

func detectCurrentProfile() int {
	if data, err := os.ReadFile("/sys/firmware/acpi/platform_profile"); err == nil {
		prof := strings.TrimSpace(string(data))
		switch prof {
		case "low-power":
			return 0
		case "balanced":
			return 1
		case "performance":
			return 2
		}
	}

	cmd := exec.Command("tlp-stat", "-m")
	if out, err := cmd.Output(); err == nil {
		outStr := strings.ToLower(string(out))
		if strings.Contains(outStr, "power-saver") || strings.Contains(outStr, "powersave") {
			return 0
		} else if strings.Contains(outStr, "performance") {
			return 2
		} else if strings.Contains(outStr, "balanced") || strings.Contains(outStr, "balance") {
			return 1
		} else if strings.Contains(outStr, "mode = ac") || strings.Contains(outStr, "manual = ac") {
			return 3
		}
	}
	return 1
}

func writeWithPrivilege(path, value string) error {
	cmd := exec.Command("doas", "tee", path)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func setCPUBoost(enable bool) error {
	var path, val string
	switch detectBoostDriver() {
	case boostIntelPstate:
		path = pathIntelNoTurbo
		val = "1"
		if enable {
			val = "0"
		}
	case boostGeneric:
		path = pathCpufreqBoost
		val = "0"
		if enable {
			val = "1"
		}
	default:
		return fmt.Errorf("турбо буст не поддерживается процессором")
	}
	return writeWithPrivilege(path, val)
}

func toggleBoost(newState bool) tea.Cmd {
	return func() tea.Msg {
		err := setCPUBoost(newState)
		return boostMsg{newState: newState, err: err}
	}
}

func applyTLPProfile(profile string) tea.Cmd {
	return func() tea.Msg {
		var tlpArg string
		switch profile {
		case "powersave":
			tlpArg = "power-saver"
		case "balance":
			tlpArg = "balanced"
		case "performance":
			tlpArg = "performance"
		case "ac":
			tlpArg = "ac"
		default:
			errMsg := fmt.Sprintf("Неизвестный профиль: %s", profile)
			return statusMsg(errMsg)
		}

		cmd := exec.Command("doas", "tlp", tlpArg)
		if err := cmd.Run(); err != nil {
			errMsg := fmt.Sprintf("Ошибка TLP (%s): %v", profile, err)
			return statusMsg(errMsg)
		}
		okMsg := fmt.Sprintf("Применён профиль: %s", profile)
		return statusMsg(okMsg)
	}
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func runPowertop() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("doas", "powertop", "--auto-tune")
		err := cmd.Run()
		if err != nil {
			errMsg := fmt.Sprintf("Ошибка Powertop: %v", err)
			return statusMsg(errMsg)
		}
		return statusMsg("Энергопотребление оптимизировано")
	}
}

func (m model) Init() tea.Cmd {
	return doTick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.updateMetrics()
		return m, doTick()
	case statusMsg:
		m.status = string(msg)
		return m, nil
	case boostMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Ошибка Турбо буста: %v", msg.err)
		} else {
			m.cpuBoost = msg.newState
			state := "Выкл"
			if msg.newState {
				state = "Вкл"
			}
			m.status = "Турбо буст: " + state
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "й", "Й", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.activeRow > 0 {
				m.activeRow--
			}
		case "down", "j":
			if m.activeRow < 3 {
				m.activeRow++
			}
		case "left", "h":
			if m.activeRow == 0 && m.profilesExpanded && m.currentProfile > 0 {
				m.currentProfile--
			}
		case "right", "l":
			if m.activeRow == 0 && m.profilesExpanded && m.currentProfile < len(m.profiles)-1 {
				m.currentProfile++
			}
		case "enter", " ":
			switch m.activeRow {
			case 0:
				m.profilesExpanded = !m.profilesExpanded
				if !m.profilesExpanded {
					m.systemProfile = m.currentProfile
					selectedProfile := m.profiles[m.currentProfile]
					m.status = fmt.Sprintf("Применение %s...", selectedProfile)
					return m, applyTLPProfile(selectedProfile)
				} else {
					m.status = "Выберите профиль"
				}
			case 1:
				m.status = "Переключение Турбо буста..."
				return m, toggleBoost(!m.cpuBoost)
			case 2:
				m.status = "Оптимизация..."
				return m, runPowertop()
			case 3:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var s string
	s += "╭──[ TSP • ThinkPad Sigmo's Power ]──╮\n"
	s += fmt.Sprintf("│  Потребление:  %-6.2fW             │\n", m.watts)
	s += fmt.Sprintf("│  Температура:  %-6.1f°C            │\n", m.temp)
	s += fmt.Sprintf("│  Частота ЦП:   %-6.2fGHz           │\n", m.ghz)
	s += fmt.Sprintf("│  Загрузка ЦП:  %-6.1f%%             │\n", m.cpuPercent)
	s += "├────────────────────────────────────┤\n\n"

	sysProfileStr := m.profiles[m.systemProfile]

	if m.activeRow == 0 {
		if m.profilesExpanded {
			s += "   Профили питания (показать/скрыть) <---\n"
		} else {
			s += fmt.Sprintf("   Профили питания [%s] <---\n", sysProfileStr)
		}
	} else {
		s += fmt.Sprintf(" Профили питания [%s]\n", sysProfileStr)
	}

	if m.profilesExpanded {
		s += "   "
		for i, prof := range m.profiles {
			marker := " "
			if i == m.systemProfile {
				marker = "•"
			}

			if i == m.currentProfile {
				s += fmt.Sprintf("[%s%s] ", marker, prof)
			} else {
				s += fmt.Sprintf(" %s%s  ", marker, prof)
			}
		}
		s += "\n"
	}

	boostState := "Выкл"
	if m.cpuBoost {
		boostState = "Вкл"
	}

	if m.activeRow == 1 {
		s += fmt.Sprintf("   Турбо буст: %s <---\n", boostState)
	} else {
		s += fmt.Sprintf(" Турбо буст: %s\n", boostState)
	}

	if m.activeRow == 2 {
		s += "   Автонастройка Powertop <---\n"
	} else {
		s += " Автонастройка Powertop\n"
	}
	s += "\n├────────────────────────────────────┤\n"
	if m.activeRow == 3 {
		s += "   Выход <---\n"
	} else {
		s += " Выход\n"
	}
	s += "╰────────────────────────────────────╯\n"
	if m.status != "" {
		s += fmt.Sprintf("\n Действие: %s\n", m.status)
	} else {
		s += "\n Управление: Выбор — Enter, Навигация — Стрелки\n"
	}
	return s
}

func (m *model) updateMetrics() {
	if data, err := os.ReadFile("/sys/class/power_supply/BAT0/power_now"); err == nil {
		val, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		m.watts = val / 1000000.0
	} else {
		m.watts = 0.0
	}
	if data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		val, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		m.temp = val / 1000.0
	}
	if data, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq"); err == nil {
		val, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		m.ghz = val / 1000000.0
	}
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 5 {
				var total uint64
				for i := 1; i < len(fields); i++ {
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					total += val
				}
				idle, _ := strconv.ParseUint(fields[4], 10, 64)
				if m.prevTotal > 0 {
					diffIdle := idle - m.prevIdle
					diffTotal := total - m.prevTotal
					if diffTotal > 0 {
						m.cpuPercent = 100.0 * (1.0 - float64(diffIdle)/float64(diffTotal))
					}
				}
				m.prevIdle = idle
				m.prevTotal = total
			}
		}
	}
}

func main() {
	sysProf := detectCurrentProfile()
	initalModel := model{
		activeRow:        0,
		profiles:         []string{"powersave", "balance", "performance", "ac"},
		currentProfile:   sysProf,
		systemProfile:    sysProf,
		profilesExpanded: false,
		cpuBoost:         false,
	}
	initalModel.updateMetrics()

	if boost, err := getCPUBoost(); err == nil {
		initalModel.cpuBoost = boost
	}

	p := tea.NewProgram(initalModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка запуска: %v\n", err)
		os.Exit(1)
	}
}
