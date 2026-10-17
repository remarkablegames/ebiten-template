// Package assets embeds the game's art and metadata with go:embed.
// The hot-reload script reads them here (exposed via symbols/).
package assets

import _ "embed"

//go:embed sprites/sheet.png
var sheetPNG []byte

//go:embed sprites/sheet.json
var sheetJSON string

// SheetPNG returns the sprite sheet image (PNG bytes).
func SheetPNG() []byte { return sheetPNG }

// SheetJSON returns the sprite sheet metadata (TexturePacker JSON).
func SheetJSON() string { return sheetJSON }
