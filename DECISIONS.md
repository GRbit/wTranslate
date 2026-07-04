# DECISIONS — LibreTranslate Translator

Locked-in implementation decisions delegated to the agent by `SPEC.md`.
These do **not** modify `SPEC.md` (immutable per §11.2); they resolve the
explicit choices the spec leaves to the agent.

**Date:** 2026-07-04

## D1 — Frontend template (resolves §8.1.2)
- Template: **`svelte-ts`** (Svelte + Vite + TypeScript).
- Rationale: spec recommends `svelte-ts` or `react-ts`; Svelte stores give a
  clean single state layer (§8.3.3). Pinned in `frontend/package.json`.
- Versions (modernised for Node 24 toolchain): Svelte 4, Vite 5, TypeScript 5,
  svelte-check 3.

## D2 — Backend package layout (resolves §8.2, §10.3)
- `internal/settings`  — pure-Go, no Wails import. `Service` + `Settings`.
- `internal/libretranslate` — pure-Go, no Wails import. `Service` + data types.
- `main` package (`main.go`, `app.go`) — Wails lifecycle; binds the three
  instances (`app`, `settingsService`, `translatorService`) per §8.2.2.
- Services live outside `main` so they are unit-testable with `httptest`
  without CGO/GTK (this build host lacks webkit2gtk/gcc).

## D3 — Translate wire format (resolves §4.2.1)
- Unified on **JSON** body (`Content-Type: application/json`) for `/translate`.
- `/languages` is a plain `GET`.

## D4 — Settings storage (resolves §9.2)
- Simple self-contained JSON wrapper (option 2 of §9.2), no third-party store.
- Path: `os.UserConfigDir()/LibreTranslateTranslator/settings.json`.
- Atomic write (temp file + rename). Missing file → defaults written.

## D5 — Defaults & limits (resolves §3.2, §6.1, §7.1)
- Base URL default: `https://libretranslate.com`; API key default: empty.
- `Default to Auto`: **on** by default; `Live Translation`: **off** by default.
- `Translation Shortcut` default: `ctrl_enter`.
- HTTP per-request timeout: **10s** (context.WithTimeout) per §7.1.
- Char limit: **2000**.
- Live translation: debounce **600ms**, min interval (throttle) **1500ms**.

## D6 — Error surface to UI (resolves §7.3)
- Backend returns descriptive Go `error` strings (Wails rejects the JS promise
  with that message). Frontend shows them in a non-blocking toast bar.
- On translate error, language state is unchanged and the output field is
  cleared (consistent behaviour per §7.4).

## D7 — Frontend ↔ Go bindings (resolves §8.3)
- Bindings under `frontend/wailsjs/go/...` are produced by **`wails generate
  module`** (Wails v2.12). On a host without CGO/GTK build deps they can be
  (re)generated with:
  `CGO_ENABLED=0 LIBRETRANSLATE_CONFIG_DIR=/tmp/ltcfg wails generate module`
  (the env var lets the App constructor's settings init succeed in a
  read-only HOME; on a normal dev machine it is unnecessary).
- Wails namespaces bound structs by **package name**, so the generated layout
  is `frontend/wailsjs/go/settings/Service.js` and
  `.../go/libretranslate/Service.js`, with `models.ts` exposing namespaced
  classes (`settings.Settings`, `libretranslate.Language`, …). The frontend
  consumes these via flat type aliases in `frontend/src/lib/types.ts`.
- `context.Context` is **not** used in bound method signatures: Wails v2.12
  does not strip it, and SPEC §8.2.3 specifies `Translate(req)` / `GetLanguages()`
  with no context param. Per-request timeouts still use
  `context.WithTimeout(context.Background(), 10s)` internally (SPEC §7.1).
- The full GUI binary (`wails build`) still requires CGO + webkit2gtk/gtk
  headers, which this build host lacks.

## D8 — Debug logging option (user-requested addition beyond SPEC)
- Verbose logging is enabled by any of: the `Debug` setting (persisted,
  toggle in Settings → Developer), the `--debug` / `-debug` CLI flag, or the
  `LIBRETRANSLATE_DEBUG=1` env var. `--debug` sets the env var at startup, so
  `settings.EnvDebug()` is the single source of truth.
- Logs go to stderr via `log` (visible in the terminal under `wails dev` or
  when launched from a console). They cover: app startup, settings load/save,
  and every LibreTranslate HTTP request (method, URL, source/target, q length,
  api_key presence only — never the key, status, duration, response size,
  detected language).
- The API key is never logged; only `<set>` / `<none>`.

## D9 — Settings modal input editability (bug fix)
- The Base URL / API key inputs were reported as non-editable. The modal now
  uses the canonical nested pattern: `.modal` is a child of the `.overlay`
  backdrop (flex-centered), so inputs sit above the backdrop without relying
  on z-index. Backdrop-close uses a `target === currentTarget` check instead
  of `stopPropagation` on the modal (avoids a11y warnings). Inputs use
  explicit `type="text"`, `id`/`for`, and a global `user-select: text` guard.
