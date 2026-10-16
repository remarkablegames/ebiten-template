<p align="center">
  <img src="assets/logo.svg" alt="ebiten-template" width="200">
</p>

# ebiten-template

🐹 [Ebiten/Ebitengine](https://ebitengine.org/) template for making 2D games with Go.

Play in your browser:

- [remarkablegames](https://remarkablegames.org/ebiten-template/)

Or download for desktop:

- [Windows](https://github.com/remarkablegames/ebiten-template/releases/latest/download/windows.zip)
- [macOS](https://github.com/remarkablegames/ebiten-template/releases/latest/download/macos.zip)
- [Linux](https://github.com/remarkablegames/ebiten-template/releases/latest/download/linux.zip)

## Credits

### Art

- [Gophers](https://github.com/egonelbre/gophers)

## Prerequisites

Install [Go](https://go.dev/doc/install):

```sh
brew install go
```

## Install

Clone the repository:

```sh
git clone https://github.com/remarkablegames/ebiten-template.git
cd ebiten-template
```

Rename the game:

```sh
git grep -l ebiten-template | xargs sed -i '' -e 's/ebiten-template/my-game/g'
```

Update the files:

- [ ] `README.md`
- [ ] [`assets/icon.icns`](https://remarkablemark.org/ico-icns-generator/)
- [ ] [`assets/icon.ico`](https://remarkablemark.org/ico-icns-generator/)
- [ ] `assets/logo.svg`
- [ ] `web/index.html`
- [ ] `web/manifest.webmanifest`

## Scripts

### Run

Open the game with hot reload:

```sh
go run .
```

### Lint

Lint the code:

```sh
go tool golangci-lint run
```

### Test

Run the tests:

```sh
go test ./...
```

## Development

### Hot Reload

On desktop, the game reloads its code while it runs: edit a file in [`game/`](game) and save, and the window picks up the change without restarting.

- Every Go file in `game/` (except tests) is loaded as one script.
- Reloads trigger automatically when a file in `game/` changes (debounced), and apply at the start of the next frame.
- If the script fails to compile, the error is shown on screen and the last working version keeps running. Panics in the script are caught the same way.
- A reload re-evaluates the script from scratch, so the game state resets.

The script can import any package exposed in [`symbols/`](symbols). To expose another package, add a `go:generate` directive in [`symbols/symbols.go`](symbols/symbols.go) and regenerate:

```sh
go generate ./symbols
```

## Build

### Desktop

Build a native binary for production:

```sh
go build -tags release .
```

### Web

Build the web game:

```sh
env GOOS=js GOARCH=wasm go build -o web/ebiten-template.wasm .
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
```

Serve the `web` directory with a static server:

```sh
python3 -m http.server -d web
```

Then open http://localhost:8000/ in your browser:

```sh
open http://localhost:8000/
```

Alternatively, use [wasmserve](https://github.com/hajimehoshi/wasmserve):

```sh
go run github.com/hajimehoshi/wasmserve@latest .
```

## License

[MIT](LICENSE)
