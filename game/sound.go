package game

import (
	"bytes"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/remarkablegames/ebiten-template/assets"
)

// sampleRate is the audio context's rate in Hz. The context is process-global,
// so every sound resamples to this one rate.
const sampleRate = 44100

// audioContext returns the process-wide audio context, creating it on first
// use. The context must outlive hot reloads: audio.NewContext panics if called
// twice in a process, and package vars are re-evaluated on every reload.
func audioContext() *audio.Context {
	if ctx := audio.CurrentContext(); ctx != nil {
		return ctx
	}

	return audio.NewContext(sampleRate)
}

// The jump player and the last sound error. Both reset on hot reload (the
// interpreter restarts), so the player is rebuilt lazily on the next jump.
// The error is shown on screen rather than stopping the game.
var (
	jumpPlayer *audio.Player
	soundErr   error
)

// playJump restarts the jump sound from the beginning. A jump cannot retrigger
// mid-air.
func playJump() {
	if jumpPlayer == nil {
		ctx := audioContext()
		stream, err := vorbis.DecodeWithSampleRate(ctx.SampleRate(), bytes.NewReader(assets.JumpOGG()))

		if err != nil {
			soundErr = fmt.Errorf("decode jump sound: %w", err)
			return
		}

		if jumpPlayer, err = ctx.NewPlayer(stream); err != nil {
			soundErr = fmt.Errorf("jump sound player: %w", err)
			return
		}
	}

	if err := jumpPlayer.Rewind(); err != nil {
		soundErr = fmt.Errorf("rewind jump sound: %w", err)
		return
	}

	jumpPlayer.Play()
}
