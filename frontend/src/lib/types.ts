// Type aliases for the Wails-generated models (frontend/wailsjs/go/models.ts).
// The generated file exposes namespaced classes (e.g. `settings.Settings`),
// which would clash with the `settings` Svelte store name. Re-exporting flat
// aliases here keeps the rest of the frontend decoupled from the generated
// namespace layout. This import is type-only and erased at build time.
import type * as models from '../../wailsjs/go/models.js';

export type Settings = models.settings.Settings;
export type Language = models.libretranslate.Language;
export type DetectedLanguage = models.libretranslate.DetectedLanguage;
export type TranslateRequest = models.libretranslate.TranslateRequest;
export type TranslateResponse = models.libretranslate.TranslateResponse;
