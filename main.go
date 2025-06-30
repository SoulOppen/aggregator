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
	defer db.Close()
	dbQueries := database.New(db)
	states.Db = dbQueries
	var commands state.Commands
	commands.Register("login", state.HandlerLogin)
	commands.Register("register", state.HandlerRegister)
	commands.Register("reset", state.HandlerReset)
	commands.Register("users", state.HandlerGetUser)
	commands.Register("agg", state.HandlerAgg)
	commands.Register("addfeed", state.MiddlewareLoggedIn(state.AddFeed))
	commands.Register("feeds", state.HandlerFeed)
	commands.Register("follow", state.MiddlewareLoggedIn(state.HandlerFollow))
	commands.Register("following", state.MiddlewareLoggedIn(state.HandlerListFeedFollows))
	commands.Register("unfollow", state.MiddlewareLoggedIn(state.HandlerUnfollow))
	args := os.Args

	var command state.Command
	command.Name = args[1]
	command.Args = args[2:]
	err = commands.Run(&states, command)
	if err != nil {
		os.Exit(1)
	}

}
