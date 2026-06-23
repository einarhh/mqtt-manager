<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { EventsOn, EventsOff } from "../wailsjs/runtime/runtime";
  import { Version, Connections, ListProfiles } from "../wailsjs/go/main/App";
  import { ingest, setStatus, addConnection, status, connections, activeId } from "./lib/stores";
  import type { IncomingMessage } from "./lib/stores";
  import { reload as reloadPlugins } from "./lib/plugins";

  // Tagged event payloads emitted by the Go backend (ConnMessages / ConnState).
  interface MessagesEvent {
    id: string;
    messages: IncomingMessage[];
  }
  interface StatusEvent {
    id: string;
    status: string;
    detail: string;
  }

  let appVersion = "";
  let showPlugins = false;
  let showAbout = false;
  import Logo from "./lib/Logo.svelte";
  import Icon from "./lib/Icon.svelte";
  import ConnectionPanel from "./lib/ConnectionPanel.svelte";
  import TopicTree from "./lib/TopicTree.svelte";
  import TopicDetail from "./lib/TopicDetail.svelte";
  import PublishPanel from "./lib/PublishPanel.svelte";
  import PluginManager from "./lib/PluginManager.svelte";
  import About from "./lib/About.svelte";
  import { themeMode, cycleTheme } from "./lib/theme";

  const STATUS_LABELS: Record<string, string> = {
    disconnected: "Disconnected",
    connecting: "Connecting…",
    connected: "Connected",
    reconnecting: "Reconnecting…",
    error: "Error",
  };

  // The active connection's name, shown in the header. `status.detail` carries
  // the broker URL while connected (less useful than the name) and an error
  // message otherwise, so we surface detail only for error/reconnecting states.
  $: activeName = $connections.find((c) => c.id === $activeId)?.name ?? "";
  $: showDetail =
    !!$status.detail &&
    ($status.status === "error" || $status.status === "reconnecting");

  const THEME_ICON = { system: "monitor", light: "sun", dark: "moon" } as const;
  const THEME_LABEL = {
    system: "Theme: follow system",
    light: "Theme: light",
    dark: "Theme: dark",
  } as const;

  // Resizable columns: the left (connections) and right (detail) widths are drag-
  // adjustable via the gutters between panels and persisted across reloads. The
  // center column takes the remaining space.
  const LAYOUT_KEY = "mqtt-manager:layout";
  const MIN_COL = 220;
  const MAX_COL = 600;
  const MIN_CENTER = 240;

  function readLayout(): { left: number; right: number } {
    try {
      const v = JSON.parse(localStorage.getItem(LAYOUT_KEY) ?? "");
      return { left: Number(v.left) || 340, right: Number(v.right) || 380 };
    } catch {
      return { left: 340, right: 380 };
    }
  }
  let { left: leftW, right: rightW } = readLayout();
  $: localStorage.setItem(LAYOUT_KEY, JSON.stringify({ left: leftW, right: rightW }));

  let dragging: "left" | "right" | null = null;
  let startX = 0;
  let startW = 0;

  const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

  function startDrag(which: "left" | "right", e: PointerEvent): void {
    dragging = which;
    startX = e.clientX;
    startW = which === "left" ? leftW : rightW;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }
  function onDrag(e: PointerEvent): void {
    if (!dragging) return;
    const dx = e.clientX - startX;
    // Keep the center column at least MIN_CENTER wide by capping against the sibling.
    const avail = window.innerWidth - MIN_CENTER;
    if (dragging === "left") {
      leftW = clamp(startW + dx, MIN_COL, Math.min(MAX_COL, avail - rightW));
    } else {
      rightW = clamp(startW - dx, MIN_COL, Math.min(MAX_COL, avail - leftW));
    }
  }
  function endDrag(e: PointerEvent): void {
    if (!dragging) return;
    dragging = null;
    try {
      (e.currentTarget as HTMLElement).releasePointerCapture(e.pointerId);
    } catch {
      // capture may already be gone — ignore
    }
  }

  onMount(() => {
    // Clear any listeners left over from a previous mount (e.g. a dev hot-reload
    // while the Go backend keeps running) so each batch is ingested exactly once.
    EventsOff("mqtt:messages");
    EventsOff("mqtt:status");
    EventsOn("mqtt:messages", (p: MessagesEvent) => ingest(p.id, p.messages));
    EventsOn("mqtt:status", (p: StatusEvent) =>
      setStatus(p.id, { status: p.status, detail: p.detail }),
    );
    Version().then((v) => (appVersion = v));
    // Status is only pushed on transitions, so rebuild the live connections on load
    // (a reload would otherwise lose the connection list while messages still flow).
    Promise.all([Connections(), ListProfiles()]).then(([live, profs]) => {
      const names = new Map(profs.map((p) => [p.id, p.name]));
      for (const c of live) {
        addConnection(c.id, names.get(c.id) ?? c.id);
        setStatus(c.id, { status: c.status, detail: c.detail });
      }
    });
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
      {#if appVersion}
        <button class="version" title="About MQTT Manager" on:click={() => (showAbout = true)}>
          {appVersion}
        </button>
      {/if}
    </div>
    <div class="status">
      <span class="dot {$status.status}"></span>
      {#if activeName}<span class="conn-name">{activeName}</span>{/if}
      <span class="status-text">{STATUS_LABELS[$status.status] ?? $status.status}</span>
      {#if showDetail}<span class="detail">{$status.detail}</span>{/if}
    </div>
    <button
      class="icon-btn push-right"
      title={THEME_LABEL[$themeMode]}
      aria-label={THEME_LABEL[$themeMode]}
      on:click={cycleTheme}
    >
      <Icon name={THEME_ICON[$themeMode]} size={15} />
    </button>
    <button class="plugins-btn" on:click={() => (showPlugins = true)}>
      <Icon name="code" size={14} />
      Plugins
    </button>
  </header>

  {#if showPlugins}
    <PluginManager on:close={() => (showPlugins = false)} />
  {/if}

  {#if showAbout}
    <About version={appVersion} on:close={() => (showAbout = false)} />
  {/if}

  <main
    class:dragging
    style="grid-template-columns: {leftW}px 5px minmax(0, 1fr) 5px {rightW}px"
  >
    <aside class="left"><ConnectionPanel /></aside>
    <div
      class="gutter"
      class:active={dragging === "left"}
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize connection panel"
      title="Drag to resize"
      on:pointerdown={(e) => startDrag("left", e)}
      on:pointermove={onDrag}
      on:pointerup={endDrag}
      on:pointercancel={endDrag}
    ></div>
    <section class="center"><TopicTree /></section>
    <div
      class="gutter"
      class:active={dragging === "right"}
      role="separator"
      aria-orientation="vertical"
      aria-label="Resize detail panel"
      title="Drag to resize"
      on:pointerdown={(e) => startDrag("right", e)}
      on:pointermove={onDrag}
      on:pointerup={endDrag}
      on:pointercancel={endDrag}
    ></div>
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
    gap: 18px;
    height: 48px;
    padding: 0 14px;
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
    background: transparent;
    cursor: pointer;
  }
  .version:hover {
    color: var(--text);
    border-color: var(--border-strong);
    background: var(--bg-hover);
  }
  .status {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
  }
  .conn-name {
    font-weight: 600;
  }
  .status-text {
    color: var(--text-dim);
    font-size: 12px;
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
  /* Pushes the right-hand controls (theme, plugins) to the far edge now that the
     header no longer holds the topic/message counters — they live under the
     filter in the topic tree instead. */
  .push-right {
    margin-left: auto;
  }
  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-dim);
    background: transparent;
    border: 1px solid transparent;
    border-radius: 6px;
    padding: 5px;
    cursor: pointer;
  }
  .icon-btn:hover {
    color: var(--text);
    background: var(--bg-hover);
    border-color: var(--border);
  }
  .plugins-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-dim);
    background: var(--bg-hover);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 4px 10px;
    cursor: pointer;
  }
  .plugins-btn:hover {
    color: var(--text);
    border-color: var(--border-strong);
  }
  main {
    flex: 1;
    display: grid;
    /* grid-template-columns is set inline so the gutters can resize the columns. */
    min-height: 0;
  }
  main.dragging {
    cursor: col-resize;
    user-select: none;
  }
  /* During a drag, stop panel content (notably the map iframe) from swallowing
     the pointer moves; the gutter keeps pointer capture regardless. */
  main.dragging .left,
  main.dragging .center,
  main.dragging .right {
    pointer-events: none;
  }
  .left {
    background: var(--bg-panel);
    min-height: 0;
  }
  .center {
    min-width: 0;
    min-height: 0;
  }
  .right {
    background: var(--bg-panel);
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  /* A wide-enough grab target with a thin centred divider line that highlights
     on hover/drag. Replaces the panels' old static border lines. */
  .gutter {
    position: relative;
    cursor: col-resize;
    background: transparent;
  }
  .gutter::before {
    content: "";
    position: absolute;
    top: 0;
    bottom: 0;
    left: 50%;
    width: 1px;
    transform: translateX(-50%);
    background: var(--border);
  }
  .gutter:hover::before,
  .gutter.active::before {
    background: var(--accent);
    width: 2px;
  }
  .detail-wrap {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
