package main

import (
	"database/sql"
	"os"

	"github.com/SoulOppen/aggregator/internal/config"
	"github.com/SoulOppen/aggregator/internal/database"
	"github.com/SoulOppen/aggregator/internal/state"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		os.Exit(1)
	}
	var states state.State
	states.Config = &cfg
	db, err := sql.Open("postgres", cfg.Config["db_url"])
	if err != nil {
		os.Exit(1)
	}
	dbQueries := database.New(db)
	states.Db = dbQueries
	var commands state.Commands
	commands.Register("login", state.HandlerLogin)
	commands.Register("register", state.HandlerRegister)
	commands.Register("reset", state.HandlerReset)
	commands.Register("users", state.HandlerGetUser)
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
