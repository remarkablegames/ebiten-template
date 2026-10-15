//go:build !js && !release

// Package runner supplies the game to ebiten.RunGame.
//
// In development builds (the default on desktop), this file interprets the
// game logic in package game with yaegi and hot-reloads it: it watches
// game/game.go and swaps the running Update/Draw/Layout hooks whenever the
// file is saved. Any failure (compile error, missing symbol, runtime panic)
// is shown on screen and the last working version keeps running.
//
// Release builds (-tags release) and web builds (GOOS=js) use production.go
// instead, which runs the compiled game package.
package runner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/remarkablegames/ebiten-template/game"
	"github.com/remarkablegames/ebiten-template/symbols"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const (
	scriptPath     = "game/game.go"
	scriptDir      = "game"
	reloadDebounce = 200 * time.Millisecond
)

// hooks are the game callbacks the shell delegates to. They come either from
// the interpreted script or, as a fallback, from the compiled game package.
type hooks struct {
	update func() error
	draw   func(*ebiten.Image)
	layout func(int, int) (int, int)
}

// compiledHooks returns the game logic compiled into the binary. It is the
// baseline the shell falls back to when the script cannot be loaded.
func compiledHooks() hooks {
	return hooks{
		update: game.Update,
		draw:   game.Draw,
		layout: game.Layout,
	}
}

// hotReloadGame implements ebiten.Game by delegating to hooks that are
// swapped whenever the script changes on disk.
type hotReloadGame struct {
	current       hooks
	pendingReload atomic.Bool // set by the watcher goroutine
	lastErr       error       // only touched on the game goroutine
}

// NewGame returns the game to run in development: the script, interpreted
// live. If game/game.go cannot be read (the binary is running from another
// working directory), the compiled game runs instead.
func NewGame() ebiten.Game {
	g := &hotReloadGame{current: compiledHooks()}
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		log.Printf("hot reload: cannot read %s, running compiled game: %v", scriptPath, err)
	} else {
		g.load(src)
	}
	g.watch()
	return g
}

// Update consumes a pending reload, then runs one tick of game logic.
// Script failures are recorded for display instead of stopping the game.
func (g *hotReloadGame) Update() error {
	if g.pendingReload.Swap(false) {
		src, err := os.ReadFile(scriptPath)
		if err != nil {
			g.lastErr = err
		} else {
			g.load(src)
		}
	}
	defer g.recoverHook("Update")
	if err := g.current.update(); err != nil {
		g.lastErr = err
	}
	return nil
}

// Draw renders one frame and, if the last reload or a script hook failed,
// the error on top of it.
func (g *hotReloadGame) Draw(screen *ebiten.Image) {
	g.drawScript(screen)
	if g.lastErr != nil {
		ebitenutil.DebugPrint(screen, "SCRIPT ERROR:\n"+g.lastErr.Error())
	}
}

// drawScript runs the script Draw with panic recovery, so the error overlay
// is still drawn after a panic.
func (g *hotReloadGame) drawScript(screen *ebiten.Image) {
	defer g.recoverHook("Draw")
	g.current.draw(screen)
}

// Layout returns the script layout, falling back to the compiled layout if
// the script panics.
func (g *hotReloadGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	defer func() {
		if r := recover(); r != nil {
			g.lastErr = fmt.Errorf("Layout panicked: %v", r)
			screenWidth, screenHeight = game.Layout(outsideWidth, outsideHeight)
		}
	}()
	return g.current.layout(outsideWidth, outsideHeight)
}

// recoverHook turns a panic from a script hook into a visible error so a
// broken script never crashes the game.
func (g *hotReloadGame) recoverHook(hook string) {
	if r := recover(); r != nil {
		g.lastErr = fmt.Errorf("%s panicked: %v", hook, r)
	}
}

// load evaluates src and swaps in the resulting hooks. On failure the current
// hooks stay in place and the error is shown on screen.
func (g *hotReloadGame) load(src []byte) {
	h, err := evalHooks(src)
	if err != nil {
		g.lastErr = err
		return
	}
	g.current = h
	g.lastErr = nil
}

// evalHooks evaluates the game script in a fresh interpreter and returns the
// hooks it defines. It recovers from interpreter panics because yaegi raises
// panics, not errors, for some invalid scripts (e.g. unknown identifiers).
func evalHooks(src []byte) (h hooks, err error) {
	defer func() {
		if r := recover(); r != nil {
			h = hooks{}
			err = fmt.Errorf("interpreter panic: %v", r)
		}
	}()

	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return hooks{}, fmt.Errorf("stdlib symbols: %w", err)
	}
	if err := i.Use(symbols.Symbols); err != nil {
		return hooks{}, fmt.Errorf("game symbols: %w", err)
	}
	if _, err := i.Eval(string(src)); err != nil {
		return hooks{}, err
	}

	vu, err := i.Eval("game.Update")
	if err != nil {
		return hooks{}, fmt.Errorf("game.Update: %w", err)
	}
	update, ok := vu.Interface().(func() error)
	if !ok {
		return hooks{}, fmt.Errorf("game.Update must be func() error, got %T", vu.Interface())
	}

	vd, err := i.Eval("game.Draw")
	if err != nil {
		return hooks{}, fmt.Errorf("game.Draw: %w", err)
	}
	draw, ok := vd.Interface().(func(*ebiten.Image))
	if !ok {
		return hooks{}, fmt.Errorf("game.Draw must be func(*ebiten.Image), got %T", vd.Interface())
	}

	vl, err := i.Eval("game.Layout")
	if err != nil {
		return hooks{}, fmt.Errorf("game.Layout: %w", err)
	}
	layout, ok := vl.Interface().(func(int, int) (int, int))
	if !ok {
		return hooks{}, fmt.Errorf("game.Layout must be func(int, int) (int, int), got %T", vl.Interface())
	}

	return hooks{update: update, draw: draw, layout: layout}, nil
}

// watch requests a reload whenever a Go file in the script directory changes.
// Reloading is debounced so multi-step editor saves trigger a single reload,
// and deferred to Update so hooks are only swapped on the game goroutine.
func (g *hotReloadGame) watch() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("hot reload disabled: %v", err)
		return
	}
	if err := w.Add(scriptDir); err != nil {
		log.Printf("hot reload disabled: %v", err)
		_ = w.Close()
		return
	}
	go func() {
		defer func() { _ = w.Close() }()
		var timer *time.Timer
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if filepath.Ext(event.Name) != ".go" {
					continue
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
					continue
				}
				if timer == nil {
					timer = time.AfterFunc(reloadDebounce, func() { g.pendingReload.Store(true) })
				} else {
					timer.Reset(reloadDebounce)
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("hot reload: %v", err)
			}
		}
	}()
}
