package main

import (
	"log"

	"hordefall/internal/game"
)

func main() {
	if err := game.Run(); err != nil {
		log.Fatal(err)
	}
}
