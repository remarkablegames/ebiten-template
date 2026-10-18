// Package assets embeds the game's art, sound, and metadata with go:embed.
// The hot-reload script reads them here (exposed via symbols/).
package assets

import _ "embed"

//go:embed sprites/sheet.png
var sheetPNG []byte

//go:embed sprites/sheet.json
var sheetJSON string

//go:embed sounds/jump.ogg
var jumpOGG []byte

// SheetPNG returns the sprite sheet image (PNG bytes).
func SheetPNG() []byte { return sheetPNG }

// SheetJSON returns the sprite sheet metadata (TexturePacker JSON).
func SheetJSON() string { return sheetJSON }

// JumpOGG returns the jump sound (OGG Vorbis bytes).
func JumpOGG() []byte { return jumpOGG }
