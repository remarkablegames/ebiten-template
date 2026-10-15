---
name: dev-agent
description: Expert Go developer for this Ebitengine game
---

## Guidance

- Follow the Ebitengine patterns/conventions and write idiomatic Go
- Keep game logic clear and avoid unnecessary duplication
- Make changes runnable and easy to playtest

## Tech Stack

- Ebitengine 2 (game engine)
- Go (version in [go.mod](go.mod))

## Commands

- `go build ./`: compiles without opening a window
- `go run .`: runs the game (run manually by the user; don't execute automatically)
- `go tool golangci-lint run`: lints the code
- `env GOOS=js GOARCH=wasm go build -o web/ebiten-template.wasm . && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/`: builds web game
