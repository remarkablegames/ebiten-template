package assets

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/png"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// sheet is the subset of the metadata format the cross-check needs: frame
// rectangles plus the declared sheet size.
type sheet struct {
	Frames []struct {
		Frame struct {
			X int `json:"x"`
			Y int `json:"y"`
			W int `json:"w"`
			H int `json:"h"`
		} `json:"frame"`
		Rotated bool `json:"rotated"`
	} `json:"frames"`
	Meta struct {
		Size struct {
			W int `json:"w"`
			H int `json:"h"`
		} `json:"size"`
	} `json:"meta"`
}

// TestEmbeddedSheetConsistency cross-checks the embedded PNG against the
// embedded sheet.json: every frame rect must fit inside the PNG, no frame may
// be rotated or empty, and meta.size must match the PNG. This used to run
// when regenerating the old embedded data; now that both files are embedded
// directly it runs at test time, so swapping one file without the other fails
// `go test ./...` instead of showing up on screen.
func TestEmbeddedSheetConsistency(t *testing.T) {
	pngData, jsonData := SheetPNG(), []byte(SheetJSON())

	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	var s sheet
	if err := json.Unmarshal(jsonData, &s); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(s.Frames) == 0 {
		t.Fatal("parse json: no frames")
	}

	b := img.Bounds()
	for i, f := range s.Frames {
		if f.Rotated {
			t.Fatalf("frame %d is rotated, which the game does not support", i)
		}
		if f.Frame.W <= 0 || f.Frame.H <= 0 {
			t.Fatalf("frame %d has empty rect %dx%d", i, f.Frame.W, f.Frame.H)
		}
		if f.Frame.X < 0 || f.Frame.Y < 0 ||
			f.Frame.X+f.Frame.W > b.Dx() || f.Frame.Y+f.Frame.H > b.Dy() {
			t.Fatalf("frame %d rect %+v does not fit image %dx%d (png and json out of sync?)",
				i, f.Frame, b.Dx(), b.Dy())
		}
	}
	if s.Meta.Size.W != 0 && (s.Meta.Size.W != b.Dx() || s.Meta.Size.H != b.Dy()) {
		t.Fatalf("meta.size %dx%d does not match image %dx%d (png and json out of sync?)",
			s.Meta.Size.W, s.Meta.Size.H, b.Dx(), b.Dy())
	}
}

// TestEmbeddedJumpOGG checks the embedded jump sound decodes as OGG Vorbis,
// so swapping in a corrupt or non-Vorbis file fails `go test ./...` instead
// of the sound silently failing at runtime.
func TestEmbeddedJumpOGG(t *testing.T) {
	stream, err := vorbis.DecodeWithoutResampling(bytes.NewReader(JumpOGG()))
	if err != nil {
		t.Fatalf("decode jump.ogg: %v", err)
	}
	if stream.Length() == 0 {
		t.Fatal("jump.ogg decodes to zero samples")
	}
}
