import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { Mock } from 'vitest';
import { get } from 'svelte/store';

// translate.ts talks to the backend only through ./api; mocking it here keeps
// the tests free of Wails bindings (which need a running runtime).
vi.mock('./api', () => ({
  Translate: vi.fn(),
  UpdateSettings: vi.fn(),
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
}));

// translate.ts keeps module-level state (generation counter, attempt cache,
// debounce/throttle timers, throttleLast). Each test gets a fresh module graph
// so state never leaks between tests.
let st: typeof import('./store');
let tr: typeof import('./translate');
let api: { Translate: Mock; UpdateSettings: Mock };

beforeEach(async () => {
  vi.resetModules();
  vi.useFakeTimers();
  api = (await import('./api')) as unknown as typeof api;
  // resetModules does not reset the mock registry: the mocked ./api module is
  // shared across tests, so its call history and queued results must be wiped.
  api.Translate.mockReset();
  api.UpdateSettings.mockReset();
  st = await import('./store');
  tr = await import('./translate');
});

afterEach(() => {
  vi.useRealTimers();
});

describe('runTranslate', () => {
  it('sends the request and applies the response', async () => {
    st.sourceText.set('hello');
    st.targetLang.set('de');
    api.Translate.mockResolvedValue({
      translatedText: 'hallo',
      detectedLanguage: { language: 'en', confidence: 92 },
    });

    await tr.runTranslate();

    expect(api.Translate).toHaveBeenCalledWith({ q: 'hello', source: st.AUTO, target: 'de' });
    expect(get(st.translatedText)).toBe('hallo');
    expect(get(st.detected)).toEqual({ language: 'en', confidence: 92 });
    expect(get(st.isTranslating)).toBe(false);
  });

  it('clears output and skips the API on blank input', async () => {
    st.translatedText.set('stale');
    st.detected.set({ language: 'en', confidence: 50 });
    st.sourceText.set('   ');

    await tr.runTranslate();

    expect(api.Translate).not.toHaveBeenCalled();
    expect(get(st.translatedText)).toBe('');
    expect(get(st.detected)).toBeNull();
  });

  it('skips a repeat of the last successful (text, source, target) request', async () => {
    st.sourceText.set('hello');
    api.Translate.mockResolvedValue({ translatedText: 'hallo' });

    await tr.runTranslate();
    await tr.runTranslate();
    expect(api.Translate).toHaveBeenCalledTimes(1);

    // Changing any part of the triple must translate again.
    st.targetLang.set('fr');
    await tr.runTranslate();
    expect(api.Translate).toHaveBeenCalledTimes(2);
  });

  it('on error: shows a toast, clears output, and allows an identical retry', async () => {
    st.sourceText.set('hello');
    st.translatedText.set('stale');
    api.Translate.mockRejectedValueOnce(new Error('boom'));

    await tr.runTranslate();

    expect(get(st.toast)?.kind).toBe('error');
    expect(get(st.toast)?.message).toContain('boom');
    expect(get(st.translatedText)).toBe('');
    expect(get(st.detected)).toBeNull();
    expect(get(st.isTranslating)).toBe(false);

    // The failed attempt must not populate the redundancy cache.
    api.Translate.mockResolvedValueOnce({ translatedText: 'hallo' });
    await tr.runTranslate();
    expect(api.Translate).toHaveBeenCalledTimes(2);
    expect(get(st.translatedText)).toBe('hallo');
  });

  it('discards a stale in-flight response (generation counter)', async () => {
    let resolveFirst!: (v: unknown) => void;
    api.Translate.mockReturnValueOnce(new Promise((r) => (resolveFirst = r)));
    st.sourceText.set('first');
    const first = tr.runTranslate();

    st.sourceText.set('second');
    api.Translate.mockResolvedValueOnce({ translatedText: 'SECOND' });
    await tr.runTranslate();
    expect(get(st.translatedText)).toBe('SECOND');

    resolveFirst({ translatedText: 'FIRST' });
    await first;

    expect(get(st.translatedText)).toBe('SECOND');
    expect(get(st.isTranslating)).toBe(false);
  });

  it('clearAll invalidates an in-flight request', async () => {
    let resolve!: (v: unknown) => void;
    api.Translate.mockReturnValueOnce(new Promise((r) => (resolve = r)));
    st.sourceText.set('hello');
    const pending = tr.runTranslate();

    tr.clearAll();
    resolve({ translatedText: 'hallo' });
    await pending;

    expect(get(st.translatedText)).toBe('');
    expect(get(st.detected)).toBeNull();
    expect(get(st.isTranslating)).toBe(false);
  });

  it('enforces the instance char limit: toast on manual, silent in live mode', async () => {
    st.charLimit.set(5);
    st.sourceText.set('longer than five');

    await tr.runTranslate(false);
    expect(api.Translate).not.toHaveBeenCalled();
    expect(get(st.toast)).toBeNull();

    await tr.runTranslate(true);
    expect(api.Translate).not.toHaveBeenCalled();
    expect(get(st.toast)?.kind).toBe('error');
    expect(get(st.toast)?.message).toContain('5');
  });
});

