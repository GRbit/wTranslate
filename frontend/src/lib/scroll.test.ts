import { describe, it, expect } from 'vitest';
import { normalizeScroll } from './scroll';

// The tests run in plain node (no DOM), so a fake element with the three
// geometry fields stands in for the real textarea. What this guards: the
// scroll resets when the content fits the viewport (the "scrollbar
// disappeared" case), clamps when it is beyond the new range, and never
// disturbs a valid position. The Svelte wiring that calls it after a store
// update is not covered here and is verified by running the app.
function fakeEl(scrollHeight: number, clientHeight: number, scrollTop: number) {
  return { scrollHeight, clientHeight, scrollTop };
}

describe('normalizeScroll', () => {
  it('resets a stale scroll position when the content fits the viewport', () => {
    const el = fakeEl(100, 200, 150);
    normalizeScroll(el);
    expect(el.scrollTop).toBe(0);
  });

  it('resets when content height equals the viewport exactly', () => {
    const el = fakeEl(200, 200, 40);
    normalizeScroll(el);
    expect(el.scrollTop).toBe(0);
  });

  it('keeps the scroll position while the content still overflows', () => {
    const el = fakeEl(500, 200, 150);
    normalizeScroll(el);
    expect(el.scrollTop).toBe(150);
  });

  // Geometry observed live in WebKitGTK: after a long scrolled translation
  // was replaced by a short one, the engine kept scrollTop=393 with
  // scrollHeight=238 and clientHeight=237. scrollHeight is rounded up 1px
  // above clientHeight even with no real overflow, so an exact
  // sh <= ch comparison never fires.
  it('resets despite the 1px scrollHeight rounding artifact', () => {
    const el = fakeEl(238, 237, 393);
    normalizeScroll(el);
    expect(el.scrollTop).toBe(0);
  });

  it('clamps a stale position when shorter content still overflows', () => {
    const el = fakeEl(630, 237, 500);
    normalizeScroll(el);
    expect(el.scrollTop).toBe(393);
  });
});
