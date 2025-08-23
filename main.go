package main

import (
	"fmt"
	"krunker-checker/services"
)

const (
	ColorCyan  = "\033[36m"
	ColorReset = "\033[0m"
)

func main() {
	if err := services.ProcessAccounts(); err != nil {
		fmt.Printf("Error processing accounts: %v\n", err)
		return
	}

	fmt.Println(ColorCyan + "\nchecking completed!" + ColorReset)
	fmt.Println(ColorCyan + "Check the results." + ColorReset)
}
