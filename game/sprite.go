package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/remarkablegames/ebiten-template/assets"
)

// spriteScale draws each frame this many times larger, so the pixel art is
// readable on the 320x240 screen.
const spriteScale = 2

// ticksPerSecond converts the per-frame durations in sheet.json (milliseconds)
// to animation speed in ticks.
const ticksPerSecond = 60

// The animations the game plays are looked up by tag name in sheet.json, so
// the sheet's metadata (not this code) decides which frames show and for how
// long. The gopher sheet tags its standing pose "Left" because its art faces
// left by default, and its "Jump" tag holds three frames the game reads as
// rise, float and fall.
const (
	idleTag = "Left"
	walkTag = "Walk"
	jumpTag = "Jump"
)

// frame is one frame of the sheet: the region to draw and how long to show
// it, in ticks.
type frame struct {
	rect  image.Rectangle
	ticks int
}

// sheetLayout is the sheet.json metadata resolved for play: every frame, plus
// the frames each animation uses. err records why the sheet could not be
// used (bad JSON, missing tag); Update reports it so the game shows the
// problem instead of animating the wrong frames.
type sheetLayout struct {
	frames       []frame // one entry per sheet frame, indexed by frame number
	idle         int     // standing frame
	walk         []int   // walk cycle, in play order
	jump         [3]int  // rise, float, fall
	walkTickStep int     // ticks each walk frame is shown
	err          error
}

// scaledWidth is the on-screen width of one sprite frame, taken from the
// sheet's own frame size.
func (m sheetLayout) scaledWidth() int {
	if len(m.frames) == 0 {
		return 0
	}
	return m.frames[m.idle].rect.Dx() * spriteScale
}

// scaledHeight is the on-screen height of one sprite frame, taken from the
// sheet's own frame size.
func (m sheetLayout) scaledHeight() int {
	if len(m.frames) == 0 {
		return 0
	}
	return m.frames[m.idle].rect.Dy() * spriteScale
}

// sheetDoc mirrors the TexturePacker JSON metadata the assets package embeds
// (see assets/sprites/sheet.json).
type sheetDoc struct {
	Frames []jsonFrame `json:"frames"`
	Meta   jsonMeta    `json:"meta"`
}

type jsonFrame struct {
	Frame    jsonRect `json:"frame"`
	Duration int      `json:"duration"` // milliseconds this frame is shown
}

type jsonRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type jsonMeta struct {
	FrameTags []jsonTag `json:"frameTags"`
}

