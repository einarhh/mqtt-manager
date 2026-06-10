<script lang="ts">
  import { tree, totalTopics, totalMessages, clearTree } from "./stores";
  import TreeNode from "./TreeNode.svelte";

  let filter = "";

  $: rootChildren = [...$tree.children.values()].sort((a, b) =>
    a.name.localeCompare(b.name)
  );
</script>

<div class="tree-panel">
  <div class="toolbar">
    <input
      class="filter"
      type="text"
      placeholder="Filter topics…"
      bind:value={filter}
    />
    <button class="ghost" title="Clear all received topics" on:click={clearTree}>
      Clear
    </button>
  </div>

  <div class="stats">
    <span>{$totalTopics} topics</span>
    <span>·</span>
    <span>{$totalMessages} messages</span>
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
  .stats {
    display: flex;
    gap: 6px;
    padding: 6px 12px;
    font-size: 12px;
    color: var(--text-dim);
    border-bottom: 1px solid var(--border);
  }
  .tree-scroll {
    flex: 1;
    overflow: auto;
    padding: 6px 4px;
  }
  .empty {
    color: var(--text-dim);
    padding: 24px 12px;
    font-size: 13px;
    text-align: center;
  }
</style>
