<script lang="ts">
  import { tree, selectedPath, findNode } from "./stores";
  import type { TreeNode, HistoryEntry } from "./stores";
  import { prettyJSON, formatTime, parseCoord, mapURL, mapEmbedURL } from "./util";
  import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
  import { decodeRaw, pluginList } from "./plugins";
  import type { DecodeResult } from "./plugins";

  // Re-resolve whenever the tree updates or the selection changes.
  $: node = ($tree, findNode($selectedPath));
  // Newest first for the history list.
  $: history = node ? [...node.history].reverse() : [];

  // A specific history message the user pinned for inspection (null = live/latest).
  let viewing: HistoryEntry | null = null;
  let viewingPath: string | null = null;
  // Reset the pin when the selected topic changes.
  $: if (node && node.path !== viewingPath) {
    viewingPath = node.path;
    viewing = null;
  }

  // The payload currently shown: a pinned history entry, or the node's latest.
  $: sample = node
    ? viewing ?? { raw: node.raw, text: node.text, ts: node.ts, qos: node.qos, retained: node.retained }
    : null;
  $: parsed = sample && sample.text !== null ? prettyJSON(sample.text) : null;
  // If the payload looks like a GPS coordinate, offer an embedded map.
  $: coord = parseCoord(sample?.text);
  let showMap = false;
  let mapExpanded = false;

  // Decode the shown payload. Re-runs on selection, new message, pin change,
  // and when the plugin set changes ($pluginList).
  let decoded: DecodeResult | null = null;
  let decodeSeq = 0;
  let showRaw = false;
  $: refreshDecode(node, sample, $pluginList);

  async function refreshDecode(
    n: TreeNode | null,
    s: { raw: string | null; ts: number | null } | null,
    _plugins: unknown,
  ): Promise<void> {
    const seq = ++decodeSeq;
    let result: DecodeResult | null = null;
    if (n && s && s.raw !== null) {
      const key = viewing ? `${n.path}:h:${s.ts}` : `${n.path}:${n.count}`;
      result = await decodeRaw(n.path, s.raw, key, s.ts);
    }
    if (seq === decodeSeq) decoded = result;
  }

  function view(h: HistoryEntry): void {
    // Pin this entry; if it's the newest one, treat as live.
    viewing = history.length && h === history[0] ? null : h;
  }
  function backToLive(): void {
    viewing = null;
  }
  function isActive(h: HistoryEntry, i: number): boolean {
    return viewing ? viewing === h : i === 0;
  }

  // Show the plugin view unless it errored or the user toggled to raw.
  $: usePlugin = !!decoded && !decoded.error && !showRaw;

  function fmtValue(v: unknown): string {
    if (v === null || v === undefined) return "";
    if (typeof v === "object") return JSON.stringify(v);
    return String(v);
  }
</script>

