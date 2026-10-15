//go:build js || release

// Package runner supplies the game to ebiten.RunGame.
//
// This is the variant used when hot reload is off: web builds (no file
// watching or interpreter in the browser) and release builds
// (go build -tags release), which run the game logic compiled into the
// binary instead of interpreting it. Development builds use development.go.
// This file's build tag is the complement of development.go's, so exactly one
// of the two files is built.
package runner

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/game"
)

// NewGame returns the compiled game.
func NewGame() ebiten.Game {
	return compiledGame{}
}

type compiledGame struct{}

func (compiledGame) Update() error {
	return game.Update()
}

func (compiledGame) Draw(screen *ebiten.Image) {
	game.Draw(screen)
}

func (compiledGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return game.Layout(outsideWidth, outsideHeight)
}
