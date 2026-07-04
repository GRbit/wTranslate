import { get } from 'svelte/store';
import * as st from './store';
import { GetSettings, GetLanguages, errorMessage } from './api';

/**
 * Load settings from the backend and initialise the UI accordingly
 * (SPEC §9.3: apply Default-to-Auto / last selected languages).
 */
export async function initApp(): Promise<void> {
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
  await loadLanguages();
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
      'Could not load languages: ' + errorMessage(e) + ' — check the Base URL in Settings.',
    );
  }
}
