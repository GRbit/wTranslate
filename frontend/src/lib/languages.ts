import { get } from 'svelte/store';
import { languages } from './store';

/** Human-readable name for a language code, falling back to uppercased code. */
export function langName(code: string): string {
  const found = get(languages).find((l) => l.code === code);
  return found ? found.name : code.toUpperCase();
}
