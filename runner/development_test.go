//go:build !js && !release

package runner

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/game"
)

// TestEvalHooks runs the script the way the game does at startup: interpret
// it, call every hook, and check the layout matches the compiled game.
func TestEvalHooks(t *testing.T) {
	files, err := readScriptFiles("../game")
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	h, err := evalHooks(files)
	if err != nil {
		t.Fatalf("evalHooks: %v", err)
	}
	if err := h.update(); err != nil {
		t.Fatalf("update: %v", err)
	}
	h.draw(ebiten.NewImage(32, 32))

	wantW, wantH := game.Layout(640, 480)
	gotW, gotH := h.layout(640, 480)
	if gotW != wantW || gotH != wantH {
		t.Errorf("layout = %dx%d, want %dx%d", gotW, gotH, wantW, wantH)
	}
}

// TestEvalHooksDecodeSheet interprets a script that decodes the embedded
// sheet the way the game does, proving the interpreter can fetch bytes from
// the assets package. The game's Draw swallows this error via DebugPrint, so
// TestEvalHooks alone would not notice a broken interpreted PNG path.
func TestEvalHooksDecodeSheet(t *testing.T) {
	src := `package game

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/assets"
)

func Update() error {
	img, _, err := image.Decode(bytes.NewReader(assets.SheetPNG()))
	if err != nil {
		return err
	}
	if got, want := img.Bounds().Dx(), 324; got != want {
		return fmt.Errorf("width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), 14; got != want {
		return fmt.Errorf("height = %d, want %d", got, want)
	}
	return nil
}

func Draw(screen *ebiten.Image) {}

func Layout(w, h int) (int, int) { return 0, 0 }
`
	h, err := evalHooks([]scriptFile{{path: "game/decode.go", src: []byte(src)}})
	if err != nil {
		t.Fatalf("evalHooks: %v", err)
	}
	if err := h.update(); err != nil {
		t.Fatalf("decode sheet: %v", err)
	}
}


// TestEvalHooksFileOrder evaluates a consumer before the file it references,
// proving evalHooks loads files in any order (by retrying). Both files import
// "time", so the test also covers two script files sharing one import, which
// yaegi rejects when files import directly instead of through the hoisted
// preamble.
func TestEvalHooksFileOrder(t *testing.T) {
	files := []scriptFile{
		{
			path: "game/game.go",
			src: []byte(`package game

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var started = time.Now()

func Update() error { _ = time.Since(started); return nil }
func Draw(screen *ebiten.Image) {}
func Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}
`),
		},
		{
			path: "game/config.go",
			src: []byte(`package game

import "time"

const (
	ScreenWidth  = 320
	ScreenHeight = 240
)

var loaded = time.Now()
`),
		},
	}
	h, err := evalHooks(files)
	if err != nil {
		t.Fatalf("evalHooks: %v", err)
	}
	gotW, gotH := h.layout(640, 480)
	if gotW != 320 || gotH != 240 {
		t.Errorf("layout = %dx%d, want 320x240", gotW, gotH)
	}
}

// TestEvalHooksFileOrderFunctionCall evaluates a consumer before a referenced
// file and needs more than one retry round since an undefined function call
// panics yaegi mid-file rather than returning an error.
func TestEvalHooksFileOrderFunctionCall(t *testing.T) {
	files := []scriptFile{
		{
			path: "game/game.go",
			src: []byte(`package game

import "github.com/hajimehoshi/ebiten/v2"

var player = newPlayer()

func Update() error { player.step(); return nil }
func Draw(screen *ebiten.Image) {}
func Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }
`),
		},
		{
			path: "game/player.go",
			src: []byte(`package game

type player struct{ x float64 }

func newPlayer() *player { return &player{x: PlayerStartX} }
func (p *player) step()  { p.x++ }
`),
		},
		{
			path: "game/config.go",
			src: []byte(`package game

const (
	ScreenWidth  = 320
	ScreenHeight = 240

	PlayerStartX = 100
)
`),
		},
	}
	h, err := evalHooks(files)
	if err != nil {
		t.Fatalf("evalHooks: %v", err)
	}
	if err := h.update(); err != nil {
		t.Fatalf("update: %v", err)
	}
}

// TestEvalHooksFoldedError documents, in miniature, the pattern the game
// script uses for package vars that can fail to load (see game/sprite.go): a
// two-value initializer (var m, err = f()) miscompiles in yaegi when the
// script is split across files and methods are called on the var — load
// succeeds, then the hooks panic with a reflect.Set type mismatch. Folding
// the error into the value avoids it.
func TestEvalHooksFoldedError(t *testing.T) {
	files := []scriptFile{
		{
			path: "game/game.go",
			src: []byte(`package game

import "github.com/hajimehoshi/ebiten/v2"

var p = newPlayer()

func Update() error { p.step(); return nil }
func Draw(screen *ebiten.Image) {}
func Layout(w, h int) (int, int) { return ScreenWidth, ScreenHeight }
`),
		},
		{
			path: "game/player.go",
			src: []byte(`package game

type player struct{ x float64 }

func newPlayer() *player { return &player{x: float64(cfg.width())} }
func (p *player) step()  { p.x += float64(cfg.width()) }
`),
		},
		{
			path: "game/sprite.go",
			src: []byte(`package game

import "errors"

type config struct {
	N   int
	err error
}

func (c config) width() int { return c.N * 2 }

func loadConfig() config {
	var c config
	if err := parseConfig(&c); err != nil {
		c.err = err
	}
	return c
}

func parseConfig(c *config) error {
	c.N = 12
	return errors.New("x")
}

var cfg = loadConfig()
`),
		},
		{
			path: "game/config.go",
			src: []byte(`package game

const (
	ScreenWidth  = 320
	ScreenHeight = 240
)
`),
		},
	}
	h, err := evalHooks(files)
	if err != nil {
		t.Fatalf("evalHooks: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("update panicked: %v", r)
		}
	}()
	if err := h.update(); err != nil {
		t.Fatalf("update: %v", err)
	}
}

// TestEvalHooksBrokenScript checks that every way a script can fail surfaces
// as an error, not a panic: yaegi errors on syntax problems and panics on
// unresolved symbols or missing hooks, and evalHooks must report both.
func TestEvalHooksBrokenScript(t *testing.T) {
	tests := map[string]string{
		"syntax error":   "package game\nfunc Update() { x := }\nfunc Draw(s *ebiten.Image) {}\nfunc Layout(w, h int) (int, int) { return 0, 0 }",
		"unknown import": "package game\nfunc Update() error { return nil }\nfunc Draw(s *ebiten.Image) {}\nfunc Layout(w, h int) (int, int) { return 0, 0 }",
		"missing hooks":  "package game\nfunc Update() error { return nil }",
		"wrong signature": `package game

import "github.com/hajimehoshi/ebiten/v2"

func Update() int { return 0 }
func Draw(screen *ebiten.Image) {}
func Layout(w, h int) (int, int) { return 0, 0 }
`,
	}
	for name, src := range tests {
		t.Run(name, func(t *testing.T) {
			file := scriptFile{path: "game/game.go", src: []byte(src)}
			if _, err := evalHooks([]scriptFile{file}); err == nil {
				t.Fatal("evalHooks succeeded, want error")
			}
		})
	}
}
