# LibreTranslate Translator

Cross-platform desktop translator built with **Go + Wails v2.12** and the
**LibreTranslate** API. See [`SPEC.md`](./SPEC.md) for the authoritative
specification and [`DECISIONS.md`](./DECISIONS.md) for locked-in design choices.

## Stack
- Backend: Go 1.23+, Wails v2.12. `internal/settings` (config store) and
  `internal/libretranslate` (API client) are pure-Go and unit-tested.
- Frontend: Svelte 4 + Vite 5 + TypeScript (Wails `svelte-ts` template).
- Provider: LibreTranslate (`GET /languages`, `POST /translate`).

## Requirements
- Go 1.23+, Node 18+, Wails CLI v2.12 (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`)
- Native GUI build deps: on Linux `libwebkit2gtk-4.1-dev libgtk-3-dev` + gcc
  (`webkit2gtk-4.0` on older distros); on Windows `webview2`; on macOS Xcode.

## Development
```bash
wails dev      # hot-reload dev app
```

## Production build
```bash
wails build    # outputs build/bin/translator
```

## Regenerate frontend bindings
After changing bound Go methods/types:
```bash
wails generate module
# On a host without CGO/GTK or a read-only HOME:
# CGO_ENABLED=0 LIBRETRANSLATE_CONFIG_DIR=/tmp/ltcfg wails generate module
```

## Tests & checks
```bash
go test ./internal/...            # backend unit tests (httptest)
go vet ./...                      # incl. main package against Wails API
cd frontend && npm run check      # svelte-check (0 errors expected)
cd frontend && npm run build      # vite production build
```

## Debug logging
Enable verbose logs (every HTTP request to LibreTranslate, settings load/save,
startup) via any of:
```bash
wails dev -- -debug                # CLI flag (or run the binary with --debug)
LIBRETRANSLATE_DEBUG=1 wails dev   # env var
```
or toggle **Settings → Developer → Debug logging**. The API key is never
logged. Logs go to stderr (visible in the terminal / `wails dev` console).

## Project layout
```
main.go, app.go        Wails entry + App lifecycle (binds 3 structs)
wails.json             Wails project config
internal/settings/     Settings store (OS config dir, JSON, atomic writes)
internal/libretranslate/  LibreTranslate client (languages/translate, timeouts, errors)
frontend/src/lib/      store, api, translate/swap logic, init, types
frontend/src/lib/components/  Translator, SettingsModal, Toast
frontend/wailsjs/go/   generated Wails bindings (do not edit)
```

## Settings
Stored at `<os.UserConfigDir>/LibreTranslateTranslator/settings.json`
(override with `LIBRETRANSLATE_CONFIG_DIR`). Defaults: Base URL
`https://libretranslate.com`, empty API key, Default-to-Auto on, Live
Translation off.