describe('scheduleLive', () => {
  it('does nothing when live translation is off', async () => {
    st.sourceText.set('hello');
    tr.scheduleLive();
    await vi.advanceTimersByTimeAsync(60_000);
    expect(api.Translate).not.toHaveBeenCalled();
  });

  it('debounces 600ms after the last edit', async () => {
    st.settings.update((s) => ({ ...s, liveTranslation: true }));
    st.sourceText.set('hello');
    api.Translate.mockResolvedValue({ translatedText: 'hallo' });

    tr.scheduleLive();
    await vi.advanceTimersByTimeAsync(599);
    expect(api.Translate).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(1);
    expect(api.Translate).toHaveBeenCalledTimes(1);
  });

  it('a new edit within the debounce window restarts it', async () => {
    st.settings.update((s) => ({ ...s, liveTranslation: true }));
    st.sourceText.set('hel');
    api.Translate.mockResolvedValue({ translatedText: 'x' });

    tr.scheduleLive();
    await vi.advanceTimersByTimeAsync(400);
    st.sourceText.set('hello');
    tr.scheduleLive();
    await vi.advanceTimersByTimeAsync(400);
    expect(api.Translate).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(200);
    expect(api.Translate).toHaveBeenCalledTimes(1);
    expect(api.Translate).toHaveBeenCalledWith(
      expect.objectContaining({ q: 'hello' }),
    );
  });

  it('throttles to one request per 5s after the first live run', async () => {
    st.settings.update((s) => ({ ...s, liveTranslation: true }));
    api.Translate.mockResolvedValue({ translatedText: 'x' });

    st.sourceText.set('one');
    tr.scheduleLive();
    await vi.advanceTimersByTimeAsync(600);
    expect(api.Translate).toHaveBeenCalledTimes(1);

    st.sourceText.set('two');
    tr.scheduleLive();
    // Debounce elapses, but the throttle window (5s since the first run) has
    // not, so the run is queued instead of fired.
    await vi.advanceTimersByTimeAsync(600);
    expect(api.Translate).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(4_400);
    expect(api.Translate).toHaveBeenCalledTimes(2);
    expect(api.Translate).toHaveBeenLastCalledWith(
      expect.objectContaining({ q: 'two' }),
    );
  });
});

describe('swap', () => {
  beforeEach(() => {
    api.UpdateSettings.mockResolvedValue({});
  });

  it('swaps languages and panes in explicit-language mode', () => {
    st.sourceLang.set('en');
    st.targetLang.set('de');
    st.sourceText.set('hi');
    st.translatedText.set('hallo');

    tr.swap();

    expect(get(st.sourceLang)).toBe('de');
    expect(get(st.targetLang)).toBe('en');
    expect(get(st.sourceText)).toBe('hallo');
    expect(get(st.translatedText)).toBe('hi');
  });

  it('in Auto mode uses the detected language and drops Auto', () => {
    st.sourceLang.set(st.AUTO);
    st.targetLang.set('de');
    st.detected.set({ language: 'en', confidence: 90 });
    st.sourceText.set('hi');
    st.translatedText.set('hallo');

    tr.swap();

    expect(get(st.sourceLang)).toBe('de');
    expect(get(st.targetLang)).toBe('en');
    expect(get(st.sourceText)).toBe('hallo');
    expect(get(st.translatedText)).toBe('hi');
    expect(get(st.detected)).toBeNull();
  });

  it('is a no-op in Auto mode without a detected language', () => {
    st.sourceLang.set(st.AUTO);
    st.targetLang.set('de');
    st.sourceText.set('hi');

    tr.swap();

    expect(get(st.sourceLang)).toBe(st.AUTO);
    expect(get(st.targetLang)).toBe('de');
    expect(get(st.sourceText)).toBe('hi');
  });

  it('is a no-op when the detected language equals the target', () => {
    st.sourceLang.set(st.AUTO);
    st.targetLang.set('en');
    st.detected.set({ language: 'en', confidence: 90 });
    st.sourceText.set('hi');

    tr.swap();

    expect(get(st.sourceLang)).toBe(st.AUTO);
    expect(get(st.targetLang)).toBe('en');
    expect(get(st.sourceText)).toBe('hi');
  });

  it('clips the swapped-in text to the instance char limit', () => {
    st.charLimit.set(3);
    st.sourceLang.set('en');
    st.targetLang.set('de');
    st.sourceText.set('hi');
    st.translatedText.set('abcdef');

    tr.swap();

    expect(get(st.sourceText)).toBe('abc');
  });

  it('persists the swapped language pair (debounced)', async () => {
    st.sourceLang.set('en');
    st.targetLang.set('de');

    tr.swap();
    expect(api.UpdateSettings).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(400);
    expect(api.UpdateSettings).toHaveBeenCalledWith({
      lastSourceLang: 'de',
      lastTargetLang: 'en',
    });
  });
});

describe('updateSettings', () => {
  it('syncs the store with the merged result from the backend', async () => {
    const merged = { ...st.defaultSettings(), apiKey: 'k', liveTranslation: true };
    api.UpdateSettings.mockResolvedValue(merged);

    const ok = await tr.updateSettings({ apiKey: 'k' });

    expect(ok).toBe(true);
    expect(get(st.settings)).toEqual(merged);
  });

  it('reports failure with a toast and leaves the store unchanged', async () => {
    const before = get(st.settings);
    api.UpdateSettings.mockRejectedValue(new Error('disk full'));

    const ok = await tr.updateSettings({ apiKey: 'k' });

    expect(ok).toBe(false);
    expect(get(st.settings)).toEqual(before);
    expect(get(st.toast)?.kind).toBe('error');
    expect(get(st.toast)?.message).toContain('disk full');
  });
});
