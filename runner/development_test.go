//go:build !js && !release

package runner

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/game"
)

// TestEvalHooks runs the game script through the same path the game uses at
// startup: interpret it, call every hook, and check the layout matches the
// compiled game. The script is read from the repository because tests run
// with this package's directory as the working directory.
func TestEvalHooks(t *testing.T) {
	src, err := os.ReadFile("../game/game.go")
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	h, err := evalHooks(src)
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

// TestEvalHooksBrokenScript checks that every way a script can fail surfaces
// as an error instead of a panic: yaegi returns errors for syntax problems
// and panics for unresolved symbols or missing hooks, and evalHooks must
// report both.
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
			if _, err := evalHooks([]byte(src)); err == nil {
				t.Fatal("evalHooks succeeded, want error")
			}
		})
	}
}
