<script lang="ts">
  import { settings, settingsOpen, SHORTCUT_CTRL_ENTER, SHORTCUT_ENTER } from '../store';
  import { saveCurrentSettings } from '../translate';
  import { loadLanguages } from '../init';
  import type { Settings } from '../types';

  let form: Settings = { ...$settings };
  let providerOpen = true;

  // Refresh local form only when open
  let lastOpen = false;
  $: if ($settingsOpen && !lastOpen) {
    // Инициализация только при переходе из false в true
    form = { ...$settings };
    providerOpen = true;
    lastOpen = true;
  } else if (!$settingsOpen) {
    lastOpen = false;
  }


  function close(): void {
    settingsOpen.set(false);
  }

  async function save(): Promise<void> {
    const urlChanged = form.baseUrl !== $settings.baseUrl;
    settings.set(form); // optimistic UI update
    await saveCurrentSettings(form);
    if (urlChanged) {
      await loadLanguages(); // Re-fetch languages on URL change
    }
    close();
  }
</script>

<svelte:window on:keydown={(e) => e.key === 'Escape' && close()} />

{#if $settingsOpen}
  <div class="overlay">
    <!-- invisible backdrop layer to close on click/keypress ->
    <div
      class="backdrop"
      role="button"
      tabindex="0"
      aria-label="Close settings"
      on:click={close}
      on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && close()}
    />

    <!-- Само модальное окно теперь изолировано от обработчиков закрытия -->
    <div class="modal" role="dialog" aria-modal="true">
      <header>
        <h2>Settings</h2>
        <button class="x" on:click={close} aria-label="Close">×</button>
      </header>

      <section>
        <h3>Providers</h3>
        <div class="row">
          <span>Translator</span>
          <strong>LibreTranslate</strong>
          <button class="gear" on:click={() => (providerOpen = !providerOpen)} title="Configure">⚙</button>
        </div>
        {#if providerOpen}
          <div class="sub">
            <div class="field">
              <label for="base-url">Base URL</label>
              <input
                id="base-url"
                type="text"
                bind:value={form.baseUrl}
                placeholder="https://libretranslate.com"
                autocomplete="off"
                spellcheck="false"
              />
            </div>
            <div class="field">
              <label for="api-key">API key</label>
              <input
                id="api-key"
                type="text"
                bind:value={form.apiKey}
                placeholder="(optional)"
                autocomplete="off"
                spellcheck="false"
              />
            </div>
          </div>
        {/if}
      </section>

      <section>
        <h3>Behavior</h3>
        <label class="toggle">
          <input type="checkbox" bind:checked={form.liveTranslation} />
          <span>Live Translation</span>
        </label>
        <p class="warn">
          Warning: Live Translation sends a request on every edit. Aggressive use
          may get your IP banned by the instance.
        </p>

        <div class="field">
          <label for="shortcut">Translation Shortcut</label>
          <select id="shortcut" bind:value={form.shortcut}>
            <option value={SHORTCUT_CTRL_ENTER}>Ctrl + Enter translates, Enter = newline</option>
            <option value={SHORTCUT_ENTER}>Enter translates, Ctrl + Enter = newline</option>
          </select>
        </div>

        <label class="toggle">
          <input type="checkbox" bind:checked={form.defaultToAuto} />
          <span>Default to Auto</span>
        </label>
      </section>

      <section>
        <h3>Appearance</h3>
        <div class="field disabled">
          <label for="font-size">Font size</label>
          <select id="font-size" disabled><option>System default</option></select>
        </div>
        <p class="muted">Custom font size is disabled in this version (system font is used).</p>
      </section>

      <section>
        <h3>Developer</h3>
        <label class="toggle">
          <input type="checkbox" bind:checked={form.debug} />
          <span>Debug logging</span>
        </label>
        <p class="muted">
          Write verbose logs to stderr (every HTTP request, response status,
          timings, settings load/save). The API key is never logged. Also enabled
          by launching with <code>--debug</code> or
          <code>LIBRETRANSLATE_DEBUG=1</code>.
        </p>
      </section>

      <footer>
        <button class="ghost" on:click={close}>Cancel</button>
        <button class="primary" on:click={save}>Save</button>
      </footer>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10;
  }
  .backdrop {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    cursor: default;
    z-index: 1;
  }
  .modal {
    position: relative;
    z-index: 2; 
    width: 520px;
    max-width: 92vw;
    max-height: 88vh;
    overflow: auto;
    background: #fff;
    border-radius: 10px;
    padding: 16px 20px;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.25);
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  h2 {
    margin: 0;
    font-size: 18px;
  }
  h3 {
    margin: 16px 0 8px;
    font-size: 14px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .x {
    background: transparent;
    border: none;
    font-size: 22px;
    line-height: 1;
    color: var(--muted);
    cursor: pointer;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 0;
  }
  .row strong {
    flex: 1;
  }
  .gear {
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
    padding: 2px 10px;
    cursor: pointer;
  }
  .sub {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 8px 0 4px 12px;
    margin-left: 12px;
    border-left: 2px solid var(--panel-border);
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 10px;
  }
  .field label {
    color: var(--muted);
    font-size: 13px;
  }
  .field input,
  .field select {
    width: 100%;
    padding: 6px 8px;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
    color: var(--text);
    cursor: text;
  }
  .field input:focus,
  .field select:focus {
    outline: 2px solid var(--accent);
    outline-offset: -1px;
  }
  .toggle {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
    font-size: 13px;
    cursor: pointer;
  }
  .toggle input {
    cursor: pointer;
  }
  .disabled {
    opacity: 0.6;
  }
  .warn {
    margin: 6px 0 0;
    padding: 8px 10px;
    background: #fff7ed;
    color: #9a3412;
    border-radius: var(--radius);
    font-size: 12px;
    line-height: 1.4;
  }
  .muted {
    margin: 6px 0 0;
    color: var(--muted);
    font-size: 12px;
    line-height: 1.4;
  }
  code {
    background: var(--panel);
    padding: 1px 4px;
    border-radius: 4px;
    font-size: 11px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 20px;
  }
  .primary {
    padding: 8px 20px;
    border: none;
    border-radius: var(--radius);
    background: var(--accent);
    color: #fff;
    font-weight: 600;
    cursor: pointer;
  }
  .primary:hover {
    background: var(--accent-hover);
  }
  .ghost {
    padding: 8px 16px;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
    cursor: pointer;
  }
</style>
