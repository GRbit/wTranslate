/**
 * normalizeScroll brings a pane's scroll position back into the valid range
 * after its content changed.
 *
 * WebKitGTK neither resets nor clamps scrollTop when a textarea's value is
 * replaced with shorter text: observed live scrollTop=393 with
 * scrollHeight=238 / clientHeight=237. The engine then treats the scroll
 * range as empty, so no scrollbar is shown and wheel input is ignored: the
 * content is stuck above the viewport with no way back. Only a programmatic
 * write recovers it.
 */
export function normalizeScroll(
  el: Pick<HTMLElement, 'scrollHeight' | 'clientHeight' | 'scrollTop'>,
): void {
  const maxScroll = el.scrollHeight - el.clientHeight;
  // scrollHeight and clientHeight round independently: a pane with no real
  // overflow still reports sh = ch + 1, so an exact sh <= ch check misses
  // the broken state. Treat a couple of px as "no overflow".
  if (maxScroll <= 2) {
    if (el.scrollTop !== 0) el.scrollTop = 0;
  } else if (el.scrollTop > maxScroll) {
    el.scrollTop = maxScroll;
  }
}
