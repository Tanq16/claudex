package utils

import (
	"errors"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var ErrNoTerminal = errors.New("no interactive terminal")

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
	if !StdinIsTerminal {
		return "", ErrNoTerminal
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
	if !StdinIsTerminal {
		return "", ErrNoTerminal
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
	if !StdinIsTerminal {
		return "", ErrNoTerminal
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
	chosen    map[int]bool
	multi     bool
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
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.options)-1)
		case " ":
			if m.multi {
				m.chosen[m.cursor] = !m.chosen[m.cursor]
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
		style := selectOption
		row := opt
		if m.multi {
			check := "[ ]"
			if m.chosen[i] {
				check = "[●]"
				style = selectCursor
			}
			row = check + " " + opt
		}
		marker := "    "
		if i == m.cursor {
			marker = "  › "
			if !m.multi {
				style = selectCursor
			}
		}
		b.WriteString(style.Render(marker + row))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	hint := "  enter select · esc cancel"
	if m.multi {
		hint = "  space toggle · enter confirm · esc cancel"
	}
	b.WriteString(selectHint.Render(hint))
	b.WriteString("\n\n")
	return tea.NewView(b.String())
}

func PromptSelect(label string, options []string) (int, error) {
	if !StdinIsTerminal {
		return -1, ErrNoTerminal
	}

	finalModel, err := tea.NewProgram(selectModel{label: label, options: options, selected: -1}).Run()
	if err != nil {
		return -1, err
	}

	result := finalModel.(selectModel)
	if result.cancelled {
		return -1, nil
	}
	return result.selected, nil
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
	if !StdinIsTerminal {
		return nil, ErrNoTerminal
	}

	finalModel, err := tea.NewProgram(selectModel{
		label:   label,
		options: options,
		chosen:  make(map[int]bool),
		multi:   true,
	}).Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(selectModel)
	if result.cancelled {
		return nil, nil
	}
	return result.chosen, nil
}
