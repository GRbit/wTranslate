<script lang="ts">
  import { onMount } from 'svelte';
  import Toast from './lib/components/Toast.svelte';
  import Translator from './lib/components/Translator.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import InfoModal from './lib/components/InfoModal.svelte';
  import { initApp } from './lib/init';
  import { infoModal } from './lib/store';
  import { translateClipboard } from './lib/translate';
  import { TranslateClipboardOnLaunch } from '../wailsjs/go/main/App.js';
  import { EventsEmit, EventsOn, Quit } from '../wailsjs/runtime/runtime.js';

  onMount(() => {
    void (async () => {
      await initApp();
      // --translate-clipboard on a fresh start: translate once the languages
      // and settings are loaded.
      if (await TranslateClipboardOnLaunch()) void translateClipboard();
    })();
    // Window-menu items open modals via events from the backend.
    EventsOn('menu:help', () => infoModal.set('help'));
    EventsOn('menu:credits', () => infoModal.set('credits'));
    // A second `translator --translate-clipboard` launch forwards here.
    EventsOn('app:translate-clipboard', () => void translateClipboard());
    // Keep the backend's window-visibility flag (used by the tray hide/show
    // toggle) in sync - fires when the window is hidden via the close button
    // (HideWindowOnClose), minimize, WindowHide, or shown again.
    document.addEventListener('visibilitychange', () =>
      EventsEmit('window:visibility', document.visibilityState === 'visible'),
    );
    // Ctrl+Q while the window has focus (not global) quits the app.
    window.addEventListener('keydown', (e) => {
      if (e.ctrlKey && !e.shiftKey && !e.altKey && !e.metaKey && e.key.toLowerCase() === 'q') {
        e.preventDefault();
        Quit();
      }
    });
  });
</script>

<Toast />
<Translator />
<SettingsModal />
<InfoModal />
