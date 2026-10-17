package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Player physics, in pixels per tick (Ebitengine runs 60 ticks per second).
// The numbers follow the gopher sprite's demo (demo.html), doubled to match
// spriteScale: accelerate while a direction is held, friction every tick
// (which caps the top speed), gravity pulls down, jump launches only from
// the ground.
const (
	moveAccel = 0.2   // horizontal acceleration per tick
	friction  = 0.92  // horizontal velocity kept per tick
	gravity   = 0.07  // downward acceleration per tick
	jumpVel   = -1.75 // launch velocity, negative is up
	maxFall   = 4     // terminal fall speed per tick

	// idleVel is the horizontal speed below which the gopher is considered
	// stopped and shows the idle frame.
	idleVel = 0.05

	// groundHeight is the height of the ground strip the player stands on.
	groundHeight = 16

	// groundY is the screen height of the top of the ground: the player's
	// feet rest here.
	groundY = ScreenHeight - groundHeight
)

// input is one tick of keyboard input. Update gathers it from the keyboard
// and hands it to the player, so the physics can be driven from tests
// without a window.
type input struct {
	left  bool
	right bool
	jump  bool
}

// player is the demo character. Its position is the top-left corner of the
// scaled sprite; velocity is in pixels per tick.
type player struct {
	x, y     float64
	vx, vy   float64
	onGround bool
	facing   int // +1 right, -1 left; the sheet art faces left
	animTick int // ticks spent walking, drives the walk cycle
}

// newPlayer returns the player standing at the center of the ground.
func newPlayer() *player {
	return &player{
		x:        float64((ScreenWidth - sheetMeta.scaledWidth()) / 2),
		y:        float64(groundY - sheetMeta.scaledHeight()),
		onGround: true,
		facing:   1,
	}
}

// update advances the player one tick: horizontal input with friction
// (friction every tick caps the top speed, as in the demo), jumping from
// the ground, gravity, then clamping to the ground and the screen edges.
func (p *player) update(in input) {
	switch {
	case in.left && !in.right:
		p.vx -= moveAccel
		p.facing = -1
	case in.right && !in.left:
		p.vx += moveAccel
		p.facing = 1
	}
	p.vx *= friction

	if in.jump && p.onGround {
		p.vy = jumpVel
		p.onGround = false
	}

	p.vy += gravity
	if p.vy > maxFall {
		p.vy = maxFall
	}

	p.x += p.vx
	p.y += p.vy

	// Land on the ground.
	if p.y >= float64(groundY-sheetMeta.scaledHeight()) {
		p.y = float64(groundY - sheetMeta.scaledHeight())
		p.vy = 0
		p.onGround = true
	}

	// Stay on the screen.
	if p.x < 0 {
		p.x = 0
		p.vx = 0
	}
	if max := float64(ScreenWidth - sheetMeta.scaledWidth()); p.x > max {
		p.x = max
		p.vx = 0
	}

	// Animate the walk cycle only while walking on the ground.
	if p.onGround && (in.left || in.right) {
		p.animTick++
	}
}

// frame returns the sheet frame for the player's current state: jump
// rise/float/fall frames by vertical speed while airborne, the walk cycle
// while moving, and the idle frame otherwise.
func (p *player) frame() int {
	if !p.onGround {
		switch {
		case p.vy < -0.2:
			return sheetMeta.jump[0]
		case p.vy < 0.2:
			return sheetMeta.jump[1]
		default:
			return sheetMeta.jump[2]
		}
	}
	if p.vx < idleVel && p.vx > -idleVel {
		return sheetMeta.idle
	}
	return sheetMeta.walk[(p.animTick/sheetMeta.walkTickStep)%len(sheetMeta.walk)]
}

// draw draws the player's current frame, mirrored when facing right (the
// sheet art faces left).
func (p *player) draw(screen *ebiten.Image) error {
	return drawSprite(screen, p.frame(), p.x, p.y, p.facing > 0)
}
