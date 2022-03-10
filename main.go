package main

import (
	"github.com/Diez37/go-skeleton/interface/cli"
	"github.com/diez37/go-packages/closer"
	_ "github.com/joho/godotenv/autoload"
	"os"
)

func main() {
	cmd, err := cli.NewRootCommand()
	if err != nil {
		os.Exit(closer.ExitCodeError)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(closer.ExitCodeError)
	}
}
