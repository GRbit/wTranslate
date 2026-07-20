package main

import (
	"context"
	"fmt"
	"log"

	"translator/internal/libretranslate"
	"translator/internal/settings"
)

// App is the Wails application struct (SPEC §8.2.1). It owns the context
// handed to the runtime package and the two bound service instances.
type App struct {
	ctx        context.Context
	settings   *settings.Service
	translator *libretranslate.Service
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
	return &App{
		settings:   settingsSvc,
		translator: translatorSvc,
	}
}

// startup is the Wails OnStartup hook; the context is stored so the runtime
// package can be reached (SPEC §8.2.5).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if settings.DebugEnabled(a.settings.GetSettings()) {
		log.Printf("[app] startup; debug=on; %s", a.settings.Snapshot())
	}
}

// shutdown is the Wails OnShutdown hook.
func (a *App) shutdown(_ context.Context) {}
