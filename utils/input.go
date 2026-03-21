package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func PromptInput(prompt, placeholder string) string {
	if GlobalForAIFlag {
		return readPipedLine()
	}
	if placeholder != "" {
		fmt.Printf("%s [%s]: ", prompt, placeholder)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return ""
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return placeholder
	}
	return input
}

func readPipedLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}
