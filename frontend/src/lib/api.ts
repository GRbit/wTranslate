import { GetSettings, UpdateSettings, LoadWarning } from '../../wailsjs/go/settings/Service.js';
import { GetLanguages, GetFrontendSettings, Translate } from '../../wailsjs/go/libretranslate/Service.js';

export { GetSettings, UpdateSettings, LoadWarning, GetLanguages, GetFrontendSettings, Translate };

/**
 * Normalise a value rejected by a Wails binding call into a human-readable
 * string. Wails may reject with an Error, a plain string, or an object.
 */
export function errorMessage(e: unknown): string {
  if (e == null) return 'Unknown error';
  if (typeof e === 'string') return e;
  if (e instanceof Error) return e.message;
  if (typeof e === 'object' && 'message' in e) {
    return String((e as { message: unknown }).message);
  }
  return String(e);
}
