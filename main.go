// main.go
package main

import (
	"fmt"
	"os"

	"github.com/aveni/pb-cli/cmd"
)

func main() {
	token, err := cmd.LoadToken()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	root := cmd.NewRootCmd()
	_ = token // commands will be wired in later tasks
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
