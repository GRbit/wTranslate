import { writable, derived } from 'svelte/store';
import type { Settings, Language, DetectedLanguage } from './types';

/** Pseudo language code used for auto-detection (SPEC §4.1.3). */
export const AUTO = 'auto';

export const SHORTCUT_CTRL_ENTER = 'ctrl_enter';
export const SHORTCUT_ENTER = 'enter';

/**
 * Instance per-request character limit (SPEC §6.1). Discovered at runtime from
 * the LibreTranslate instance's /frontend/settings; `null` means the instance
 * imposes no limit (so the UI shows only a running count). Until the first
 * fetch resolves it stays `null`: the server still enforces its own bound.
 */
export const charLimit = writable<number | null>(null);

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
    autoCopy: false,
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
/** Which window-menu modal is open: the Help or Credits one (null = none). */
export const infoModal = writable<'help' | 'credits' | null>(null);

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
export const overLimit = derived(
  [sourceText, charLimit],
  ([$s, $limit]) => $limit != null && $s.length > $limit,
);

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
