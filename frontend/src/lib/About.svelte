<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
  import Logo from "./Logo.svelte";
  import Icon from "./Icon.svelte";

  export let version = "";

  const REPO = "https://github.com/einarhh/mqtt-manager";
  const dispatch = createEventDispatcher();

  function close() {
    dispatch("close");
  }
</script>

<div class="overlay" on:click|self={close}>
  <div class="modal">
    <header>
      <span class="title">About</span>
      <button class="x" on:click={close} aria-label="Close"><Icon name="close" size={16} /></button>
    </header>

    <div class="body">
      <Logo size={64} />
      <div class="name">MQTT Manager</div>
      {#if version}<div class="version">Version {version}</div>{/if}
      <p class="desc">
        A desktop MQTT client — a live topic tree, per-topic inspection with
        history, publishing, and custom decoder plugins.
      </p>

      <button class="repo" on:click={() => BrowserOpenURL(REPO)}>
        <Icon name="external" size={14} />
        View on GitHub
      </button>

      <div class="note">
        <Icon name="warning" size={14} />
        <span>Connection passwords are stored unencrypted on this device.</span>
      </div>

      <div class="copyright">© 2026 Einar Helseth</div>
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 50;
  }
  .modal {
    width: min(420px, 92vw);
    background: var(--bg-panel);
    border: 1px solid var(--border);
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow: 0 18px 48px var(--shadow);
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-bottom: 1px solid var(--border);
    background: var(--bg-bar);
  }
  .title {
    font-weight: 700;
  }
  .x {
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    padding: 2px;
    border-radius: 6px;
  }
  .x:hover {
    color: var(--text);
    background: var(--bg-hover);
  }
  .body {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 8px;
    padding: 24px 24px 20px;
  }
  .name {
    font-size: 18px;
    font-weight: 700;
    margin-top: 4px;
  }
  .version {
    font-size: 12px;
    color: var(--text-dim);
    font-family: var(--mono);
  }
  .desc {
    font-size: 13px;
    color: var(--text-dim);
    line-height: 1.5;
    margin: 6px 0 4px;
    max-width: 36ch;
  }
  .repo {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
  }
  .note {
    display: flex;
    align-items: center;
    gap: 8px;
    text-align: left;
    font-size: 12px;
    color: var(--text-dim);
    background: var(--bg-inset);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 8px 12px;
    margin-top: 8px;
  }
  .note :global(.icon) {
    color: var(--warn);
    flex: 0 0 auto;
  }
  .copyright {
    font-size: 11px;
    color: var(--text-dim);
    margin-top: 8px;
  }
</style>
