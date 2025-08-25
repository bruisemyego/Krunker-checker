// main.go

package main

import (
	"fmt"
	"krunker-checker/src"
	"os"
	"os/signal"
	"syscall"
)

const (
	ColorCyan   = "\033[36m"
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorPurple = "\033[35m"
	ColorBold   = "\033[1m"
)

func main() {
	fmt.Print("\033[H\033[2J")
	printHeader()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan error, 1)
	go func() {
		done <- src.ProcessAccounts()
	}()

	select {
	case err := <-done:
		if err != nil {
			fmt.Printf(ColorRed+"error processing accounts: %v\n"+ColorReset, err)
			os.Exit(1)
		}

		fmt.Println(ColorGreen + "\nchecking completed!" + ColorReset)

	case <-sigChan:
		fmt.Println(ColorCyan + "\n\nShutting down..." + ColorReset)
		os.Exit(0)
	}
}

func printHeader() {
	fmt.Printf(ColorBold + ColorCyan + "               Krunker Account Checker - @cleanest\n" + ColorReset)
	fmt.Printf(ColorPurple + " 			   Version: 3.00\n\n" + ColorReset)
}