type jsonTag struct {
	Name string `json:"name"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// tagRange is the inclusive frame range one tag covers.
type tagRange struct{ first, last int }

// resolveSheetMeta parses sheet.json (as read by the assets package) and
// resolves the tags the game animates with. Frame regions and animation
// speeds come entirely from the sheet, so a new sheet needs no code changes
// as long as it keeps the idle, walk and jump tags.
//
// It returns one value with the error folded in, not (layout, error): a
// two-value package var initializer corrupts yaegi's frame types when the
// interpreted script is split across files and the var has methods called
// on it.
func resolveSheetMeta(raw string) sheetLayout {
	var m sheetLayout
	if err := parseSheetMeta(raw, &m); err != nil {
		m.err = err
	}
	return m
}

// parseSheetMeta resolves the sheet JSON into m, returning why the sheet
// cannot be used if it cannot.
func parseSheetMeta(raw string, m *sheetLayout) error {
	var doc sheetDoc
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return fmt.Errorf("parse sheet.json: %w", err)
	}
	if len(doc.Frames) == 0 {
		return errors.New("sheet.json has no frames")
	}

	m.frames = make([]frame, len(doc.Frames))
	for i, f := range doc.Frames {
		m.frames[i] = frame{
			rect:  image.Rect(f.Frame.X, f.Frame.Y, f.Frame.X+f.Frame.W, f.Frame.Y+f.Frame.H),
			ticks: f.Duration * ticksPerSecond / 1000,
		}
	}

	tags := make(map[string]tagRange, len(doc.Meta.FrameTags))
	for _, t := range doc.Meta.FrameTags {
		tags[t.Name] = tagRange{first: t.From, last: t.To}
	}

	idle, err := tagFrames(tags, idleTag, len(m.frames))
	if err != nil {
		return err
	}
	m.idle = idle[0]

	if m.walk, err = tagFrames(tags, walkTag, len(m.frames)); err != nil {
		return err
	}
	m.walkTickStep = m.frames[m.walk[0]].ticks
	if m.walkTickStep < 1 {
		m.walkTickStep = 1
	}

	jump, err := tagFrames(tags, jumpTag, len(m.frames))
	if err != nil {
		return err
	}
	m.jump = [3]int{jump[0], jump[len(jump)/2], jump[len(jump)-1]}

	return nil
}

// tagFrames expands a tag into the frame numbers it covers, checking the
// tag exists and points at frames the sheet actually has.
func tagFrames(tags map[string]tagRange, name string, frames int) ([]int, error) {
	t, ok := tags[name]
	if !ok {
		return nil, fmt.Errorf("sheet.json: no %q tag", name)
	}
	if t.first < 0 || t.last >= frames || t.first > t.last {
		return nil, fmt.Errorf("sheet.json: tag %q covers frames %d..%d, sheet has %d frames",
			name, t.first, t.last, frames)
	}
	out := make([]int, 0, t.last-t.first+1)
	for i := t.first; i <= t.last; i++ {
		out = append(out, i)
	}
	return out, nil
}

// sheetMeta is sheet.json resolved for play, evaluated when the script loads.
// sheetMeta.err records why the sheet could not be used (bad JSON, missing
// tag); Update reports it so the game shows the problem instead of animating
// the wrong frames.
//
// It is declared below resolveSheetMeta on purpose: the hot-reload
// interpreter evaluates declarations in a file top-to-bottom, so an
// initializer calling a function defined further down would not resolve.
var sheetMeta = resolveSheetMeta(assets.SheetJSON())

// sheet is the decoded sprite sheet, created on first use so the script can
// be evaluated before the graphics context exists (tests, hot reload).
var sheet *ebiten.Image

// decodeSheet decodes the sprite sheet bytes embedded by the assets package.
func decodeSheet() (*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(assets.SheetPNG()))
	if err != nil {
		return nil, fmt.Errorf("decode sheet: %w", err)
	}
	return ebiten.NewImageFromImage(img), nil
}

// loadSheet decodes the sprite sheet once and returns it.
func loadSheet() (*ebiten.Image, error) {
	if sheet == nil {
		img, err := decodeSheet()
		if err != nil {
			return nil, err
		}
		sheet = img
	}
	return sheet, nil
}

// transformSprite sets op's geometry for drawing frame f at (x, y): scaled
// up by spriteScale and mirrored horizontally when flip is true. Both
// facings land in the same [x, x+scaled width] box with only the art
// reversed: Scale(-1, 1) reflects the frame across the origin into
// [-width, 0], so the flip translates it by +width to bring it back over x.
// The sign is easy to get backwards — translating by -width draws the mirror
// two widths to the left of the sprite's position, which shows up as the
// sprite teleporting on every turn of direction and stopping short of the
// right edge of the screen.
func transformSprite(op *ebiten.DrawImageOptions, f frame, x, y float64, flip bool) {
	op.GeoM.Scale(spriteScale, spriteScale)
	if flip {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(f.rect.Dx()*spriteScale), 0)
	}
	op.GeoM.Translate(x, y)
}

// drawSprite draws frame i of the sheet at (x, y), scaled up by spriteScale
// and mirrored horizontally when flip is true.
func drawSprite(screen *ebiten.Image, i int, x, y float64, flip bool) error {
	if i < 0 || i >= len(sheetMeta.frames) {
		return fmt.Errorf("frame %d out of range, sheet has %d frames", i, len(sheetMeta.frames))
	}
	img, err := loadSheet()
	if err != nil {
		return err
	}
	f := sheetMeta.frames[i]
	frame := img.SubImage(f.rect).(*ebiten.Image)

	op := &ebiten.DrawImageOptions{}
	transformSprite(op, f, x, y, flip)
	screen.DrawImage(frame, op)
	return nil
}
