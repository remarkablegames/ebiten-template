//go:build !js && !release

// Package runner supplies the game to ebiten.RunGame.
//
// For development builds, this file interprets package game with yaegi and
// hot-reloads it: it watches game/ and swaps the Update/Draw/Layout hooks on
// save. Every Go file in game/ (except tests) is part of the script, in any
// filename order. Failures are shown on screen; the last working version
// keeps running.
//
// Release (-tags release) and web (GOOS=js) builds use production.go instead,
// which runs the compiled game package.
package runner

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
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

// scriptFile is one file of the interpreted game script.
type scriptFile struct {
	path string // path as shown in errors, e.g. "game/config.go"
	src  []byte
}

// readScriptFiles reads every Go file in dir except test files, in directory
// (name-sorted) order. Eval order does not matter: evalHooks retries files
// that reference declarations loaded later.
func readScriptFiles(dir string) ([]scriptFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", dir, err)
	}
	var files []scriptFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", path, err)
		}
		files = append(files, scriptFile{path: path, src: src})
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no Go files in %s", dir)
	}
	return files, nil
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
// live. If the game directory cannot be read (the binary is running from
// another working directory), the compiled game runs instead.
func NewGame() ebiten.Game {
	g := &hotReloadGame{current: compiledHooks()}
	files, err := readScriptFiles(scriptDir)
	if err != nil {
		log.Printf("hot reload: %v, running compiled game", err)
	} else {
		g.load(files)
	}
	g.watch()
	return g
}

// Update consumes a pending reload, then runs one tick of game logic.
// Script failures are recorded for display instead of stopping the game.
func (g *hotReloadGame) Update() error {
	if g.pendingReload.Swap(false) {
		files, err := readScriptFiles(scriptDir)
		if err != nil {
			g.lastErr = err
		} else {
			g.load(files)
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

// load evaluates the script files and swaps in the resulting hooks. On
// failure the current hooks stay in place and the error is shown on screen.
func (g *hotReloadGame) load(files []scriptFile) {
	h, err := evalHooks(files)
	if err != nil {
		g.lastErr = err
		return
	}
	g.current = h
	g.lastErr = nil
}

// hoistImports parses the script files and returns an import preamble plus
// each file without its imports. Imports are hoisted because yaegi binds
// them into the package scope, so two files importing the same path would
// collide. Import blocks are blanked but keep their newlines so errors still
// report original line numbers.
func hoistImports(files []scriptFile) (preamble string, sources []scriptFile, err error) {
	fset := token.NewFileSet()
	var (
		pkg   string
		specs []string
	)
	seen := make(map[string]bool)
	sources = make([]scriptFile, 0, len(files))
	for _, f := range files {
		pf, err := parser.ParseFile(fset, f.path, f.src, parser.SkipObjectResolution)
		if err != nil {
			return "", nil, fmt.Errorf("%s: %w", f.path, err)
		}
		if pkg == "" {
			pkg = pf.Name.Name
		} else if pf.Name.Name != pkg {
			return "", nil, fmt.Errorf("%s: package %s, want package %s", f.path, pf.Name.Name, pkg)
		}
		var out strings.Builder
		last := 0
		for _, decl := range pf.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.IMPORT {
				continue
			}
			start := fset.File(g.Pos()).Offset(g.Pos())
			end := fset.File(g.End()).Offset(g.End())
			out.Write(f.src[last:start])
			out.WriteString(strings.Repeat("\n", strings.Count(string(f.src[start:end]), "\n")))
			last = end
			for _, spec := range g.Specs {
				is, ok := spec.(*ast.ImportSpec)
				if !ok {
					continue
				}
				s := is.Path.Value
				if is.Name != nil {
					s = is.Name.Name + " " + s
				}
				if !seen[s] {
					seen[s] = true
					specs = append(specs, s)
				}
			}
		}
		out.Write(f.src[last:])
		sources = append(sources, scriptFile{path: f.path, src: []byte(out.String())})
	}
	preamble = "package " + pkg + "\nimport (\n" + strings.Join(specs, "\n") + "\n)\n"
	return preamble, sources, nil
}

// evalHooks evaluates the game script and returns the hooks it defines.
// Imports are hoisted into one preamble (see hoistImports); files are then
// evaluated in rounds, retrying failures after the rest load, so filename
// order does not matter.
//
// Each round uses a NEW interpreter: retrying in one is unsafe, because an
// undefined function call panics yaegi mid-file and corrupts its state, so
// even a later successful load yields hooks that panic. Panics are recovered
// because yaegi panics rather than errors on some invalid scripts.
func evalHooks(files []scriptFile) (h hooks, err error) {
	defer func() {
		if r := recover(); r != nil {
			h = hooks{}
			err = fmt.Errorf("interpreter panic: %v", r)
		}
	}()

	preamble, sources, err := hoistImports(files)
	if err != nil {
		return hooks{}, err
	}

	var failures []error
	pending := sources
	for range len(sources) + 1 {
		i := interp.New(interp.Options{})
		if err := i.Use(stdlib.Symbols); err != nil {
			return hooks{}, fmt.Errorf("stdlib symbols: %w", err)
		}
		if err := i.Use(symbols.Symbols); err != nil {
			return hooks{}, fmt.Errorf("game symbols: %w", err)
		}
		if _, err := i.Eval(preamble); err != nil {
			return hooks{}, fmt.Errorf("imports: %w", err)
		}

		var (
			ok        []scriptFile
			next      []scriptFile
			loadedAny bool
		)
		failures = failures[:0]
		for _, f := range pending {
			if err := evalFile(i, f); err != nil {
				next = append(next, f)
				failures = append(failures, fmt.Errorf("%s: %w", f.path, err))
			} else {
				ok = append(ok, f)
				loadedAny = true
			}
		}
		if len(next) == 0 {
			return extractHooks(i)
		}
		if !loadedAny {
			// Nothing loaded this round: these are real errors (syntax
			// mistakes, unknown identifiers), not ordering problems.
			return hooks{}, errors.Join(failures...)
		}
		// Retry the failed files after the ones that loaded, in a fresh
		// interpreter.
		pending = append(ok, next...)
	}
	return hooks{}, errors.Join(failures...)
}

// extractHooks returns the game callbacks defined by an evaluated script.
func extractHooks(i *interp.Interpreter) (h hooks, err error) {
	defer func() {
		if r := recover(); r != nil {
			h = hooks{}
			err = fmt.Errorf("interpreter panic: %v", r)
		}
	}()

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

// evalFile evaluates one script file, turning interpreter panics into errors
// so a single broken file is reported with its path instead of aborting the
// whole load.
func evalFile(i *interp.Interpreter, f scriptFile) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("interpreter panic: %v", r)
		}
	}()
	_, err = i.Eval(string(f.src))
	return err
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
