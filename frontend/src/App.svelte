<script lang="ts">
  import { onMount } from "svelte";
  import { EventsOn } from "../wailsjs/runtime/runtime";
  import { ingest, status, totalTopics, totalMessages } from "./lib/stores";
  import type { IncomingMessage, ConnStatus } from "./lib/stores";
  import ConnectionPanel from "./lib/ConnectionPanel.svelte";
  import TopicTree from "./lib/TopicTree.svelte";
  import TopicDetail from "./lib/TopicDetail.svelte";
  import PublishPanel from "./lib/PublishPanel.svelte";

  const STATUS_LABELS: Record<string, string> = {
    disconnected: "Disconnected",
    connecting: "Connecting…",
    connected: "Connected",
    reconnecting: "Reconnecting…",
    error: "Error",
  };

  onMount(() => {
    EventsOn("mqtt:messages", (batch: IncomingMessage[]) => ingest(batch));
    EventsOn("mqtt:status", (s: ConnStatus) => status.set(s));
  });
</script>

<div class="app">
  <header>
    <div class="brand">
      <span class="logo">▤</span>
      <span class="app-name">MQTT Manager</span>
    </div>
    <div class="status">
      <span class="dot {$status.status}"></span>
      <span class="status-text">{STATUS_LABELS[$status.status] ?? $status.status}</span>
      {#if $status.detail}<span class="detail">{$status.detail}</span>{/if}
    </div>
    <div class="counters">
      {$totalTopics} topics · {$totalMessages} messages
    </div>
  </header>

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
  .logo {
    color: var(--accent);
    font-size: 18px;
  }
  .app-name {
    font-weight: 700;
    letter-spacing: 0.02em;
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
