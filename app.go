package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"translator/internal/libretranslate"
	"translator/internal/settings"
)

// App is the Wails application struct (SPEC §8.2.1). It owns the context
// handed to the runtime package and the two bound service instances.
type App struct {
	ctx        context.Context
	settings   *settings.Service
	translator *libretranslate.Service
	tray       *Tray

	// winVisible tracks window visibility for the tray hide/show toggle
	// (wails v2 has no visibility query). Tray actions update it directly;
	// the frontend confirms via "window:visibility" page-visibility events,
	// which also covers minimize and the window close button.
	winVisible atomic.Bool
}

// NewApp constructs the App and its services. If the settings store cannot be
// initialised (e.g. the config directory can't be resolved), the app still
// launches with in-memory defaults rather than crashing; the failure is
// surfaced to the UI via LoadWarning (BUGS #6).
func NewApp() *App {
	settingsSvc, err := settings.NewService()
	if err != nil {
		log.Printf("[app] settings store unavailable (%v); continuing with in-memory defaults", err)
		settingsSvc = settings.NewInMemoryService(
			fmt.Sprintf("Settings could not be loaded (%v); changes won't be saved this session.", err))
	}
	translatorSvc := libretranslate.NewService(settingsSvc)
	a := &App{
		settings:   settingsSvc,
		translator: translatorSvc,
		tray:       NewTray(),
	}
	a.winVisible.Store(true) // the window starts visible
	return a
}

// startup is the Wails OnStartup hook; the context is stored so the runtime
// package can be reached (SPEC §8.2.5).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if settings.DebugEnabled(a.settings.GetSettings()) {
		log.Printf("[app] startup; debug=on; %s", a.settings.Snapshot())
	}

	// The frontend reports page visibility so the tray toggle stays correct
	// even when the window is hidden or minimized without the tray's help.
	runtime.EventsOn(ctx, "window:visibility", func(args ...interface{}) {
		if len(args) > 0 {
			if visible, ok := args[0].(bool); ok {
				if settings.DebugEnabled(a.settings.GetSettings()) {
					log.Printf("[app] window visibility reported by frontend: %v", visible)
				}
				a.winVisible.Store(visible)
				a.tray.SetVisible(visible)
			}
		}
	})

	a.tray.Start(a.toggleWindow, func() {
		log.Printf("[tray] quit requested from tray menu")
		runtime.Quit(a.ctx)
	})
}

// toggleWindow is the tray left-click / menu hide-show action.
func (a *App) toggleWindow() {
	if a.winVisible.Load() {
		if settings.DebugEnabled(a.settings.GetSettings()) {
			log.Printf("[app] tray toggle: hiding window")
		}
		runtime.WindowHide(a.ctx)
		a.winVisible.Store(false)
		a.tray.SetVisible(false)
	} else {
		if settings.DebugEnabled(a.settings.GetSettings()) {
			log.Printf("[app] tray toggle: showing window")
		}
		runtime.WindowShow(a.ctx)
		a.winVisible.Store(true)
		a.tray.SetVisible(true)
	}
}

// shutdown is the Wails OnShutdown hook.
func (a *App) shutdown(_ context.Context) {
	a.tray.Stop()
}
