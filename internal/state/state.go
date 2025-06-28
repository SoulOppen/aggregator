package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/SoulOppen/aggregator/internal/config"
	"github.com/SoulOppen/aggregator/internal/database"
	"github.com/google/uuid"
)

type State struct {
	Db     *database.Queries
	Config *config.Config
}
type Command struct {
	Name string
	Args []string
}
type Commands struct {
	Callbacks map[string]func(*State, Command) error
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return errors.New("the login handler expects a single argument, the username")
	} else if len(cmd.Args) == 1 {
		_, err := s.Db.GetUser(context.Background(), sql.NullString{String: cmd.Args[0], Valid: true})
		if err != nil {
			return err
		}
		s.Config.SetUser(cmd.Args[0])
		fmt.Printf("User set to %s\n", cmd.Args[0])
	} else {
		return errors.New("the login handler expects a single argument, the username")
	}
	return nil
}
func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) == 0 || len(cmd.Args) > 2 {
		return errors.New("the register handler expects a single argument, the username")
	}

	_, err := s.Db.GetUser(context.Background(), sql.NullString{String: cmd.Args[0], Valid: true})
	if err == nil {
		return errors.New("registro ya existe")
	}
	var newUser database.CreateUserParams
	newUser.ID = uuid.New()
	newUser.Name = sql.NullString{String: cmd.Args[0], Valid: true}
	newUser.CreatedAt = time.Now()
	newUser.UpdatedAt = time.Now()
	createdUser, err := s.Db.CreateUser(context.Background(), newUser)
	if err != nil {
		return errors.New("Failed")
	}
	fmt.Printf("user created %s\n", createdUser.Name.String)
	s.Config.SetUser(createdUser.Name.String)
	return nil

}
func HandlerReset(s *State, cmd Command) error {
	err := s.Db.DeleteAllUsers(context.Background())
	if err != nil {
		return errors.New("no se pudo borrar")
	}
	return nil

}
func HandlerGetUser(s *State, cmd Command) error {
	current, ok := s.Config.Config["current_user_name"]
	if !ok {
		return errors.New("no hay usuario")
	}
	names, err := s.Db.GetAllUsers(context.Background())
	if err != nil {
		return errors.New("no se pudo acceder a usuarios")
	}
	for _, name := range names {
		if name.String == current {
			fmt.Printf("* %s (current)\n", name.String)
		} else {
			fmt.Printf("* %s\n", name.String)
		}
	}
	return nil

}
func (c *Commands) Run(s *State, cmd Command) error {
	cb, ok := c.Callbacks[cmd.Name]
	if !ok {
		return errors.New("not function")
	}
	err := cb(s, cmd)
	if err != nil {
		return err
	}
	return nil
}
func (c *Commands) Register(name string, f func(*State, Command) error) {
	if c.Callbacks == nil {
		c.Callbacks = make(map[string]func(*State, Command) error)
	}
	if _, exists := c.Callbacks[name]; !exists {
		c.Callbacks[name] = f
	}
}
