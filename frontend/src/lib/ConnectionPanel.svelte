<script lang="ts">
  import { onMount } from "svelte";
  import { status, clearTree } from "./stores";
  import {
    Connect,
    Disconnect,
    Subscribe,
    ListProfiles,
    SaveProfile,
    DeleteProfile,
  } from "../../wailsjs/go/main/App";
  import type { profiles as models } from "../../wailsjs/go/models";
  import Icon from "./Icon.svelte";

  // The wails-generated ConnectionProfile is a class with helper methods; our
  // local Profile is a plain object, so cast at the serialization boundary.
  const asProfile = (p: Profile) =>
    p as unknown as models.ConnectionProfile;

  interface Subscription {
    filter: string;
    qos: number;
  }

  interface Profile {
    id: string;
    name: string;
    host: string;
    port: number;
    useTls: boolean;
    tlsInsecure: boolean;
    caCertPath: string;
    clientId: string;
    username: string;
    password: string;
    keepAlive: number;
    subscriptions: Subscription[];
    // Legacy single-topic fields, present on profiles saved before multi-topic.
    subFilter?: string;
    subQos?: number;
  }

  let profiles: Profile[] = [];
  let draft: Profile = blank();
  let error = "";
  // The editor form is hidden by default; the panel shows just the saved list.
  // It opens for New / editing, and stays open while a connection is live so the
  // Disconnect control remains reachable.
  let editing = false;

  $: connected = $status.status === "connected";
  $: busy = $status.status === "connecting";
  $: showForm = editing || connected || busy;

  function randomClientId(): string {
    return "mqtt-manager-" + Math.random().toString(16).slice(2, 8);
  }

  function blank(): Profile {
    return {
      id: "",
      name: "New connection",
      host: "localhost",
      port: 1883,
      useTls: false,
      tlsInsecure: false,
      caCertPath: "",
      clientId: randomClientId(),
      username: "",
      password: "",
      keepAlive: 30,
      subscriptions: [{ filter: "#", qos: 0 }],
    };
  }

  // Old profiles store a single subFilter/subQos; fold them into the list so the
  // editor always works with subscriptions[].
  function normalize(p: Profile): Profile {
    const subs =
      p.subscriptions && p.subscriptions.length > 0
        ? p.subscriptions
        : p.subFilter
          ? [{ filter: p.subFilter, qos: p.subQos ?? 0 }]
          : [];
    return { ...p, subscriptions: subs, subFilter: undefined, subQos: undefined };
  }

  // Keep the default port in sync with the TLS toggle while it's untouched.
  function onToggleTls() {
    if (!draft.useTls && draft.port === 8883) draft.port = 1883;
    if (draft.useTls && draft.port === 1883) draft.port = 8883;
  }

  async function loadProfiles() {
    try {
      profiles = (await ListProfiles()) as Profile[];
    } catch (e) {
      error = String(e);
    }
  }

  function newProfile() {
    draft = blank();
    error = "";
    editing = true;
  }

  function edit(p: Profile) {
    draft = normalize({ ...p });
    error = "";
    editing = true;
  }

  // Close the editor back to the bare connection list.
  function cancel() {
    editing = false;
    error = "";
    draft = blank();
  }

  function addSubscription() {
    draft.subscriptions = [...draft.subscriptions, { filter: "", qos: 0 }];
  }

  function removeSubscription(i: number) {
    draft.subscriptions = draft.subscriptions.filter((_, idx) => idx !== i);
  }

  async function save() {
    error = "";
    try {
      const saved = (await SaveProfile(asProfile(draft))) as Profile;
      draft = normalize({ ...saved });
      await loadProfiles();
    } catch (e) {
      error = String(e);
    }
  }

  async function remove() {
    if (!draft.id) return;
    try {
      await DeleteProfile(draft.id);
      draft = blank();
      editing = false;
      await loadProfiles();
    } catch (e) {
      error = String(e);
    }
  }

  async function connect() {
    error = "";
    try {
      clearTree();
      await Connect(asProfile(draft));
      const subs = draft.subscriptions.filter((s) => s.filter.trim() !== "");
      if (subs.length === 0) {
        await Subscribe("#", 0);
      } else {
        for (const s of subs) {
          await Subscribe(s.filter.trim(), Number(s.qos));
        }
      }
    } catch (e) {
      error = String(e);
    }
  }

  async function disconnect() {
    try {
      await Disconnect();
    } catch (e) {
      error = String(e);
    }
  }

  onMount(loadProfiles);
</script>

