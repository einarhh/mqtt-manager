<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { selectedPath } from "./stores";
  import { topicMatch } from "./util";
  import {
    pluginList,
    savePlugin,
    deletePlugin,
    setPluginEnabled,
    importPlugin,
    exportPlugin,
  } from "./plugins";
  import type { PluginInfo } from "./plugins";

  const dispatch = createEventDispatcher();

  const TEMPLATE = `// A decoder plugin: matched payloads are decoded for the detail panel.
// Define any helpers first, then a single trailing \`export default\`.
export default {
  name: "My decoder",
  topic: "my/topic/+",            // MQTT-style filter (+ and # wildcards)
  // Optional: return false to let another plugin handle the message.
  // match(ctx) { return ctx.bytes.length > 2; },
  decode(ctx) {
    // ctx = { topic, bytes: Uint8Array, hex, text, ts }
    return {
      summary: ctx.hex,
      fields: [
        { label: "length", value: ctx.bytes.length, hint: "bytes" },
      ],
    };
  },
};
`;

  let draft: PluginInfo | null = null;
  let error = "";

  function newPlugin() {
    error = "";
    draft = {
      id: "",
      name: "New decoder",
      filename: "",
      enabled: true,
      order: 0,
      source: TEMPLATE,
      topic: "#",
      loadError: "",
    };
  }

  function editPlugin(p: PluginInfo) {
    error = "";
    draft = { ...p };
  }

  async function save() {
    if (!draft) return;
    error = "";
    try {
      const saved = await savePlugin(draft);
      draft = { ...saved };
      // Pull the post-load status (parsed topic / compile error) back in.
      const fresh = $pluginList.find((p) => p.id === saved.id);
      if (fresh) draft = { ...fresh };
    } catch (e) {
      error = String(e);
    }
  }

  async function remove() {
    if (!draft?.id) {
      draft = null;
      return;
    }
    try {
      await deletePlugin(draft.id);
      draft = null;
    } catch (e) {
      error = String(e);
    }
  }

  async function toggle(p: PluginInfo, e: Event) {
    const enabled = (e.target as HTMLInputElement).checked;
    try {
      await setPluginEnabled(p, enabled);
      if (draft?.id === p.id) draft.enabled = enabled;
    } catch (err) {
      error = String(err);
    }
  }

  async function doImport() {
    error = "";
    try {
      const p = await importPlugin();
      if (p) {
        const fresh = $pluginList.find((x) => x.id === p.id);
        draft = { ...(fresh ?? p) };
      }
    } catch (e) {
      error = String(e);
    }
  }

  async function doExport() {
    if (!draft?.id) return;
    error = "";
    try {
      await exportPlugin(draft.id);
    } catch (e) {
      error = String(e);
    }
  }

  function close() {
    dispatch("close");
  }
</script>

<div class="overlay" on:click|self={close}>
  <div class="modal">
    <header>
      <span class="title">Decoder plugins</span>
      <button class="x" on:click={close}>✕</button>
    </header>

    <div class="body">
      <div class="list">
        <div class="list-head">
          <span>Installed</span>
          <div class="head-actions">
            <button class="link" on:click={doImport}>import</button>
            <button class="link" on:click={newPlugin}>+ new</button>
          </div>
        </div>
        {#if $pluginList.length === 0}
          <div class="hint">No plugins yet. Create one, or drop a .js file into the plugins folder.</div>
        {:else}
          <ul>
            {#each $pluginList as p (p.id)}
              <li class:active={draft?.id === p.id} on:click={() => editPlugin(p)}>
                <input
                  type="checkbox"
                  checked={p.enabled}
                  on:click|stopPropagation
                  on:change={(e) => toggle(p, e)}
                />
                <div class="pmeta">
                  <span class="pname">{p.name}</span>
                  <span class="ptopic">
                    {p.topic}
                    {#if p.enabled && $selectedPath && topicMatch(p.topic, $selectedPath)}
                      <span class="match">• matches selection</span>
                    {/if}
                  </span>
                  {#if p.loadError}<span class="perr">⚠ {p.loadError}</span>{/if}
                </div>
              </li>
            {/each}
          </ul>
        {/if}
      </div>

      <div class="editor">
        {#if !draft}
          <div class="hint center">Select a plugin to edit, or create a new one.</div>
        {:else}
          <label>
            Name
            <input bind:value={draft.name} />
          </label>
          {#if draft.filename}
            <div class="filerow">file: <code>{draft.filename}</code></div>
          {/if}
          <label class="grow">
            Source
            <textarea bind:value={draft.source} spellcheck="false"></textarea>
          </label>
          {#if draft.loadError}
            <div class="error">⚠ {draft.loadError}</div>
          {/if}
          {#if error}<div class="error">{error}</div>{/if}
          <div class="actions">
            <button class="ghost" on:click={remove}>{draft.id ? "Delete" : "Discard"}</button>
            {#if draft.id}
              <button class="ghost" on:click={doExport}>Export</button>
            {/if}
            <button class="primary" on:click={save}>Save</button>
          </div>
        {/if}
      </div>
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
    width: min(900px, 92vw);
    height: min(640px, 88vh);
    background: var(--bg-panel);
    border: 1px solid var(--border);
    border-radius: 10px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
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
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    font-size: 14px;
  }
  .body {
    flex: 1;
    display: grid;
    grid-template-columns: 280px 1fr;
    min-height: 0;
  }
  .list {
    border-right: 1px solid var(--border);
    overflow: auto;
    padding: 10px;
  }
  .list-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
    margin-bottom: 8px;
  }
  .head-actions {
    display: flex;
    gap: 10px;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    display: flex;
    gap: 8px;
    align-items: flex-start;
    padding: 8px;
    border-radius: 6px;
    cursor: pointer;
  }
  li:hover {
    background: var(--bg-hover);
  }
  li.active {
    background: var(--accent-soft);
  }
  .pmeta {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .pname {
    font-weight: 600;
    font-size: 13px;
  }
  .ptopic {
    font-size: 11px;
    color: var(--text-dim);
    font-family: var(--mono);
  }
  .match {
    color: var(--ok);
  }
  .perr {
    font-size: 11px;
    color: var(--warn);
  }
  .editor {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
    min-height: 0;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 12px;
    color: var(--text-dim);
  }
  label.grow {
    flex: 1;
    min-height: 0;
  }
  input,
  textarea {
    font-size: 13px;
  }
  textarea {
    flex: 1;
    min-height: 0;
    resize: none;
    font-family: var(--mono);
    font-size: 12px;
    line-height: 1.5;
    white-space: pre;
    overflow: auto;
  }
  .filerow {
    font-size: 11px;
    color: var(--text-dim);
  }
  .filerow code {
    font-family: var(--mono);
  }
  .actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
  .error {
    color: var(--err);
    font-size: 12px;
  }
  .hint {
    font-size: 12px;
    color: var(--text-dim);
  }
  .hint.center {
    margin: auto;
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
