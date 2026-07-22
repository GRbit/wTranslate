<script lang="ts">
  import { infoModal } from '../store';
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';

  const REPO_URL = 'https://github.com/GRbit/wTranslate';

  function close(): void {
    infoModal.set(null);
  }

  function openRepo(): void {
    // External links must go through the OS browser, not the webview.
    BrowserOpenURL(REPO_URL);
  }
</script>

<svelte:window on:keydown={(e) => e.key === 'Escape' && close()} />

{#if $infoModal}
  <div class="overlay">
    <div
      class="backdrop"
      role="button"
      tabindex="0"
      aria-label="Close"
      on:click={close}
      on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && close()}
    />

    <div class="modal" role="dialog" aria-modal="true">
      <header>
        <h2>{$infoModal === 'help' ? 'How to run' : 'Credits'}</h2>
        <button class="x" on:click={close} aria-label="Close">×</button>
      </header>

      {#if $infoModal === 'help'}
        <section>
          <h3>In the app</h3>
          <p>
            Type or paste text and press <code>Ctrl+Enter</code> (or
            <code>Enter</code>, depending on the Translation Shortcut setting)
            to translate. The app lives in the system tray: left click the tray
            icon to hide or show the window, right click for a menu. Closing
            the window only hides it to the tray; quit via the Help or tray
            menu, or with <code>Ctrl+Q</code> in the window.
          </p>
        </section>

        <section>
          <h3>Command line</h3>
          <table>
            <tr>
              <td><code>translator</code></td>
              <td>
                Start the app. If it is already running, the existing window is
                brought to the front instead of starting a second copy.
              </td>
            </tr>
            <tr>
              <td><code>translator --translate-clipboard</code></td>
              <td>
                Also paste the clipboard into the source box and translate it
                right away.
              </td>
            </tr>
            <tr>
              <td><code>translator --debug</code></td>
              <td>Verbose logging to stderr.</td>
            </tr>
            <tr>
              <td><code>translator --help</code></td>
              <td>Print this help in the terminal.</td>
            </tr>
          </table>
        </section>

        <section>
          <h3>Global hotkey</h3>
          <p>
            The app does not grab keys system-wide itself. Instead, in your
            desktop environment's keyboard shortcut settings, bind a shortcut
            to:
          </p>
          <p><code>translator --translate-clipboard</code></p>
          <p>
            Then copy some text anywhere, press the shortcut, and the window
            pops up with the translation.
          </p>
        </section>
      {:else}
        <section>
          <p>
            <strong>LibreTranslate Translator</strong> - a desktop client for
            LibreTranslate, built with Wails and Svelte.
          </p>
          <p>
            Source code:
            <button class="link" on:click={openRepo}>{REPO_URL}</button>
          </p>
          <p class="muted">
            Contributions are welcome!
          </p>
        </section>
      {/if}

      <footer>
        <button class="primary" on:click={close}>Close</button>
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
    width: 560px;
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
  p {
    margin: 6px 0;
    font-size: 13px;
    line-height: 1.5;
  }
  table {
    border-collapse: collapse;
    font-size: 13px;
  }
  td {
    padding: 4px 12px 4px 0;
    vertical-align: top;
    line-height: 1.5;
  }
  code {
    background: var(--panel);
    padding: 1px 4px;
    border-radius: 4px;
    font-size: 12px;
    white-space: nowrap;
  }
  .link {
    background: transparent;
    border: none;
    padding: 0;
    color: var(--accent);
    text-decoration: underline;
    cursor: pointer;
    font-size: 13px;
  }
  .muted {
    color: var(--muted);
    font-size: 12px;
  }
  footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
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
</style>
