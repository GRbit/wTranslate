import { writable, derived } from 'svelte/store';
import type { Settings, Language, DetectedLanguage } from './types';

/** Pseudo language code used for auto-detection (SPEC §4.1.3). */
export const AUTO = 'auto';

export const SHORTCUT_CTRL_ENTER = 'ctrl_enter';
export const SHORTCUT_ENTER = 'enter';

/** Hard character limit on the source field (SPEC §6.1). */
export const CHAR_LIMIT = 2000;

export function defaultSettings(): Settings {
  return {
    baseUrl: 'https://libretranslate.com',
    apiKey: '',
    liveTranslation: false,
    shortcut: SHORTCUT_CTRL_ENTER,
    defaultToAuto: true,
    lastSourceLang: 'en',
    lastTargetLang: 'en',
    debug: false,
  };
}

// --- Central application state (SPEC §8.3.3: single state layer) ---
export const settings = writable<Settings>(defaultSettings());
export const languages = writable<Language[]>([]);
export const sourceLang = writable<string>(AUTO);
export const targetLang = writable<string>('en');
export const sourceText = writable<string>('');
export const translatedText = writable<string>('');
export const detected = writable<DetectedLanguage | null>(null);
export const isTranslating = writable<boolean>(false);
export const settingsOpen = writable<boolean>(false);

// --- Toast / error bar (SPEC §7.3) ---
export type ToastKind = 'error' | 'info';
export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}
export const toast = writable<Toast | null>(null);

// --- Derived UI state ---
export const charCount = derived(sourceText, ($s) => $s.length);
export const overLimit = derived(sourceText, ($s) => $s.length > CHAR_LIMIT);

/** Swap is disabled only in Auto mode before a detected language exists (SPEC §5.1). */
export const swapDisabled = derived(
  [sourceLang, detected],
  ([$source, $detected]) => $source === AUTO && !$detected,
);

let toastSeq = 0;

export function showToast(kind: ToastKind, message: string, ttl = 5000): void {
  const id = ++toastSeq;
  toast.set({ id, kind, message });
  if (ttl > 0) {
    setTimeout(() => {
      toast.update((t) => (t && t.id === id ? null : t));
    }, ttl);
  }
}

export function clearToast(): void {
  toast.set(null);
}
