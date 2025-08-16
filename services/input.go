package services

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var scanner *bufio.Scanner

func init() {
	scanner = bufio.NewScanner(os.Stdin)
}

func GetInput() string {
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(scanner.Text())
}

func GetInputWithPrompt(prompt string) string {
	fmt.Print(prompt)
	return GetInput()
}

func ConfirmAction(message string) bool {
	fmt.Printf("%s (y/n): ", message)
	response := strings.ToLower(GetInput())
	return response == "y" || response == "yes"
}
