// main.go
package main

import (
	"os"

	"github.com/aveni/pb-cli/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
