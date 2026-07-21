package main

import (
	"embed"
	"fmt"
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

const usageText = `LibreTranslate Translator - desktop client for LibreTranslate.

Usage:
  translator [options]

Options:
  --translate-clipboard  Paste the clipboard into the source box and translate
                         it. Works both on a fresh start and when the app is
                         already running.
  --debug                Verbose logging to stderr (same as
                         LIBRETRANSLATE_DEBUG=1).
  -h, --help             Show this help and exit.

Single instance:
  Only one copy of the app runs. Launching translator again brings the
  existing window to the front instead of starting a second instance.

Global hotkey:
  Bind a keyboard shortcut in your desktop environment to:
      translator --translate-clipboard
  e.g. XFCE: Settings -> Keyboard -> Application Shortcuts -> Add.
  Copy some text, press the shortcut, and the translation window pops up
  with the result.
`

func main() {
	opts := parseArgs(os.Args[1:])

	if opts.help {
		fmt.Print(usageText)
		return
	}

	// SPEC-add: debug launch option. `--debug`/`-debug` (or the
	// LIBRETRANSLATE_DEBUG env var) enables verbose logging, including every
	// network request to LibreTranslate.
	if opts.debug {
		_ = os.Setenv("LIBRETRANSLATE_DEBUG", "1")
	}

	app := NewApp()
	app.launchTranslateClipboard.Store(opts.translateClipboard)

	err := wails.Run(&options.App{
		Title:     "LibreTranslate Translator",
		Width:     960,
		Height:    640,
		MinWidth:          640,
		MinHeight:         460,
		HideWindowOnClose: true, // close button hides to tray; quit via menu, tray, or Ctrl+Q
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
		Menu:             app.appMenu(),
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "com.grbit.translator",
			OnSecondInstanceLaunch: app.onSecondInstance,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
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

// cliOptions holds the flags parsed from the command line. Unknown arguments
// are ignored, as before.
type cliOptions struct {
	help               bool
	debug              bool
	translateClipboard bool
}

// parseArgs scans command-line arguments (also used for the arguments a
// second instance was launched with). The bare "debug" spelling is kept for
// backward compatibility.
func parseArgs(args []string) cliOptions {
	var o cliOptions
	for _, a := range args {
		switch strings.ToLower(a) {
		case "--help", "-h", "-help":
			o.help = true
		case "--debug", "-debug", "debug":
			o.debug = true
		case "--translate-clipboard", "-translate-clipboard":
			o.translateClipboard = true
		}
	}
	return o
}
