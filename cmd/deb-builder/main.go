package main

import (
	"deb-builder/internal/commands"

	"github.com/deweppro/go-app/console"
)

func main() {
	root := console.New("deb-builder", "help deb-builder")
	root.AddCommand(commands.CreateConfig())
	root.AddCommand(commands.Build())
	root.Exec()
}
