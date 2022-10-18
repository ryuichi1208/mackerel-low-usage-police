package main

import (
	"fmt"
	"os"

	"github.com/ryuichi1208/mackerel-low-usage-police/lib"
)

func init() {
	if os.Getenv("MACKEREL_TOKEN") == "" {
		fmt.Println("Set environment variable MACKEREL_TOKEN")
		os.Exit(1)
	}
}

func main() {
	os.Exit(lib.Do())
}
