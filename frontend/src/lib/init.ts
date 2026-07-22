import { get } from 'svelte/store';
import * as st from './store';
import { GetSettings, GetLanguages, GetFrontendSettings, LoadWarning, errorMessage } from './api';

/**
 * Load settings from the backend and initialise the UI accordingly
 * (SPEC §9.3: apply Default-to-Auto / last selected languages).
 */
export async function initApp(): Promise<void> {
  // Surface any non-fatal warning recorded while the backend loaded settings
  // (e.g. a corrupt config was reset to defaults). SPEC §9 / BUGS #6.
  try {
    const warn = await LoadWarning();
    if (warn) st.showToast('error', warn, 0);
  } catch {
    // Non-fatal; ignore.
  }

  try {
    const s = await GetSettings();
    st.settings.set(s);
    if (s.defaultToAuto) {
      st.sourceLang.set(st.AUTO);
      st.targetLang.set(s.lastTargetLang || 'en');
    } else {
      st.sourceLang.set(s.lastSourceLang || st.AUTO);
      st.targetLang.set(s.lastTargetLang || 'en');
    }
  } catch (e) {
    st.showToast('error', 'Could not load settings: ' + errorMessage(e));
  }
  await Promise.all([loadLanguages(), loadCharLimit()]);
}

/**
 * Fetch the instance's character limit (SPEC §6.1) so the UI shows and enforces
 * the same bound the server does. Re-callable when the Base URL changes. On
 * failure the limit is left as "unlimited": the server still rejects oversized
 * requests with a 400, which surfaces as an error toast.
 */
export async function loadCharLimit(): Promise<void> {
  try {
    const fs = await GetFrontendSettings();
    st.charLimit.set(fs.charLimit > 0 ? fs.charLimit : null);
  } catch {
    st.charLimit.set(null);
  }
}

/**
 * Fetch the supported languages (SPEC §4.1.1). Re-callable when the Base URL
 * changes in settings. Falls back gracefully and corrects invalid selections.
 */
export async function loadLanguages(): Promise<void> {
  try {
    const langs = await GetLanguages();
    if (!langs || !langs.length) {
      st.showToast('error', 'No languages returned. Check the LibreTranslate Base URL in Settings.');
      return;
    }
    st.languages.set(langs);

    const tgt = get(st.targetLang);
    if (!langs.some((l) => l.code === tgt)) {
      st.targetLang.set('en');
    }
    const src = get(st.sourceLang);
    if (src !== st.AUTO && !langs.some((l) => l.code === src)) {
      st.sourceLang.set(st.AUTO);
    }
  } catch (e) {
    st.showToast(
      'error',
      'Could not load languages: ' + errorMessage(e) + ' - check the Base URL in Settings.',
    );
  }
}
