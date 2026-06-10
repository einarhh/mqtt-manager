<script lang="ts">
  import { tree, selectedPath, findNode } from "./stores";
  import { prettyJSON, formatTime } from "./util";

  // Re-resolve whenever the tree updates or the selection changes.
  $: node = ($tree, findNode($selectedPath));
  $: parsed = node && node.text !== null ? prettyJSON(node.text) : null;
  // Newest first for the history list.
  $: history = node ? [...node.history].reverse() : [];
</script>

<div class="detail">
  {#if !node}
    <div class="empty">Select a topic to inspect its value and history.</div>
  {:else}
    <div class="topic" title={node.path}>{node.path}</div>

    <div class="meta">
      <span class="chip">QoS {node.qos}</span>
      {#if node.retained}<span class="chip warn">retained</span>{/if}
      <span class="chip">{node.count} msgs</span>
      {#if node.ts}<span class="chip">{formatTime(node.ts)}</span>{/if}
      {#if parsed?.isJSON}<span class="chip ok">JSON</span>{/if}
    </div>

    {#if node.text === null}
      <div class="empty small">This is a branch with no direct value.</div>
    {:else}
      {#key node.count}
        <pre class="value">{parsed?.pretty}</pre>
      {/key}

      <div class="section-title">History</div>
      <div class="history">
        {#each history as h, i (h.ts + ":" + i)}
          <div class="hrow">
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
  .value {
    background: var(--bg-inset);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    font-family: var(--mono);
    font-size: 12px;
    color: var(--text);
    max-height: 38%;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
    margin: 0 0 10px;
    animation: flash 0.6s ease-out;
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
    flex: 1;
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
</style>
