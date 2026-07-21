package main

import (
	"embed"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"

	"translator/icons"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// SPEC-add: debug launch option. `--debug`/`-debug` (or the
	// LIBRETRANSLATE_DEBUG env var) enables verbose logging, including every
	// network request to LibreTranslate.
	if debugFlagRequested() {
		_ = os.Setenv("LIBRETRANSLATE_DEBUG", "1")
	}

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "LibreTranslate Translator",
		Width:     960,
		Height:    640,
		MinWidth:  640,
		MinHeight: 460,
		Linux: &linux.Options{
			Icon:        icons.App(512), // window manager / taskbar icon
			ProgramName: "translator",   // WM_CLASS, matched by the .desktop StartupWMClass
			// Explicit Never preserves the default that applies when
			// options.Linux is nil (wails issue 2977) - adding the icon
			// must not silently change the GPU policy.
			WebviewGpuPolicy: linux.WebviewGpuPolicyNever,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		// SPEC §8.2.2: bind the app and the two service structs.
		Bind: []interface{}{
			app,
			app.settings,
			app.translator,
		},
	})
	if err != nil {
		log.Fatalf("wails run error: %v", err)
	}
}

// debugFlagRequested reports whether a --debug / -debug flag was passed.
func debugFlagRequested() bool {
	for _, a := range os.Args[1:] {
		switch strings.ToLower(a) {
		case "--debug", "-debug", "debug":
			return true
		}
	}
	return false
}
