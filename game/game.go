// Package game holds the hot-reloadable game logic for development on desktop.
package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Update advances the game state by one tick.
func Update() error {
	return nil
}

// Draw renders the game to the screen.
func Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Hello, World!")
}

// Layout returns the fixed screen size in pixels.
func Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}
