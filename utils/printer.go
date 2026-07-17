package utils

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rs/zerolog/log"
)

var (
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(12))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(11))
)

func emit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func PrintInfo(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		emit("[INFO] " + msg)
	} else {
		emit(infoStyle.Render("→ " + msg))
	}
}

func PrintSuccess(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		emit("[OK] " + msg)
	} else {
		emit(successStyle.Render("✓ " + msg))
	}
}

func PrintError(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
	} else if GlobalForAIFlag {
		emit("[ERROR] " + msg)
	} else {
		emit(errorStyle.Render("✗ " + msg))
	}
}

func PrintFatal(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
	} else if GlobalForAIFlag {
		emit("[ERROR] " + msg)
	} else {
		emit(errorStyle.Render("✗ " + msg))
	}
	os.Exit(1)
}

func PrintWarn(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
	} else if GlobalForAIFlag {
		emit("[WARN] " + msg)
	} else {
		emit(warnStyle.Render("! " + msg))
	}
}

func PrintRunning(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		emit("[RUNNING] " + msg)
	} else {
		emit(infoStyle.Render("↻ " + msg))
	}
}

func PrintIndentedSuccess(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		emit("[OK] " + msg)
	} else {
		emit(successStyle.Render("  ✓ " + msg))
	}
}

func PrintIndentedError(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Error().Err(err).Msg(msg)
		} else {
			log.Error().Msg(msg)
		}
	} else if GlobalForAIFlag {
		emit("[ERROR] " + msg)
	} else {
		emit(errorStyle.Render("  ✗ " + msg))
	}
}

func PrintIndentedWarn(msg string, err error) {
	if GlobalDebugFlag {
		if err != nil {
			log.Warn().Err(err).Msg(msg)
		} else {
			log.Warn().Msg(msg)
		}
	} else if GlobalForAIFlag {
		emit("[WARN] " + msg)
	} else {
		emit(warnStyle.Render("  ! " + msg))
	}
}

func PrintIndentedRunning(msg string) {
	if GlobalDebugFlag {
		log.Info().Msg(msg)
	} else if GlobalForAIFlag {
		emit("[RUNNING] " + msg)
	} else {
		emit(infoStyle.Render("  ↻ " + msg))
	}
}

func PrintProgress(label string, percent int) {
	if percent > 100 {
		percent = 100
	}

	if GlobalDebugFlag {
		log.Info().Int("percent", percent).Msg(label)
		return
	}

	if GlobalForAIFlag {
		emit(fmt.Sprintf("[PROGRESS] %s: %d%%", label, percent))
		return
	}

	const barWidth = 10
	filled := barWidth * percent / 100
	empty := barWidth - filled

	bar := strings.Repeat("⣿", filled) + strings.Repeat("⣀", empty)
	emit(infoStyle.Render(fmt.Sprintf("  ↻ %s: %s %d%%", label, bar, percent)))
}

func ClearLines(n int) {
	if GlobalDebugFlag || GlobalForAIFlag {
		return
	}
	for i := 0; i < n; i++ {
		fmt.Fprint(os.Stderr, "\033[A\033[2K")
	}
}

func ClearPreviousLine() {
	ClearLines(1)
}

func PrintGeneric(msg string) {
	fmt.Println(msg)
}
