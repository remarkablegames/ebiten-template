package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/game"
	"github.com/remarkablegames/ebiten-template/runner"
)

func main() {
	ebiten.SetWindowSize(game.WindowWidth, game.WindowHeight)
	ebiten.SetWindowTitle("ebiten-template")
	if err := ebiten.RunGame(runner.NewGame()); err != nil {
		log.Fatal(err)
	}
}
