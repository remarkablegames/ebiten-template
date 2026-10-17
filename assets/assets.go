// Package assets embeds the game's art and metadata into every build.
//
// The files are compiled in with go:embed, so whatever is on disk when the
// binary is built is what ships: replace a file and restart the game to pick
// it up. The hot-reload script reads them through this package (exposed to
// the interpreter in symbols/) because yaegi cannot process go:embed itself.
package assets

import _ "embed"

//go:embed sprites/sheet.png
var sheetPNG []byte

//go:embed sprites/sheet.json
var sheetJSON string

// SheetPNG returns the sprite sheet image (PNG bytes).
func SheetPNG() []byte { return sheetPNG }

// SheetJSON returns the sprite sheet metadata (TexturePacker JSON): the frame
// rectangles, per-frame durations, and tag ranges the game resolves at load.
func SheetJSON() string { return sheetJSON }
