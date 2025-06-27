package state

import (
	"errors"
	"fmt"

	"github.com/SoulOppen/aggregator/internal/config"
)

type State struct {
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
		s.Config.SetUser(cmd.Args[0])
		fmt.Printf("User set to %s\n", cmd.Args[0])
	} else {
		return errors.New("the login handler expects a single argument, the username")
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
