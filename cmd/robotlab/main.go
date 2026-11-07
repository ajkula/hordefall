package main

import (
	"log"

	"hordefall/internal/game"
)

func main() {
	if err := game.RunRobotLab(); err != nil {
		log.Fatal(err)
	}
}
