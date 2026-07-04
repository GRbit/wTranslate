<script lang="ts">
  import {
    settings,
    languages,
    sourceLang,
    targetLang,
    sourceText,
    translatedText,
    detected,
    isTranslating,
    settingsOpen,
    charCount,
    overLimit,
    swapDisabled,
    CHAR_LIMIT,
    AUTO,
  } from '../store';
  import {
    runTranslate,
    scheduleLive,
    swap,
    clearAll,
    pasteToSource,
    copyTranslation,
    persistSettingsDebounced,
  } from '../translate';
  import { langName } from '../languages';

  // SPEC §5.2: left select shows "Auto (Name – N%)" when a language was detected.
  let autoOptionLabel: string;
  $: autoOptionLabel =
    $sourceLang === AUTO && $detected
      ? `Auto (${langName($detected.language)} – ${Math.round($detected.confidence)}%)`
      : 'Auto';

  // SPEC §5.4: any explicit language change resets the detected language.
  function onSourceChange(): void {
    $detected = null;
    void persistSettingsDebounced();
  }
  function onTargetChange(): void {
    $detected = null;
    void persistSettingsDebounced();
  }

  // SPEC §3.2.2: keyboard shortcut handling on the source textarea.
  function onKeydown(e: KeyboardEvent): void {
        const withCtrl = e.ctrlKey || e.metaKey;

    // Ctrl+S: Swap languages
    if (withCtrl && e.key.toLowerCase() === 's') {
        e.preventDefault(); // avoid default browser save action
        if (!$swapDisabled) {
            swap();
        }
        return;
    }

    // Ctrl+T: Copy translated text
    if (withCtrl && e.key.toLowerCase() === 't') {
        e.preventDefault(); // avoid default new tab open
        void copyTranslation();
        return;
    }

    // Translate on hotkey press
    if (e.key === 'Enter') {
        if ($settings.shortcut === 'ctrl_enter') {
            if (withCtrl) {
                e.preventDefault();
                void runTranslate();
            }
        } else {
            if (!withCtrl) {
                e.preventDefault();
                void runTranslate();
            }
        }
    }
  }
</script>

<main class="translator">
  <section class="toolbar">
    <select bind:value={$sourceLang} on:change={onSourceChange} aria-label="Source language">
      <option value={AUTO}>{autoOptionLabel}</option>
      {#each $languages as l (l.code)}
        <option value={l.code}>{l.name}</option>
      {/each}
    </select>

    <button
      class="swap"
      disabled={$swapDisabled}
      on:click={swap}
      title="Swap languages"
      aria-label="Swap languages">⇄</button
    >

    <select bind:value={$targetLang} on:change={onTargetChange} aria-label="Target language">
      {#each $languages as l (l.code)}
        <option value={l.code}>{l.name}</option>
      {/each}
    </select>
  </section>

  <section class="panes">
    <div class="pane">
      <textarea
        bind:value={$sourceText}
        on:input={() => scheduleLive()}
        on:keydown={onKeydown}
        maxlength={CHAR_LIMIT}
        placeholder="Enter text to translate…"
        spellcheck="false"
      ></textarea>
      <div class="pane-foot">
        <button class="ghost" on:click={clearAll} title="Clear text">🧹 Clear</button>
        <button class="ghost" on:click={pasteToSource} title="Paste">📋 Paste</button>
      </div>
    </div>

    <div class="pane">
      <textarea
        bind:value={$translatedText}
        readonly
        placeholder="Translation"
        spellcheck="false"
      ></textarea>
      <div class="pane-foot right">
        <button class="ghost" on:click={copyTranslation} title="Copy translation">⧉ Copy</button>
      </div>
    </div>
  </section>

  <section class="bottombar">
    <button class="primary" on:click={() => runTranslate()} disabled={$isTranslating}>
      {#if $isTranslating}Translating…{:else}Translate{/if}
    </button>
    <span class="counter" class:over={$overLimit}>{$charCount}/{CHAR_LIMIT}</span>
    <button class="ghost" on:click={() => settingsOpen.set(true)} title="Settings">
      ⚙ Settings
    </button>
  </section>
</main>

<style>
  .translator {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    padding: 8px;
    gap: 8px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .toolbar select {
    flex: 1;
    padding: 6px 8px;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
  }
  .swap {
    flex: 0 0 auto;
    padding: 6px 12px;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
  }
  .swap:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .panes {
    flex: 1;
    display: flex;
    gap: 8px;
    min-height: 0;
  }
  .pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: var(--panel);
    overflow: hidden;
  }
  .pane textarea {
    flex: 1;
    border: none;
    outline: none;
    resize: none;
    padding: 10px;
    background: transparent;
    color: var(--text);
    line-height: 1.4;
  }
  .pane textarea::placeholder {
    color: var(--muted);
  }
  .pane-foot {
    display: flex;
    padding: 6px 8px;
    border-top: 1px solid var(--panel-border);
    background: #fff;
  }
  .pane-foot.right {
    justify-content: flex-end;
  }

  .bottombar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 4px 0;
  }
  .primary {
    padding: 8px 24px;
    border: none;
    border-radius: var(--radius);
    background: var(--accent);
    color: #fff;
    font-weight: 600;
  }
  .primary:hover {
    background: var(--accent-hover);
  }
  .primary:disabled {
    opacity: 0.6;
    cursor: progress;
  }
  .ghost {
    padding: 6px 12px;
    border: 1px solid var(--panel-border);
    border-radius: var(--radius);
    background: #fff;
    color: var(--text);
  }
  .ghost:hover {
    background: var(--panel);
  }
  .counter {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
    font-size: 13px;
  }
  .counter.over {
    color: var(--danger);
    font-weight: 600;
  }
</style>
