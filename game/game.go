// Package game holds the hot-reloadable game logic for development on desktop.
package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// p is the character. Delete this file, player.go, and sprite.go when
// starting your own game.
var p = newPlayer()

// Update advances the game state by one tick.
func Update() error {
	// A broken sheet stops the game with the error instead of animating
	// the wrong frames.
	if sheetMeta.err != nil {
		return fmt.Errorf("sprite sheet: %w", sheetMeta.err)
	}

	p.update(input{
		left:  ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA),
		right: ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD),
		jump: ebiten.IsKeyPressed(ebiten.KeySpace) ||
			ebiten.IsKeyPressed(ebiten.KeyW) ||
			ebiten.IsKeyPressed(ebiten.KeyUp),
	})
	return nil
}

// Draw renders the game to the screen.
func Draw(screen *ebiten.Image) {
	// Ground.
	vector.FillRect(screen, 0, groundY, ScreenWidth, groundHeight, color.RGBA{R: 0x8b, G: 0xb7, B: 0x52, A: 0xff}, false)

	if sheetMeta.err != nil {
		ebitenutil.DebugPrint(screen, "sprite sheet: "+sheetMeta.err.Error())
		return
	}

	if err := p.draw(screen); err != nil {
		ebitenutil.DebugPrint(screen, "sprite sheet: "+err.Error())
		return
	}

	ebitenutil.DebugPrint(screen, "arrows/WASD: move, space/W/up: jump")
}

// Layout returns the fixed screen size in pixels.
func Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return ScreenWidth, ScreenHeight
}
