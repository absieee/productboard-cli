// main.go
package main

import (
	"os"

	"github.com/aveni/pb-cli/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	cmd.AddFeaturesCmd(root, cmd.BaseURL, cmd.ResolvedToken)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
