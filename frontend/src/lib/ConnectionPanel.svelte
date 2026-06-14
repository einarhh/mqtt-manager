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
  import Icon from "./Icon.svelte";

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
    subFilter: string;
    subQos: number;
  }

  let profiles: Profile[] = [];
  let draft: Profile = blank();
  let error = "";

  $: connected = $status.status === "connected";
  $: busy = $status.status === "connecting";

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
      subFilter: "#",
      subQos: 0,
    };
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
  }

  function edit(p: Profile) {
    draft = { ...p };
  }

  async function save() {
    error = "";
    try {
      const saved = (await SaveProfile(draft)) as Profile;
      draft = { ...saved };
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
      await loadProfiles();
    } catch (e) {
      error = String(e);
    }
  }

  async function connect() {
    error = "";
    try {
      clearTree();
      await Connect(draft);
      await Subscribe(draft.subFilter || "#", Number(draft.subQos));
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

  <div class="form">
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

    <div class="two">
      <label class="grow">Subscribe<input bind:value={draft.subFilter} /></label>
      <label class="port">
        QoS
        <select bind:value={draft.subQos}>
          <option value={0}>0</option>
          <option value={1}>1</option>
          <option value={2}>2</option>
        </select>
      </label>
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
