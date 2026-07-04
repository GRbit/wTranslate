import { get } from 'svelte/store';
import type { Settings } from './types';
import * as st from './store';
import { Translate, SaveSettings, errorMessage } from './api';

// Live translation timing (SPEC §3.2.1: debounce 500–800ms, throttle 1–2s).
const DEBOUNCE_MS = 600;
const THROTTLE_MS = 1500;

let debounceTimer: ReturnType<typeof setTimeout> | null = null;
let throttleLast = 0;

/** Translate the current source text immediately (SPEC §2.1, §4.2). */
export async function runTranslate(): Promise<void> {
  const text = get(st.sourceText);
  if (!text.trim()) {
    st.translatedText.set('');
    st.detected.set(null);
    return;
  }
  const source = get(st.sourceLang);
  const target = get(st.targetLang);
  st.isTranslating.set(true);
  st.clearToast();
  try {
    const res = await Translate({ q: text, source, target });
    st.translatedText.set(res.translatedText ?? '');
    st.detected.set(res.detectedLanguage ?? null);
  } catch (e) {
    // SPEC §7.4: keep language state, clear output, show error.
    st.showToast('error', errorMessage(e));
    st.translatedText.set('');
    st.detected.set(null);
  } finally {
    st.isTranslating.set(false);
  }
}

/** Debounced + throttled translation for Live Translation mode (SPEC §3.2.1). */
export function scheduleLive(): void {
  if (!get(st.settings).liveTranslation) return;
  if (debounceTimer) clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    const now = Date.now();
    const wait = THROTTLE_MS - (now - throttleLast);
    if (wait > 0) {
      setTimeout(() => {
        throttleLast = Date.now();
        void runTranslate();
      }, wait);
    } else {
      throttleLast = now;
      void runTranslate();
    }
  }, DEBOUNCE_MS);
}

/**
 * Swap source and target (SPEC §5.3). In Auto mode with a detected language,
 * the detected language becomes the new target and Auto is dropped.
 */
export function swap(): void {
  const src = get(st.sourceLang);
  const srcText = get(st.sourceText);
  const tgtText = get(st.translatedText);

  if (src === st.AUTO) {
    const det = get(st.detected);
    if (!det) return; // disabled; no-op guard
    const prevTarget = get(st.targetLang);
    st.sourceLang.set(prevTarget);
    st.targetLang.set(det.language);
    st.sourceText.set(tgtText.slice(0, st.CHAR_LIMIT));
    st.translatedText.set(srcText);
    st.detected.set(null);
  } else {
    const tgt = get(st.targetLang);
    st.sourceLang.set(tgt);
    st.targetLang.set(src);
    st.sourceText.set(tgtText.slice(0, st.CHAR_LIMIT));
    st.translatedText.set(srcText);
  }
  void persistSettingsDebounced();
}

/** Clear input, output and detected language (SPEC §6.3). */
export function clearAll(): void {
  st.sourceText.set('');
  st.translatedText.set('');
  st.detected.set(null);
}

/** Paste buffer into input field */
export async function pasteToSource(): Promise<void> {
    try {
        const { ClipboardGetText } = await import('../../wailsjs/runtime/runtime');
        const text = await ClipboardGetText();

        if (text) {
            st.sourceText.set(text);
            scheduleLive();
        }
    } catch (e) {
        st.showToast('error', 'Could not paste from clipboard');
    }
}


/** Copy the whole translation to the clipboard (SPEC §6.2). */
export async function copyTranslation(): Promise<void> {
  const text = get(st.translatedText);
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    st.showToast('info', 'Translation copied', 2500);
  } catch {
    st.showToast('error', 'Could not copy to clipboard');
  }
}

let persistTimer: ReturnType<typeof setTimeout> | null = null;

/** Debounced persistence of the current language selection (SPEC §3.2.3). */
export function persistSettingsDebounced(): void {
  if (persistTimer) clearTimeout(persistTimer);
  persistTimer = setTimeout(() => {
    void saveCurrentSettings();
  }, 400);
}

/** Write current settings + selected languages to disk via the backend. */
export async function saveCurrentSettings(next?: Settings): Promise<void> {
  const cur = next ?? get(st.settings);
  const merged: Settings = {
    ...cur,
    lastSourceLang: get(st.sourceLang),
    lastTargetLang: get(st.targetLang),
  };
  try {
    await SaveSettings(merged);
    st.settings.set(merged);
  } catch (e) {
    st.showToast('error', 'Could not save settings: ' + errorMessage(e));
  }
}
