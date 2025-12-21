package main

import (
	"fmt"
	"os"

	"github.com/bmf/yagwt/internal/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
