package main

import (
	"context"
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

// NewApp constructs the App and its services. A failure to initialise the
// settings store is fatal: there is no useful state without it.
func NewApp() *App {
	settingsSvc, err := settings.NewService()
	if err != nil {
		log.Fatalf("failed to initialise settings: %v", err)
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
