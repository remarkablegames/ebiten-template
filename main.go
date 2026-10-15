package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/runner"
)

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Hello, World!")
	if err := ebiten.RunGame(runner.NewGame()); err != nil {
		log.Fatal(err)
	}
}
