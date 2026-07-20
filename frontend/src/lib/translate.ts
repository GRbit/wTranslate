import { get } from 'svelte/store';
import type { Settings } from './types';
import * as st from './store';
import { Translate, UpdateSettings, errorMessage } from './api';

// Live translation timing (SPEC §3.2.1): debounce 600ms after the last edit,
// then at most one request per THROTTLE_MS.
const DEBOUNCE_MS = 600;
const THROTTLE_MS = 5000;

let debounceTimer: ReturnType<typeof setTimeout> | null = null;
let throttleTimer: ReturnType<typeof setTimeout> | null = null;
let throttleLast = 0;

// Request generation counter. Every runTranslate bumps it; a response is only
// applied when its generation is still current, so a stale in-flight request
// can never overwrite newer state. clearAll/swap bump it too, invalidating
// whatever is in flight.
let generation = 0;

// Cache of the last successfully translated (text, source, target) so
// re-submitting an identical request is a no-op.
let lastAttempt = {
    text: '',
    source: '',
    target: ''
};

function resetAttemptCache(): void {
  lastAttempt = { text: '', source: '', target: '' };
}

/** Translate the current source text immediately (SPEC §2.1, §4.2). */
export async function runTranslate(manual: boolean = false): Promise<void> {
  const text = get(st.sourceText);
  const source = get(st.sourceLang);
  const target = get(st.targetLang);

  const isRedundant =
    text === lastAttempt.text &&
    source === lastAttempt.source &&
    target === lastAttempt.target;

  if (!text.trim()) {
    generation++; // drop any in-flight result
    st.translatedText.set('');
    st.detected.set(null);
    resetAttemptCache();
    return;
  }

  if (isRedundant) {
    // If it's a manual call, respect auto-copy option
    if (manual && get(st.settings).autoCopy) {
        void copyTranslation();
    }
    return; // Exit with no request
  }

  // The instance would reject this with an HTTP 400; fail fast instead of
  // burning a round-trip. Only nag on manual submits (live mode retries).
  if (get(st.overLimit)) {
    if (manual) {
      st.showToast('error', `Text exceeds this instance's ${get(st.charLimit)}-character limit.`);
    }
    return;
  }

  const gen = ++generation;
  st.isTranslating.set(true);
  st.clearToast();
  try {
    const res = await Translate({ q: text, source, target });
    if (gen !== generation) return; // superseded while in flight
    st.translatedText.set(res.translatedText ?? '');
    st.detected.set(res.detectedLanguage ?? null);
    lastAttempt = { text, source, target };
    if (manual && get(st.settings).autoCopy && res.translatedText) {
        void copyTranslation();
    }
  } catch (e) {
    if (gen !== generation) return; // superseded; the newer call owns the UI
    // SPEC §7.4: keep language state, clear output, show error.
    st.showToast('error', errorMessage(e));
    st.translatedText.set('');
    st.detected.set(null);
    resetAttemptCache(); // drop cache to allow retry after error
  } finally {
    if (gen === generation) st.isTranslating.set(false);
  }
}

/** Debounced + throttled translation for Live Translation mode (SPEC §3.2.1). */
export function scheduleLive(): void {
  if (!get(st.settings).liveTranslation) return;
  if (debounceTimer) clearTimeout(debounceTimer);
  if (throttleTimer) {
    // A queued throttled run would race the one we are about to schedule.
    clearTimeout(throttleTimer);
    throttleTimer = null;
  }
  debounceTimer = setTimeout(() => {
    const wait = THROTTLE_MS - (Date.now() - throttleLast);
    if (wait > 0) {
      throttleTimer = setTimeout(() => {
        throttleTimer = null;
        throttleLast = Date.now();
        void runTranslate(false);
      }, wait);
    } else {
      throttleLast = Date.now();
      void runTranslate(false);
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
  const limit = get(st.charLimit);
  const clip = (t: string): string => (limit != null ? t.slice(0, limit) : t);

  if (src === st.AUTO) {
    const det = get(st.detected);
    if (!det) return; // disabled; no-op guard
    const prevTarget = get(st.targetLang);
    if (det.language === prevTarget) return; // swap would set source == target
    generation++; // an in-flight translation must not overwrite swapped panes
    st.sourceLang.set(prevTarget);
    st.targetLang.set(det.language);
    st.sourceText.set(clip(tgtText));
    st.translatedText.set(srcText);
    st.detected.set(null);
  } else {
    const tgt = get(st.targetLang);
    generation++;
    st.sourceLang.set(tgt);
    st.targetLang.set(src);
    st.sourceText.set(clip(tgtText));
    st.translatedText.set(srcText);
  }
  st.isTranslating.set(false);
  void persistSettingsDebounced();
}

/** Clear input, output and detected language (SPEC §6.3). */
export function clearAll(): void {
  generation++; // drop any in-flight result
  resetAttemptCache(); // retyping the same text must translate again
  st.sourceText.set('');
  st.translatedText.set('');
  st.detected.set(null);
  st.isTranslating.set(false);
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
    void updateSettings({
      lastSourceLang: get(st.sourceLang),
      lastTargetLang: get(st.targetLang),
    });
  }, 400);
}

/**
 * Merge a partial settings patch into the persisted settings via the backend
 * and sync the local store with the authoritative merged result. Returns
 * whether the save succeeded.
 */
export async function updateSettings(patch: Partial<Settings>): Promise<boolean> {
  try {
    const merged = await UpdateSettings(patch);
    st.settings.set(merged);
    return true;
  } catch (e) {
    st.showToast('error', 'Could not save settings: ' + errorMessage(e));
    return false;
  }
}
