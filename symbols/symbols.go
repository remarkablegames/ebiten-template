// Package symbols exposes compiled package APIs to the yaegi interpreter so
// the hot-reloaded game script can import them like normal Go packages.
//
// The entries are filled by generated files in this directory. To expose
// another package to the script, add a generate directive below and run
// `go generate ./symbols` from the repository root.
package symbols

import "reflect"

// Generate the symbol maps for every package the game script may import.
// Each command must run with this directory as the working directory (go
// generate does that automatically): the tool names the generated package
// after the directory it runs in.

//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2
//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2/audio
//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2/audio/vorbis
//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2/ebitenutil
//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2/inpututil
//go:generate go tool yaegi extract github.com/hajimehoshi/ebiten/v2/vector
//go:generate go tool yaegi extract github.com/remarkablegames/ebiten-template/assets

// Symbols maps "import/path/package name" to the exported symbols of that
// package. It is passed to interp.Use.
var Symbols = map[string]map[string]reflect.Value{}
