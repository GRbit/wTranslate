<script lang="ts">
  import { toast, clearToast } from '../store';
</script>

{#if $toast}
  <div class="toast {$toast.kind}" role="status">
    <span class="msg">{$toast.message}</span>
    <button class="close" on:click={clearToast} aria-label="Dismiss">×</button>
  </div>
{/if}

<style>
  .toast {
    display: flex;
    align-items: center;
    gap: 12px;
    margin: 8px;
    padding: 8px 12px;
    border-radius: var(--radius);
    font-size: 13px;
    line-height: 1.3;
  }
  .toast.error {
    background: var(--danger-bg);
    color: var(--danger);
  }
  .toast.info {
    background: var(--ok-bg);
    color: var(--ok);
  }
  .msg {
    flex: 1;
  }
  .close {
    background: transparent;
    border: none;
    color: inherit;
    font-size: 18px;
    line-height: 1;
    padding: 0 4px;
    opacity: 0.7;
  }
  .close:hover {
    opacity: 1;
  }
</style>
