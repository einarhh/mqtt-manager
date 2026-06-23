<script lang="ts">
  import {
    tree,
    totalTopics,
    totalMessages,
    clearTree,
    sortMode,
    sortNodes,
  } from "./stores";
  import TreeNode from "./TreeNode.svelte";
  import Icon from "./Icon.svelte";

  let filter = "";

  $: rootChildren = sortNodes([...$tree.children.values()], $sortMode);

  const SORT_LABEL = {
    received: "Order: received — sort A–Z",
    alpha: "Order: A–Z — sort by received",
  } as const;

  function toggleSort() {
    sortMode.update((m) => (m === "alpha" ? "received" : "alpha"));
  }
</script>

<div class="tree-panel">
  <div class="toolbar">
    <input
      class="filter"
      type="text"
      placeholder="Filter topics…"
      bind:value={filter}
    />
    <button
      class="icon-btn"
      class:active={$sortMode === "alpha"}
      title={SORT_LABEL[$sortMode]}
      aria-label={SORT_LABEL[$sortMode]}
      on:click={toggleSort}
    >
      <Icon name="sort-alpha" size={15} />
    </button>
  </div>

  <div class="stats">
    <span>{$totalTopics} topics</span>
    <span>·</span>
    <span>{$totalMessages} messages</span>
    <button
      class="clear-btn"
      title="Clear all received topics"
      aria-label="Clear all received topics"
      disabled={$totalTopics === 0}
      on:click={clearTree}
    >
      <Icon name="trash" size={13} />
      <span>Clear</span>
    </button>
  </div>

  <div class="tree-scroll">
    {#if rootChildren.length === 0}
      <div class="empty">No messages yet. Connect and subscribe to a topic.</div>
    {:else}
      {#each rootChildren as child (child.path)}
        <TreeNode node={child} depth={0} {filter} />
      {/each}
    {/if}
  </div>
</div>

<style>
  .tree-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-width: 0;
  }
  .toolbar {
    display: flex;
    gap: 8px;
    padding: 10px;
    border-bottom: 1px solid var(--border);
  }
  .filter {
    flex: 1;
  }
  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-dim);
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 5px;
    cursor: pointer;
  }
  .icon-btn:hover {
    color: var(--text);
    background: var(--bg-hover);
    border-color: var(--border-strong);
  }
  .icon-btn.active {
    color: var(--accent);
    border-color: var(--accent);
    background: var(--accent-soft);
  }
  .stats {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    font-size: 12px;
    color: var(--text-dim);
    border-bottom: 1px solid var(--border);
  }
  /* Pushed to the right so the destructive "clear received data" action sits
     with the counts it affects, well clear of the filter input. */
  .clear-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
    color: var(--text-dim);
    background: transparent;
    border: none;
    padding: 2px 4px;
    font-size: 12px;
    border-radius: 5px;
    cursor: pointer;
  }
  .clear-btn:hover:not(:disabled) {
    color: var(--err);
    background: var(--bg-hover);
  }
  .clear-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .tree-scroll {
    flex: 1;
    /* Always show the vertical scrollbar (classic, via the styled
       ::-webkit-scrollbar) so it can't auto-hide or flicker in/out as messages
       stream in. overflow-x stays auto for the odd over-long row. */
    overflow-y: scroll;
    overflow-x: auto;
    padding: 6px 4px;
  }
  .empty {
    color: var(--text-dim);
    padding: 24px 12px;
    font-size: 13px;
    text-align: center;
  }
</style>
