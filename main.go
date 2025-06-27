package main

import (
	"os"

	"github.com/SoulOppen/aggregator/internal/config"
	"github.com/SoulOppen/aggregator/internal/state"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		os.Exit(1)
	}
	var states state.State
	states.Config = &cfg
	var commands state.Commands
	commands.Register("login", state.HandlerLogin)
	args := os.Args
	if len(args) < 2 {
		os.Exit(1)
	}
	var command state.Command
	command.Name = args[1]
	command.Args = args[2:]
	err = commands.Run(&states, command)
	if err != nil {
		os.Exit(1)
	}
}
