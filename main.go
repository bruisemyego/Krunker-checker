// main.go

package main

import (
	"fmt"
	"krunker-cli/services"
)

func main() {
	fmt.Println("Starting account checking...")

	if err := services.ProcessAccounts(); err != nil {
		fmt.Printf("Error processing accounts: %v\n", err)
		return
	}

	fmt.Println("\nchecking completed!")
	fmt.Println("Check the results.")
}
