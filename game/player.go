package game

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Player physics, in pixels per tick (60 ticks per second).
const (
	moveAccel = 0.2   // horizontal acceleration per tick
	friction  = 0.92  // horizontal velocity kept per tick
	gravity   = 0.07  // downward acceleration per tick
	jumpVel   = -1.75 // launch velocity, negative is up
	maxFall   = 4     // terminal fall speed per tick

	// idleVel is the speed below which the sprite counts as stopped.
	idleVel = 0.05

	// groundHeight is the ground strip's height.
	groundHeight = 16

	// groundY is the top of the ground; the player's feet rest here.
	groundY = ScreenHeight - groundHeight
)

// input is one tick of keyboard input, so physics can be tested headless.
type input struct {
	left  bool
	right bool
	jump  bool
}

// player is the character; x, y is the scaled sprite's top-left corner.
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

// update advances the player one tick: input and friction, jumping from the
// ground, gravity, then clamping to the ground and screen edges. It reports
// whether this tick launched a jump, so the caller can trigger the jump sound
// without the physics knowing about audio.
func (p *player) update(in input) (jumped bool) {
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
		jumped = true
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

	return jumped
}

// frame returns the sheet frame for the current state: jump frames by
// vertical speed while airborne, the walk cycle while moving, else idle.
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
