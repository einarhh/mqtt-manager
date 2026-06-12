<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
  import { Version } from "../wailsjs/go/main/App";
  import { ingest, status, totalTopics, totalMessages } from "./lib/stores";
  import type { IncomingMessage, ConnStatus } from "./lib/stores";
  import { reload as reloadPlugins } from "./lib/plugins";

  let appVersion = "";
  let showPlugins = false;
  import Logo from "./lib/Logo.svelte";
  import ConnectionPanel from "./lib/ConnectionPanel.svelte";
  import TopicTree from "./lib/TopicTree.svelte";
  import TopicDetail from "./lib/TopicDetail.svelte";
  import PublishPanel from "./lib/PublishPanel.svelte";
  import PluginManager from "./lib/PluginManager.svelte";

  const STATUS_LABELS: Record<string, string> = {
    disconnected: "Disconnected",
    connecting: "Connecting…",
    connected: "Connected",
    reconnecting: "Reconnecting…",
    error: "Error",
  };

  onMount(() => {
    // Clear any listeners left over from a previous mount (e.g. a dev hot-reload
    // while the Go backend keeps running) so each batch is ingested exactly once.
    EventsOff("mqtt:messages");
    EventsOff("mqtt:status");
    EventsOn("mqtt:messages", (batch: IncomingMessage[]) => ingest(batch));
    EventsOn("mqtt:status", (s: ConnStatus) => status.set(s));
    Version().then((v) => (appVersion = v));
    reloadPlugins();
  });

  onDestroy(() => {
    EventsOff("mqtt:messages");
    EventsOff("mqtt:status");
  });
</script>

<div class="app">
  <header>
    <div class="brand">
      <Logo size={26} />
      <span class="app-name">MQTT Manager</span>
      {#if appVersion}<span class="version">{appVersion}</span>{/if}
    </div>
    <div class="status">
      <span class="dot {$status.status}"></span>
      <span class="status-text">{STATUS_LABELS[$status.status] ?? $status.status}</span>
      {#if $status.detail}<span class="detail">{$status.detail}</span>{/if}
    </div>
    <div class="counters">
      {$totalTopics} topics · {$totalMessages} messages
    </div>
    <button class="plugins-btn" on:click={() => (showPlugins = true)}>⚙ Plugins</button>
  </header>

  {#if showPlugins}
    <PluginManager on:close={() => (showPlugins = false)} />
  {/if}

  <main>
    <aside class="left"><ConnectionPanel /></aside>
    <section class="center"><TopicTree /></section>
    <aside class="right">
      <div class="detail-wrap"><TopicDetail /></div>
      <PublishPanel />
    </aside>
  </main>
</div>

<style>
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }
  header {
    display: flex;
    align-items: center;
    gap: 24px;
    height: 48px;
    padding: 0 16px;
    background: var(--bg-bar);
    border-bottom: 1px solid var(--border);
    flex: 0 0 48px;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .app-name {
    font-weight: 700;
    letter-spacing: 0.02em;
  }
  .version {
    font-size: 11px;
    color: var(--text-dim);
    font-family: var(--mono);
    padding: 1px 6px;
    border: 1px solid var(--border);
    border-radius: 6px;
  }
  .status {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: var(--text-dim);
  }
  .dot.connected {
    background: var(--ok);
  }
  .dot.connecting,
  .dot.reconnecting {
    background: var(--warn);
  }
  .dot.error {
    background: var(--err);
  }
  .detail {
    color: var(--text-dim);
    font-family: var(--mono);
    font-size: 12px;
  }
  .counters {
    margin-left: auto;
    font-size: 12px;
    color: var(--text-dim);
  }
  .plugins-btn {
    font-size: 12px;
    color: var(--text-dim);
    background: var(--bg-hover);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 3px 10px;
    cursor: pointer;
  }
  .plugins-btn:hover {
    color: var(--text);
  }
  main {
    flex: 1;
    display: grid;
    grid-template-columns: 300px 1fr 380px;
    min-height: 0;
  }
  .left {
    border-right: 1px solid var(--border);
    background: var(--bg-panel);
    min-height: 0;
  }
  .center {
    min-width: 0;
    min-height: 0;
  }
  .right {
    border-left: 1px solid var(--border);
    background: var(--bg-panel);
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .detail-wrap {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
