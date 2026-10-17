package game

import "testing"

// TestPlayerMovesRight checks that holding right accelerates the player and
// faces right (the sheet art faces left, so facing right means flipped).
func TestPlayerMovesRight(t *testing.T) {
	p := newPlayer()
	start := p.x
	p.update(input{right: true})
	if p.x <= start {
		t.Errorf("x = %v, want > %v", p.x, start)
	}
	if p.facing != 1 {
		t.Errorf("facing = %d, want 1", p.facing)
	}
}

// TestPlayerMovesLeft checks that holding left accelerates the player and
// faces left.
func TestPlayerMovesLeft(t *testing.T) {
	p := newPlayer()
	start := p.x
	p.update(input{left: true})
	if p.x >= start {
		t.Errorf("x = %v, want < %v", p.x, start)
	}
	if p.facing != -1 {
		t.Errorf("facing = %d, want -1", p.facing)
	}
}

// TestPlayerStaysOnScreen checks that the player cannot walk off either edge
// of the screen.
func TestPlayerStaysOnScreen(t *testing.T) {
	p := newPlayer()
	for range 1000 {
		p.update(input{left: true})
	}
	if p.x != 0 {
		t.Errorf("x = %v, want 0", p.x)
	}
	for range 1000 {
		p.update(input{right: true})
	}
	if max := float64(ScreenWidth - sheetMeta.scaledWidth()); p.x != max {
		t.Errorf("x = %v, want %v", p.x, max)
	}
}

// TestPlayerJumpsOnlyFromGround checks that jump launches the player
// upward, that a second jump mid-air is ignored, and that gravity brings
// the player back to the ground.
func TestPlayerJumpsOnlyFromGround(t *testing.T) {
	p := newPlayer()
	p.update(input{jump: true})
	if p.vy >= 0 {
		t.Fatalf("vy = %v, want < 0 after jump", p.vy)
	}
	if p.onGround {
		t.Fatal("onGround = true, want false after jump")
	}

	// Mid-air jump attempt must not relaunch: a relaunch would reset vy to
	// the launch velocity, so vy must stay strictly above it.
	p.update(input{jump: true})
	if p.vy <= jumpVel+gravity {
		t.Errorf("vy = %v, want > %v (no double jump)", p.vy, jumpVel+gravity)
	}

	// Land again.
	for range 600 {
		p.update(input{})
	}
	if !p.onGround {
		t.Error("onGround = false, want true after falling")
	}
	if want := float64(groundY - sheetMeta.scaledHeight()); p.y != want {
		t.Errorf("y = %v, want %v", p.y, want)
	}
}

// TestPlayerFrameIdleAndWalk checks the animation state on the ground:
// the idle frame when still, walk frames while moving.
func TestPlayerFrameIdleAndWalk(t *testing.T) {
	p := newPlayer()
	if got := p.frame(); got != sheetMeta.idle {
		t.Errorf("frame = %d, want idle %d", got, sheetMeta.idle)
	}
	for range sheetMeta.walkTickStep {
		p.update(input{right: true})
	}
	if got := p.frame(); got < sheetMeta.walk[0] || got > sheetMeta.walk[len(sheetMeta.walk)-1] {
		t.Errorf("frame = %d, want walk %d..%d", got, sheetMeta.walk[0], sheetMeta.walk[len(sheetMeta.walk)-1])
	}
}

// TestPlayerFrameWhileAirborne checks the jump frames are chosen by vertical
// speed: rise going up, float near the apex, falling on the way down.
func TestPlayerFrameWhileAirborne(t *testing.T) {
	p := newPlayer()
	p.update(input{jump: true})
	if p.onGround {
		t.Fatal("onGround = true, want false after jump")
	}
	if got := p.frame(); got != sheetMeta.jump[0] {
		t.Errorf("frame = %d, want rise %d", got, sheetMeta.jump[0])
	}

	// Run until the player is clearly falling (past the float band at the
	// apex).
	for p.vy < 0.2 {
		p.update(input{})
	}
	if p.onGround {
		t.Fatal("onGround = true, want false while falling")
	}
	if got := p.frame(); got != sheetMeta.jump[2] {
		t.Errorf("frame = %d, want fall %d", got, sheetMeta.jump[2])
	}
}
