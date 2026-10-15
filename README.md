# ebiten-template

🐹 [Ebiten/Ebitengine](https://ebitengine.org/) template for making 2D games with Go. Logo from [gophers](https://github.com/egonelbre/gophers).

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

## Run

Execute the program:

```sh
go run .
```

## Lint

Lint the code:

```sh
go tool golangci-lint run
```

## Web

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
