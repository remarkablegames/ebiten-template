package game

const (
	// ScreenWidth and ScreenHeight are the game's internal render buffer in
	// logical pixels. Layout returns them every frame, so hot reload applies
	// edits immediately.
	ScreenWidth  = 320
	ScreenHeight = 240

	// WindowWidth and WindowHeight size the OS window in device-independent
	// pixels. main.go reads them at startup, so edits take effect on the
	// next run.
	WindowWidth  = 640
	WindowHeight = 480
)
