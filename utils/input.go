package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var stdinScanner *bufio.Scanner

func getStdinScanner() *bufio.Scanner {
	if stdinScanner == nil {
		stdinScanner = bufio.NewScanner(os.Stdin)
	}
	return stdinScanner
}

func ReadPipedInput() string {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return ""
	}
	scanner := getStdinScanner()
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func ReadPipedLine() string {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice != 0 {
		return ""
	}
	scanner := getStdinScanner()
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

type inputModel struct {
	textInput textinput.Model
	done      bool
	value     string
	initCmd   tea.Cmd
}

func (m inputModel) Init() tea.Cmd {
	return m.initCmd
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textInput.View())
}

func PromptInput(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = prompt + " "
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return strings.TrimSpace(result.value), nil
}

func PromptPassword(prompt string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = "••••••••"
	ti.Prompt = prompt + " "
	ti.EchoMode = textinput.EchoPassword
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return result.value, nil
}

type textAreaModel struct {
	textarea textarea.Model
	done     bool
	value    string
	initCmd  tea.Cmd
}

func (m textAreaModel) Init() tea.Cmd {
	return m.initCmd
}

func (m textAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+d":
			m.value = m.textarea.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textAreaModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textarea.View() + "\n(Ctrl+D to submit, Esc to cancel)")
}

func PromptTextArea(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedInput(), nil
	}

	PrintInfo(prompt)

	ta := textarea.New()
	ta.Placeholder = placeholder
	focusCmd := ta.Focus()

	m := textAreaModel{textarea: ta, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(textAreaModel)
	return strings.TrimSpace(result.value), nil
}

var (
	selectLabel  = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12))
	selectCursor = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10))
	selectOption = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(15))
	selectHint   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
)

type selectModel struct {
	label     string
	options   []string
	cursor    int
	selected  int
	cancelled bool
	done      bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(selectLabel.Render("  " + m.label))
	b.WriteString("\n")
	for i, opt := range m.options {
		if i == m.cursor {
			b.WriteString(selectCursor.Render("  › " + opt))
		} else {
			b.WriteString(selectOption.Render("    " + opt))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(selectHint.Render("  enter select · esc cancel"))
	b.WriteString("\n\n")
	return tea.NewView(b.String())
}

func PromptSelect(label string, options []string) (int, error) {
	if GlobalForAIFlag {
		line := ReadPipedLine()
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			return -1, fmt.Errorf("invalid selection: %s", line)
		}
		return n - 1, nil
	}

	m := selectModel{label: label, options: options, selected: -1}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}

	result := finalModel.(selectModel)
	if result.cancelled {
		return -1, nil
	}
	return result.selected, nil
}

type multiSelectModel struct {
	label     string
	options   []string
	cursor    int
	selected  map[int]bool
	cancelled bool
	done      bool
}

func (m multiSelectModel) Init() tea.Cmd { return nil }

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m multiSelectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(selectLabel.Render("  " + m.label))
	b.WriteString("\n")
	for i, opt := range m.options {
		check := "[ ]"
		style := selectOption
		if m.selected[i] {
			check = "[●]"
			style = selectCursor
		}
		if i == m.cursor {
			b.WriteString(style.Render(fmt.Sprintf("  › %s %s", check, opt)))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("    %s %s", check, opt)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(selectHint.Render("  space toggle · enter confirm · esc cancel"))
	b.WriteString("\n\n")
	return tea.NewView(b.String())
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
	if GlobalForAIFlag {
		line := ReadPipedLine()
		if line == "" || strings.EqualFold(line, "none") {
			return make(map[int]bool), nil
		}
		selected := make(map[int]bool)
		for _, part := range strings.Split(line, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 1 || n > len(options) {
				continue
			}
			selected[n-1] = true
		}
		return selected, nil
	}

	m := multiSelectModel{
		label:    label,
		options:  options,
		selected: make(map[int]bool),
	}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(multiSelectModel)
	if result.cancelled {
		return nil, nil
	}
	return result.selected, nil
}
