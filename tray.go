package main

import (
	"log"
	"sync/atomic"

	"fyne.io/systray"

	"translator/icons"
)

// Tray drives the system tray icon. The fyne.io/systray Linux backend is a
// pure-Go D-Bus StatusNotifierItem, so it runs on its own goroutine next to
// the Wails GTK main loop without conflict. The tray icon is the app icon
// (icons.App) at StatusNotifierItem pixmap size.
type Tray struct {
	ready      atomic.Bool
	toggleItem *systray.MenuItem // set inside systray.Run's onReady callback
}

func NewTray() *Tray { return &Tray{} }

// Start launches the tray loop. onToggle runs on left-click (SNI Activate)
// and on the hide/show menu item; onQuit runs on the Quit item. Menu-item
// callbacks arrive on a tray-owned goroutine, so everything they touch must
// be thread-safe.
func (t *Tray) Start(onToggle, onQuit func()) {
	// Must be set before Run: the SNI ItemIsMenu property is derived from it
	// at registration time, and ItemIsMenu=false is what makes the host send
	// left clicks to Activate instead of opening the menu.
	systray.SetOnTapped(onToggle)
	go systray.Run(func() {
		systray.SetIcon(icons.App(32))
		systray.SetTooltip("LibreTranslate Translator")

		// The window starts visible, so the toggle item starts as "Hide".
		t.toggleItem = systray.AddMenuItem("Hide window", "Hide or show the translator window")
		systray.AddSeparator()
		quitItem := systray.AddMenuItem("Quit", "Exit the app")

		go func() {
			for {
				select {
				case <-t.toggleItem.ClickedCh:
					onToggle()
				case <-quitItem.ClickedCh:
					onQuit()
				}
			}
		}()

		t.ready.Store(true)
		log.Printf("[tray] system tray ready")
	}, nil)
}

// Stop tears down the tray.
func (t *Tray) Stop() {
	if t.ready.Load() {
		systray.Quit()
	}
}

// SetVisible updates the hide/show menu item label to match the window
// visibility. Calls before the tray is ready are dropped: the item is
// created with the visible-window label, matching startup state.
func (t *Tray) SetVisible(visible bool) {
	if !t.ready.Load() {
		return
	}
	if visible {
		t.toggleItem.SetTitle("Hide window")
	} else {
		t.toggleItem.SetTitle("Show window")
	}
}
