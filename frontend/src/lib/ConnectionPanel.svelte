<script lang="ts">
  import { onMount } from "svelte";
  import {
    connections,
    activeId,
    addConnection,
    removeConnection,
    setActive,
    clearConnection,
  } from "./stores";
  import {
    Connect,
    Disconnect,
    Subscribe,
    RemoveConnection,
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

  // Statuses that count as "has a live connection" (Connect → Disconnect toggle).
  const LIVE = new Set(["connected", "connecting", "reconnecting"]);

  // Look up live connection state by profile id (drives the per-row status dot
  // and Connect/Disconnect/Remove controls).
  $: connMap = new Map($connections.map((c) => [c.id, c] as const));
  $: draftStatus = (draft.id ? connMap.get(draft.id)?.status.status : undefined) ?? "disconnected";
  $: draftLive = LIVE.has(draftStatus);
  $: busy = draftStatus === "connecting";
  // The editor is shown only via New / Edit; viewing a connection hides it.
  $: showForm = editing;

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
      if (connMap.has(draft.id)) {
        await RemoveConnection(draft.id);
        removeConnection(draft.id);
      }
      await DeleteProfile(draft.id);
      draft = blank();
      editing = false;
      await loadProfiles();
    } catch (e) {
      error = String(e);
    }
  }

  // Open (or reopen) a connection for a profile. Unsaved drafts are saved first so
  // they get a stable id, which doubles as the connection id.
  async function connectProfile(p: Profile) {
    error = "";
    try {
      let prof = normalize({ ...p });
      if (!prof.id) {
        prof = normalize((await SaveProfile(asProfile(prof))) as Profile);
        if (!draft.id) draft = prof; // reflect the new id back into the editor
        await loadProfiles();
      }
      const id = prof.id;
      addConnection(id, prof.name);
      clearConnection(id); // reset any data captured in a previous session
      setActive(id);
      await Connect(asProfile(prof));
      const subs = prof.subscriptions.filter((s) => s.filter.trim() !== "");
      if (subs.length === 0) {
        await Subscribe(id, "#", 0);
      } else {
        for (const s of subs) {
          await Subscribe(id, s.filter.trim(), Number(s.qos));
        }
      }
    } catch (e) {
      error = String(e);
    }
  }

  async function disconnectId(id: string) {
    try {
      await Disconnect(id);
    } catch (e) {
      error = String(e);
    }
  }

  // Forget a disconnected connection entirely (discards its captured tree).
  async function removeId(id: string) {
    try {
      await RemoveConnection(id);
      removeConnection(id);
    } catch (e) {
      error = String(e);
    }
  }

  // Clicking a row views its connection (if any) and closes the editor. New/Edit
  // are the only ways to open the editor.
  function selectRow(p: Profile) {
    editing = false;
    if (connMap.has(p.id)) setActive(p.id);
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
          {@const st = connMap.get(p.id)?.status.status ?? "none"}
          {@const live = LIVE.has(st)}
          <li class:active={p.id === $activeId} on:click={() => selectRow(p)}>
            <span class="dot {st}" title={st}></span>
            <div class="pinfo">
              <span class="pname">{p.name}</span>
              <span class="paddr">{p.host}:{p.port}</span>
            </div>
            {#if !live}
              <button
                class="connect-btn"
                title="Connect to this broker"
                on:click|stopPropagation={() => connectProfile(p)}
              >
                Connect
              </button>
            {/if}
            <div class="prow-actions">
              {#if live}
                <button
                  class="icon-btn act"
                  title="Disconnect"
                  on:click|stopPropagation={() => disconnectId(p.id)}
                >
                  <Icon name="power" size={14} />
                </button>
              {:else if connMap.has(p.id)}
                <button
                  class="icon-btn act"
                  title="Remove (discard captured data)"
                  on:click|stopPropagation={() => removeId(p.id)}
                >
                  <Icon name="close" size={14} />
                </button>
              {/if}
              <button
                class="icon-btn act"
                title="Edit"
                on:click|stopPropagation={() => edit(p)}
              >
                <Icon name="edit" size={14} />
              </button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  {#if showForm}
  <div class="form">
    <div class="form-head">
      <span class="title">{draft.id ? "Edit connection" : "New connection"}</span>
      <button class="icon-btn" title="Close" on:click={cancel}>
        <Icon name="close" size={14} />
      </button>
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
      {#if draftLive}
        <button class="danger" on:click={() => disconnectId(draft.id)}>
          {busy ? "Connecting…" : "Disconnect"}
        </button>
      {:else}
        <button class="primary" on:click={() => connectProfile(draft)}>Connect</button>
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
    align-items: center;
    gap: 8px;
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
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    flex: 0 0 auto;
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
  .pinfo {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-width: 0;
  }
  .prow-actions {
    display: flex;
    align-items: center;
    gap: 1px;
    opacity: 0;
  }
  li:hover .prow-actions,
  li.active .prow-actions {
    opacity: 1;
  }
  /* Row-action buttons are neutral on hover (the shared .icon-btn turns red,
     which only suits the destructive subscription-remove button). */
  .prow-actions .icon-btn.act:hover {
    color: var(--text);
  }
  /* Disconnected rows show an always-visible, labelled Connect button so the
     action is obvious without hovering or decoding an icon. */
  .connect-btn {
    flex: 0 0 auto;
    padding: 2px 10px;
    font-size: 12px;
    border: 1px solid var(--accent);
    background: transparent;
    color: var(--accent);
    border-radius: 6px;
  }
  .connect-btn:hover {
    background: var(--accent);
    border-color: var(--accent);
    color: var(--on-accent);
  }
  .pname {
    font-weight: 600;
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .paddr {
    font-size: 12px;
    color: var(--text-dim);
    font-family: var(--mono);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