<div class="panel">
  <div class="saved">
    <div class="row-head">
      <span class="title">Connections</span>
      <button class="link" on:click={newProfile}>
        <Icon name="plus" size={13} />
        New
      </button>
    </div>
    {#if profiles.length === 0}
      <div class="hint">No saved connections yet.</div>
    {:else}
      <ul>
        {#each profiles as p (p.id)}
          <li class:active={p.id === draft.id} on:click={() => edit(p)}>
            <span class="pname">{p.name}</span>
            <span class="paddr">{p.host}:{p.port}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  {#if showForm}
  <div class="form">
    <div class="form-head">
      <span class="title">{draft.id ? "Edit connection" : "New connection"}</span>
      {#if !connected && !busy}
        <button class="icon-btn" title="Close" on:click={cancel}>
          <Icon name="close" size={14} />
        </button>
      {/if}
    </div>
    <label>Name<input bind:value={draft.name} /></label>
    <div class="two">
      <label class="grow">Host<input bind:value={draft.host} /></label>
      <label class="port">Port<input type="number" bind:value={draft.port} /></label>
    </div>

    <div class="two">
      <label class="check">
        <input type="checkbox" bind:checked={draft.useTls} on:change={onToggleTls} />
        TLS
      </label>
      {#if draft.useTls}
        <label class="check">
          <input type="checkbox" bind:checked={draft.tlsInsecure} />
          Skip verify
        </label>
      {/if}
    </div>
    {#if draft.useTls}
      <label>CA cert path (optional)<input bind:value={draft.caCertPath} /></label>
    {/if}

    <label>Client ID<input bind:value={draft.clientId} /></label>
    <label>Username<input bind:value={draft.username} /></label>
    <label>Password<input type="password" bind:value={draft.password} /></label>

    <div class="subs">
      <div class="subs-head">
        <span>Subscriptions</span>
        <button class="link" on:click={addSubscription}>
          <Icon name="plus" size={13} />
          Add topic
        </button>
      </div>
      {#if draft.subscriptions.length === 0}
        <div class="hint">No topics — defaults to <code>#</code> on connect.</div>
      {/if}
      {#each draft.subscriptions as sub, i (i)}
        <div class="sub-row">
          <input class="grow" placeholder="topic/filter/#" bind:value={sub.filter} />
          <select class="qos" bind:value={sub.qos}>
            <option value={0}>0</option>
            <option value={1}>1</option>
            <option value={2}>2</option>
          </select>
          <button
            class="icon-btn"
            title="Remove topic"
            on:click={() => removeSubscription(i)}
          >
            <Icon name="trash" size={14} />
          </button>
        </div>
      {/each}
    </div>

    <div class="actions">
      <button class="ghost" on:click={save}>Save</button>
      <button class="ghost" on:click={remove} disabled={!draft.id}>Delete</button>
      {#if connected}
        <button class="danger" on:click={disconnect}>Disconnect</button>
      {:else}
        <button class="primary" on:click={connect} disabled={busy}>
          {busy ? "Connecting…" : "Connect"}
        </button>
      {/if}
    </div>

    {#if error}<div class="error">{error}</div>{/if}
  </div>
  {/if}
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: auto;
  }
  .saved {
    padding: 10px 12px;
    border-bottom: 1px solid var(--border);
  }
  .row-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }
  .title {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-dim);
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  li {
    display: flex;
    flex-direction: column;
    padding: 6px 8px;
    border-radius: 6px;
    cursor: pointer;
  }
  li:hover {
    background: var(--bg-hover);
  }
  li.active {
    background: var(--accent-soft);
  }
  .pname {
    font-weight: 600;
    font-size: 13px;
  }
  .paddr {
    font-size: 12px;
    color: var(--text-dim);
    font-family: var(--mono);
  }
  .hint {
    font-size: 12px;
    color: var(--text-dim);
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px;
  }
  .form-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 2px;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 12px;
    color: var(--text-dim);
  }
  label.check {
    flex-direction: row;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--text);
  }
  .two {
    display: flex;
    gap: 8px;
  }
  .grow {
    flex: 1;
  }
  .port {
    width: 76px;
  }
  .subs {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .subs-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    color: var(--text-dim);
  }
  .sub-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .sub-row .grow {
    flex: 1;
    min-width: 0;
  }
  .sub-row .qos {
    width: 52px;
  }
  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: var(--text-dim);
    cursor: pointer;
    padding: 4px;
    border-radius: 6px;
  }
  .icon-btn:hover {
    color: var(--err);
    background: var(--bg-hover);
  }
  .subs code {
    font-family: var(--mono);
  }
  .actions {
    display: flex;
    gap: 8px;
    margin-top: 4px;
  }
  .actions .primary,
  .actions .danger {
    margin-left: auto;
  }
  .link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: none;
    border: none;
    color: var(--accent);
    cursor: pointer;
    font-size: 12px;
    padding: 0;
  }
  .link:hover {
    color: var(--accent-strong);
  }
  .error {
    color: var(--err);
    font-size: 12px;
  }
</style>
