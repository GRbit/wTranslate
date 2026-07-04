<script lang="ts">
  import { onMount } from 'svelte';
  import Toast from './lib/components/Toast.svelte';
  import Translator from './lib/components/Translator.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import { initApp } from './lib/init';

  onMount(() => {
    void initApp();
  });
</script>

<Toast />
<Translator />
<SettingsModal />
