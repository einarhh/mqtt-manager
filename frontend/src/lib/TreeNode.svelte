<script lang="ts">
  import type { TreeNode } from "./stores";
  import { selectedPath, sortMode, sortNodes } from "./stores";
  import { preview, flashOn } from "./util";
  import Self from "./TreeNode.svelte";
  import Icon from "./Icon.svelte";

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
  $: kids = sortNodes(
    [...node.children.values()].filter((c) => matches(c, lowerFilter)),
    $sortMode
  );
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
      <span
        class="pulse"
        aria-hidden="true"
        use:flashOn={{ trigger: node.activity, animation: "tree-pulse 0.9s ease-out forwards" }}
      ></span>
      <span
        class="twisty"
        class:hidden={node.children.size === 0}
        class:open={expanded || !!lowerFilter}
      >
        <Icon name="chevron" size={13} stroke={2} />
      </span>
      <span class="name">{node.name || "/"}</span>
      {#if node.children.size > 0}
        <span class="badge">{node.children.size}</span>
      {/if}
      {#if node.text !== null}
        <span
          class="value"
          class:retained={node.retained}
          use:flashOn={{ trigger: node.count, animation: "tree-value-flash 0.6s ease-out" }}
        >
          {preview(node.text)}
        </span>
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
    position: relative;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 8px 3px 0;
    cursor: pointer;
    white-space: nowrap;
    border-radius: 4px;
  }
  /* A short accent pulse in the indent gutter when a message lands on this topic
     (or, via the propagated activity counter, anywhere beneath it). Absolutely
     positioned so it never shifts the row and fades out on its own. */
  .pulse {
    position: absolute;
    left: 1px;
    top: 50%;
    width: 4px;
    height: 4px;
    margin-top: -2px;
    border-radius: 50%;
    background: var(--accent);
    pointer-events: none;
    /* Always mounted but idle until flashOn replays the `pulse` animation, so the
       node never has to be remounted (which would churn the scroll bar). */
    opacity: 0;
  }
  /* Global name so the inline animation set by the flashOn action resolves —
     Svelte would otherwise scope (rename) these keyframes and the inline
     `animation: tree-pulse` reference wouldn't match. */
  @keyframes -global-tree-pulse {
    from {
      opacity: 0.9;
      transform: scale(1.5);
    }
    to {
      opacity: 0;
      transform: scale(1);
    }
  }
  .row:hover {
    background: var(--bg-hover);
  }
  .row.selected {
    background: var(--accent-soft);
  }
  .twisty {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    flex: 0 0 16px;
    color: var(--text-dim);
    transition: transform 0.12s ease;
  }
  .twisty.open {
    transform: rotate(90deg);
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
    /* Flash is replayed in place by the flashOn action (see the pulse note) so the
       value never remounts on a new message. */
  }
  .value.retained {
    color: var(--warn);
  }
  @keyframes -global-tree-value-flash {
    from {
      background: var(--accent-soft);
    }
    to {
      background: transparent;
    }
  }
</style>
