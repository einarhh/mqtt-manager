<script lang="ts">
  import type { TreeNode } from "./stores";
  import { selectedPath } from "./stores";
  import { preview } from "./util";
  import Self from "./TreeNode.svelte";

  export let node: TreeNode;
  export let depth = 0;
  export let filter = "";

  let expanded = false;

  // Case-insensitive subtree match: a node is visible if it or any descendant
  // matches the filter text.
  function matches(n: TreeNode, f: string): boolean {
    if (!f) return true;
    if (n.path.toLowerCase().includes(f)) return true;
    for (const c of n.children.values()) {
      if (matches(c, f)) return true;
    }
    return false;
  }

  $: lowerFilter = filter.trim().toLowerCase();
  $: visible = matches(node, lowerFilter);
  // When filtering, force-expand so matches are revealed.
  $: showChildren = (expanded || !!lowerFilter) && node.children.size > 0;
  $: kids = [...node.children.values()]
    .filter((c) => matches(c, lowerFilter))
    .sort((a, b) => a.name.localeCompare(b.name));
  $: selected = $selectedPath === node.path;

  function toggle() {
    if (node.children.size > 0) expanded = !expanded;
  }
  function select() {
    selectedPath.set(node.path);
  }
  // Clicking anywhere on the row selects the topic and toggles its children.
  function onRowClick() {
    select();
    toggle();
  }
</script>

{#if visible}
  <div class="node">
    <div
      class="row"
      class:selected
      style="padding-left: {depth * 14 + 6}px"
      on:click={onRowClick}
    >
      <span class="twisty" class:hidden={node.children.size === 0}>
        {expanded || lowerFilter ? "▾" : "▸"}
      </span>
      <span class="name">{node.name || "/"}</span>
      {#if node.children.size > 0}
        <span class="badge">{node.children.size}</span>
      {/if}
      {#if node.text !== null}
        {#key node.count}
          <span class="value" class:retained={node.retained}>
            {preview(node.text)}
          </span>
        {/key}
      {/if}
    </div>
    {#if showChildren}
      {#each kids as child (child.path)}
        <Self node={child} depth={depth + 1} {filter} />
      {/each}
    {/if}
  </div>
{/if}

<style>
  .row {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 8px 3px 0;
    cursor: pointer;
    white-space: nowrap;
    border-radius: 4px;
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.selected {
    background: var(--accent-soft);
  }
  .twisty {
    width: 16px;
    flex: 0 0 16px;
    text-align: center;
    color: var(--text-dim);
    font-size: 14px;
    line-height: 1;
  }
  .twisty.hidden {
    visibility: hidden;
  }
  .name {
    color: var(--text);
    font-weight: 500;
  }
  .badge {
    font-size: 11px;
    color: var(--text-dim);
    background: var(--bg-hover);
    border-radius: 8px;
    padding: 0 6px;
  }
  .value {
    color: var(--accent);
    font-family: var(--mono);
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    animation: flash 0.6s ease-out;
  }
  .value.retained {
    color: var(--warn);
  }
  @keyframes flash {
    from {
      background: var(--accent-soft);
    }
    to {
      background: transparent;
    }
  }
</style>
