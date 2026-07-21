<script lang="ts">
  import { onMount } from 'svelte';
  import Toast from './lib/components/Toast.svelte';
  import Translator from './lib/components/Translator.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import { initApp } from './lib/init';
  import { EventsEmit, Quit } from '../wailsjs/runtime/runtime.js';

  onMount(() => {
    void initApp();
    // Keep the backend's window-visibility flag (used by the tray hide/show
    // toggle) in sync - fires when the window is hidden via minimize,
    // WindowHide, or shown again.
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
