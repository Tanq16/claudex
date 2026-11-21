package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var stdin = bufio.NewReader(os.Stdin)

func PromptInput(prompt string) (string, error) {
	fmt.Print(prompt + " ")
	line, err := stdin.ReadString('\n')
	if line == "" && err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