<div class="detail">
  {#if !node}
    <div class="empty">Select a topic to inspect its value and history.</div>
  {:else}
    <div class="topic" title={node.path}>{node.path}</div>

    <div class="meta">
      <span class="chip">QoS {sample?.qos ?? node.qos}</span>
      {#if sample?.retained}<span class="chip warn">retained</span>{/if}
      <span class="chip">{node.count} msgs</span>
      {#if sample?.ts}<span class="chip">{formatTime(sample.ts)}</span>{/if}
      {#if decoded && !decoded.error}
        <button
          class="chip plugin"
          class:active={usePlugin}
          title="Decoded by the {decoded.pluginName} plugin — click to toggle raw"
          on:click={() => (showRaw = !showRaw)}
        >
          ⚙ {decoded.pluginName}{showRaw ? " (raw)" : ""}
        </button>
      {:else if parsed?.isJSON}
        <span class="chip ok">JSON</span>
      {/if}
      {#if coord}
        <button
          class="chip map"
          class:active={showMap}
          title="Show {coord.lat}, {coord.lon} on a map"
          on:click={() => (showMap = !showMap)}
        >
          📍 map
        </button>
      {/if}
    </div>

    {#if coord && showMap}
      {@const c = coord}
      <div class="map-panel" class:expanded={mapExpanded}>
        <iframe
          class="map-frame"
          title="Map of {c.lat}, {c.lon}"
          src={mapEmbedURL(c, mapExpanded ? 0.02 : 0.01)}
          loading="lazy"
        ></iframe>
        <div class="map-bar">
          <span class="map-coord">{c.lat}, {c.lon}</span>
          <span class="map-actions">
            <button class="link" on:click={() => (mapExpanded = !mapExpanded)}>
              {mapExpanded ? "− collapse" : "⤢ expand"}
            </button>
            <button class="link" on:click={() => BrowserOpenURL(mapURL(c))}>
              open in browser ↗
            </button>
          </span>
        </div>
      </div>
    {/if}

    {#if node.text === null}
      <div class="empty small">This is a branch with no direct value.</div>
    {:else}
      {#if viewing}
        <div class="viewing">
          Viewing history message{sample?.ts ? " · " + formatTime(sample.ts) : ""}
          <button class="link" on:click={backToLive}>← back to latest</button>
        </div>
      {/if}
      {#if decoded?.error}
        <div class="plugin-error">⚠ {decoded.pluginName}: {decoded.error}</div>
      {/if}

      <div class="content">
        {#key (viewing ? "h" + sample?.ts : "c" + node.count) + ":" + usePlugin}
          {#if usePlugin && decoded}
            <div class="decoded">
              {#if decoded.summary}<div class="summary">{decoded.summary}</div>{/if}
              {#if decoded.fields && decoded.fields.length}
                <table class="fields">
                  <tbody>
                    {#each decoded.fields as f}
                      <tr>
                        <td class="flabel">{f.label}</td>
                        <td class="fvalue">
                          {fmtValue(f.value)}{#if f.hint}<span class="fhint"> {f.hint}</span>{/if}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              {/if}
              {#if decoded.json !== undefined && decoded.json !== null}
                <pre class="value sub">{JSON.stringify(decoded.json, null, 2)}</pre>
              {/if}
              {#if decoded.text}<pre class="value sub">{decoded.text}</pre>{/if}
            </div>
          {:else}
            <pre class="value">{parsed?.pretty}</pre>
          {/if}
        {/key}
      </div>

      <div class="section-title">History</div>
      <div class="history">
        {#each history as h, i (h.ts + ":" + i)}
          <div
            class="hrow clickable"
            class:active={isActive(h, i)}
            on:click={() => view(h)}
            title="Click to decode this message"
          >
            <span class="htime">{formatTime(h.ts)}</span>
            <span class="htext">{h.text}</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .detail {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    padding: 12px;
  }
  .topic {
    font-family: var(--mono);
    font-size: 13px;
    color: var(--accent);
    word-break: break-all;
    margin-bottom: 8px;
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 10px;
  }
  .chip {
    font-size: 11px;
    color: var(--text-dim);
    background: var(--bg-hover);
    border-radius: 8px;
    padding: 2px 8px;
  }
  .chip.warn {
    color: var(--warn);
  }
  .chip.ok {
    color: var(--ok);
  }
  .chip.plugin {
    border: 1px solid var(--border);
    cursor: pointer;
    font-family: inherit;
  }
  .chip.plugin.active {
    color: var(--accent);
    border-color: var(--accent);
  }
  .chip.map {
    border: 1px solid var(--border);
    cursor: pointer;
    font-family: inherit;
  }
  .chip.map:hover,
  .chip.map.active {
    color: var(--accent);
    border-color: var(--accent);
  }
  .map-panel {
    border: 1px solid var(--border);
    border-radius: 6px;
    overflow: hidden;
    margin-bottom: 10px;
  }
  .map-frame {
    display: block;
    width: 100%;
    height: 160px;
    border: 0;
    background: var(--bg-inset);
  }
  .map-panel.expanded .map-frame {
    height: 320px;
  }
  .map-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 4px 8px;
    font-size: 12px;
    background: var(--bg-inset);
    border-top: 1px solid var(--border);
  }
  .map-coord {
    font-family: var(--mono);
    color: var(--text-dim);
  }
  .map-actions {
    display: flex;
    gap: 12px;
  }
  .plugin-error {
    font-size: 12px;
    color: var(--warn);
    background: var(--bg-inset);
    border: 1px solid var(--warn);
    border-radius: 6px;
    padding: 6px 10px;
    margin-bottom: 10px;
  }
  .viewing {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--accent);
    margin-bottom: 8px;
  }
  /* The decoded/raw value region gets the main, scrollable area. */
  .content {
    flex: 1;
    min-height: 0;
    overflow: auto;
    margin: 0 0 10px;
  }
  .decoded {
    animation: flash 0.6s ease-out;
  }
  .summary {
    font-size: 13px;
    color: var(--text);
    margin-bottom: 8px;
  }
  .fields {
    width: 100%;
    border-collapse: collapse;
    font-family: var(--mono);
    font-size: 12px;
  }
  .fields td {
    padding: 2px 8px 2px 0;
    vertical-align: top;
    border-bottom: 1px solid var(--border);
  }
  .flabel {
    color: var(--text-dim);
    white-space: nowrap;
    width: 1%;
  }
  .fvalue {
    color: var(--text);
    word-break: break-word;
  }
  .fhint {
    color: var(--text-dim);
  }
  .value {
    background: var(--bg-inset);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    font-family: var(--mono);
    font-size: 12px;
    color: var(--text);
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0;
    animation: flash 0.6s ease-out;
  }
  .value.sub {
    margin: 8px 0 0;
    animation: none;
  }
  @keyframes flash {
    from {
      border-color: var(--accent);
    }
    to {
      border-color: var(--border);
    }
  }
  .section-title {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
    margin-bottom: 6px;
  }
  .history {
    flex: 0 0 auto;
    max-height: 34%;
    overflow: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
  }
  .hrow {
    display: flex;
    gap: 8px;
    padding: 4px 8px;
    border-bottom: 1px solid var(--border);
    font-size: 12px;
  }
  .hrow:last-child {
    border-bottom: none;
  }
  .hrow.clickable {
    cursor: pointer;
  }
  .hrow.clickable:hover {
    background: var(--bg-hover);
  }
  .hrow.active {
    background: var(--accent-soft);
  }
  .htime {
    flex: 0 0 auto;
    color: var(--text-dim);
    font-family: var(--mono);
  }
  .htext {
    font-family: var(--mono);
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .empty {
    color: var(--text-dim);
    font-size: 13px;
    padding: 24px 8px;
    text-align: center;
  }
  .empty.small {
    padding: 12px 8px;
  }
  .link {
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 12px;
    padding: 0;
  }
</style>
