package main

import (
	"fmt"
	"log"

	"github.com/SoulOppen/aggregator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("No se puede leer")
	}
	cfg.SetUser("Ariel")
	cfg, err = config.Read()
	if err != nil {
		log.Fatal("No se puede leer")
	}
	for k, v := range cfg.Config {
		fmt.Printf("%s - %s\n", k, v)
	}
}
