package game

import (
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestDecodeSheet checks the embedded sheet decodes and matches the size its
// metadata declares.
func TestDecodeSheet(t *testing.T) {
	img, err := decodeSheet()
	if err != nil {
		t.Fatalf("decodeSheet: %v", err)
	}
	b := img.Bounds()
	if got, want := b.Dx(), 324; got != want {
		t.Errorf("width = %d, want %d", got, want)
	}
	if got, want := b.Dy(), 14; got != want {
		t.Errorf("height = %d, want %d", got, want)
	}
}

// TestResolveSheetMeta checks the embedded sheet.json resolves to the frames
// the game animates with: idle, the walk cycle, the jump trio, and the
// animation speed, all straight from the sheet's tags and durations.
func TestResolveSheetMeta(t *testing.T) {
	if sheetMeta.err != nil {
		t.Fatalf("resolveSheetMeta: %v", sheetMeta.err)
	}
	if got, want := len(sheetMeta.frames), 27; got != want {
		t.Errorf("frames = %d, want %d", got, want)
	}
	if got, want := sheetMeta.idle, 3; got != want {
		t.Errorf("idle = %d, want %d", got, want)
	}
	if got, want := len(sheetMeta.walk), 8; got != want {
		t.Fatalf("walk frames = %d, want %d", got, want)
	}
	if got, want := sheetMeta.walk[0], 8; got != want {
		t.Errorf("walk starts at %d, want %d", got, want)
	}
	if got, want := sheetMeta.jump, [3]int{24, 25, 26}; got != want {
		t.Errorf("jump = %v, want %v", got, want)
	}
	// The sheet shows each frame for 100ms; at 60 ticks per second that is
	// 6 ticks.
	if got, want := sheetMeta.walkTickStep, 6; got != want {
		t.Errorf("walkTickStep = %d, want %d", got, want)
	}
	if got, want := sheetMeta.scaledWidth(), 24; got != want {
		t.Errorf("scaledWidth = %d, want %d", got, want)
	}
	if got, want := sheetMeta.scaledHeight(), 28; got != want {
		t.Errorf("scaledHeight = %d, want %d", got, want)
	}
}

// TestResolveSheetMetaFrameRect checks the sheet region sliced for a frame
// comes from the metadata, not from arithmetic on frame numbers.
func TestResolveSheetMetaFrameRect(t *testing.T) {
	if sheetMeta.err != nil {
		t.Fatalf("resolveSheetMeta: %v", sheetMeta.err)
	}
	want := image.Rect(288, 0, 300, 14)
	if got := sheetMeta.frames[24].rect; got != want {
		t.Errorf("frame 24 rect = %v, want %v", got, want)
	}
}

// TestTransformSpriteFlipsInPlace checks that mirroring a frame keeps it in
// the same [x, x+width] box, only reversed: the sprite must not jump when the
// player turns around, and the flipped sprite's right edge must still reach
// x+width so the player can walk to the screen edge. (The flip translate was
// once negative, which drew the mirror two sprite-widths to the left.)
func TestTransformSpriteFlipsInPlace(t *testing.T) {
	f := sheetMeta.frames[3]
	const x, y = 100.0, 50.0

	tests := map[string]struct {
		flip bool
		// The frame's left edge (source u=0) and right edge (source u=12)
		// map to these drawn x positions. Flipping mirrors the edges within
		// the same box.
		edgeLeft, edgeRight float64
	}{
		"faces left":  {flip: false, edgeLeft: x, edgeRight: x + 24},
		"faces right": {flip: true, edgeLeft: x + 24, edgeRight: x},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var op ebiten.DrawImageOptions
			transformSprite(&op, f, x, y, tc.flip)
			left, _ := op.GeoM.Apply(0, 0)
			right, _ := op.GeoM.Apply(float64(f.rect.Dx()), 0)
			if left != tc.edgeLeft || right != tc.edgeRight {
				t.Errorf("frame edges drawn at [%v, %v], want [%v, %v]", left, right, tc.edgeLeft, tc.edgeRight)
			}
		})
	}
}

// TestResolveSheetMetaErrors checks that a sheet the game cannot use fails
// with a message naming the problem.
func TestResolveSheetMetaErrors(t *testing.T) {
	tests := map[string]string{
		"bad JSON":  "not json at all",
		"no frames": `{"frames": [], "meta": {"frameTags": []}}`,
		"no idle":   `{"frames": [{"frame": {"x":0,"y":0,"w":12,"h":14}}], "meta": {"frameTags": []}}`,
		"empty tag": `{"frames": [{"frame": {"x":0,"y":0,"w":12,"h":14}}], "meta": {"frameTags": [{"name":"Walk","from":1,"to":0}]}}`,
		"tag range": `{"frames": [{"frame": {"x":0,"y":0,"w":12,"h":14}}], "meta": {"frameTags": [{"name":"Left","from":0,"to":0},{"name":"Walk","from":0,"to":5}]}}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if m := resolveSheetMeta(raw); m.err == nil {
				t.Fatal("resolveSheetMeta succeeded, want error")
			} else if name == "bad JSON" && !strings.Contains(m.err.Error(), "sheet.json") {
				t.Errorf("error = %q, want it to mention sheet.json", m.err)
			}
		})
	}
}
